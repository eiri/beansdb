package dieci

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

const (
	intSize = 4
)

// Datalog represents a datalog file
type Datalog struct {
	name   string
	index  *Index
	reader *os.File
	writer *os.File
}

// NewDatalog returns a new datalog with the given name
func NewDatalog(name string, irw io.ReadWriter) (*Datalog, error) {
	idx, err := NewIndex(irw)
	if err != nil {
		return &Datalog{}, err
	}
	return &Datalog{name: name, index: idx}, nil
}

// Open opens the named datalog
func (d *Datalog) Open() error {
	fileName := fmt.Sprintf("%s.data", d.name)
	writer, err := os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	d.writer = writer
	//
	reader, err := os.OpenFile(fileName, os.O_RDONLY, 0600)
	if err != nil {
		return err
	}
	d.reader = reader
	if d.index.Len() == 0 {
		err = d.RebuildIndex()
	}
	return err
}

// RebuildIndex by scaning datalog and writing cache again
func (d *Datalog) RebuildIndex() error {
	var err error
	offset := 0
	buf := make([]byte, intSize+scoreSize)
	for {
		if _, err = d.reader.ReadAt(buf, int64(offset)); err == io.EOF {
			err = nil
			break
		}
		size := int(binary.BigEndian.Uint32(buf[:intSize]))
		var score Score
		copy(score[:], buf[intSize:])
		err = d.index.Write(score, Addr{pos: offset + intSize, size: size})
		if err != nil {
			break
		}
		offset += intSize + size
	}
	return err
}

// Read reads data for a given position and length
func (d *Datalog) Read(score Score) ([]byte, error) {
	a, ok := d.index.Read(score)
	if !ok {
		err := fmt.Errorf("Unknown score %s", score)
		return nil, err
	}
	data := make([]byte, a.size-scoreSize)
	n, err := d.reader.ReadAt(data, int64(a.pos+scoreSize))
	if err != nil {
		return nil, err
	}
	if n != a.size-scoreSize {
		return nil, fmt.Errorf("Read failed")
	}
	return data, nil
}

// Write writes given data into datalog and returns it's position and length
func (d *Datalog) Write(data []byte) (Score, error) {
	score := MakeScore(data)
	if _, ok := d.index.Read(score); ok {
		return score, nil
	}
	buf := d.Encode(score, data)
	n, err := d.writer.Write(buf)
	if err != nil {
		return Score{}, err
	}
	pos := d.index.Cur() + intSize
	size := n - intSize
	err = d.index.Write(score, Addr{pos: pos, size: size})
	if err != nil {
		return Score{}, err
	}
	return score, nil
}

// Encode data and its score into slice of bytes suitable to write on disk
func (d *Datalog) Encode(score Score, data []byte) []byte {
	size := scoreSize + len(data)
	buf := make([]byte, intSize+size)
	binary.BigEndian.PutUint32(buf, uint32(size))
	copy(buf[intSize:], score[:])
	copy(buf[intSize+scoreSize:], data)
	return buf
}

// Close closes the datalog
func (d *Datalog) Close() error {
	d.index = &Index{}
	d.reader.Close()
	return d.writer.Close()
}
