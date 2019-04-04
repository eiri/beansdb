package dieci

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"sync"
)

const (
	intSize       = 4
	doubleIntSize = 8
	bufSize       = 24
	pageSize      = 4080 // ~4K to match memory page
)

// Indexer is the interface for Datalog's index
type Indexer interface {
	Load(score Score) (Addr, bool)
	Store(score Score, a Addr) error
}

// Addr is data position and size in datalog
type Addr struct {
	pos  int
	size int
}

// Index represents an index of a datalog file
type Index struct {
	cache  *sync.Map
	length int
	rw     io.ReadWriter
}

// NewIndex returns a new index structure with the given name
func NewIndex(rw io.ReadWriter) (*Index, error) {
	cache := new(sync.Map)
	length := 0
	for {
		page := make([]byte, pageSize, pageSize)
		n, err := rw.Read(page)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if err == io.EOF {
			break
		}
		br := bytes.NewReader(page[:n])
		for {
			var score Score
			buf := make([]byte, bufSize, bufSize)
			_, err := br.Read(buf)
			if err != nil && err != io.EOF {
				return nil, err
			}
			if err == io.EOF {
				break
			}
			pos := binary.BigEndian.Uint32(buf[0:intSize])
			size := binary.BigEndian.Uint32(buf[intSize:doubleIntSize])
			copy(score[:], buf[doubleIntSize:bufSize])
			cache.Store(score, Addr{pos: int(pos), size: int(size)})
			length++
		}
	}
	return &Index{cache: cache, length: length, rw: rw}, nil
}

// Load returns the address stored in the index for a score
// or an empty Addr if no address is present.
// The ok result indicates if address was found in the index.
func (idx *Index) Load(score Score) (Addr, bool) {
	if a, ok := idx.cache.Load(score); ok {
		return a.(Addr), true
	}
	return Addr{}, false
}

// Store sets the address for a given score.
func (idx *Index) Store(score Score, a Addr) error {
	if _, ok := idx.cache.Load(score); ok {
		return nil
	}
	buf := make([]byte, bufSize, bufSize)
	binary.BigEndian.PutUint32(buf[0:intSize], uint32(a.pos))
	binary.BigEndian.PutUint32(buf[intSize:doubleIntSize], uint32(a.size))
	copy(buf[doubleIntSize:bufSize], score[:])
	_, err := idx.rw.Write(buf)
	if err != nil {
		return fmt.Errorf("index write failed: %s", err)
	}
	idx.cache.Store(score, a)
	idx.length++
	return nil
}

// Len returns the number of scores in the index.
func (idx *Index) Len() int {
	return idx.length
	// length := 0
	// idx.cache.Range(func(_, _ interface{}) bool {
	// 	length++
	// 	return true
	// })
	// return length
}
