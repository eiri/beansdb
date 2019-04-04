package dieci

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// Datalogger is the interface for Datalog
type Datalogger interface {
	Open() error
	Read(score Score) (data []byte, err error)
	Write(data []byte) (score Score, err error)
	Close() error
}

// Datalog represents a datalog file
type Datalog struct {
	name    string
	index   Indexer
	indexRW *os.File
	cur     int
	rwc     *os.File
}

// NewDatalog returns a new datalog with the given name
func NewDatalog(name string) *Datalog {
	return &Datalog{name: name, cur: 0}
}

// Open opens the named datalog
func (d *Datalog) Open() error {
	fileName := fmt.Sprintf("%s.data", d.name)
	rwc, err := os.OpenFile(fileName, os.O_APPEND|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	stat, err := rwc.Stat()
	if err != nil {
		return err
	}
	d.rwc = rwc
	d.cur = int(stat.Size())

	fileName = fmt.Sprintf("%s.idx", d.name)
	indexRW, err := os.OpenFile(fileName, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	idx, err := NewIndex(indexRW)
	if err != nil {
		return err
	}
	d.index = idx
	d.indexRW = indexRW
	if idx.Len() == 0 {
		err = d.RebuildIndex()
	}
	return err
}

// RebuildIndex by scaning datalog and writing cache again
func (d *Datalog) RebuildIndex() error {
	var err error
	d.rwc.Seek(0, 0)
	pos := intSize
	buf := make([]byte, intSize+scoreSize)
	for {
		if _, err = d.rwc.Read(buf); err == io.EOF {
			err = nil
			break
		}
		size := int(binary.BigEndian.Uint32(buf[:intSize]))
		var score Score
		copy(score[:], buf[intSize:])
		err = d.index.Store(score, Addr{pos: pos, size: size})
		if err != nil {
			break
		}
		pos += size + intSize
		d.rwc.Seek(int64(size-scoreSize), 1)
	}
	return err
}

// Read reads data for a given position and length
func (d *Datalog) Read(score Score) ([]byte, error) {
	a, ok := d.index.Load(score)
	if !ok {
		err := fmt.Errorf("Unknown score %s", score)
		return nil, err
	}
	data := make([]byte, a.size-scoreSize)
	n, err := d.rwc.ReadAt(data, int64(a.pos+scoreSize))
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
	if _, ok := d.index.Load(score); ok {
		return score, nil
	}
	size := len(data) + scoreSize
	buf := make([]byte, intSize+size)
	binary.BigEndian.PutUint32(buf, uint32(size))
	copy(buf[intSize:], score[:])
	copy(buf[intSize+scoreSize:], data)
	n, err := d.rwc.Write(buf)
	if err != nil {
		return Score{}, err
	}
	pos := int(d.cur) + intSize
	size = n - intSize
	d.cur += n
	err = d.index.Store(score, Addr{pos: pos, size: size})
	if err != nil {
		return Score{}, err
	}
	return score, nil
}

// Close closes the datalog
func (d *Datalog) Close() error {
	d.indexRW.Close()
	d.index = new(Index)
	d.cur = 0
	return d.rwc.Close()
}
