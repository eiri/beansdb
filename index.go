package dieci

import (
	"bufio"
	"encoding/binary"
	"io"
)

const (
	blockSize = 16 // 4 + 4 + 8
)

// Addr is data position and size in datalog
type Addr struct {
	pos  int
	size int
}

// cache is in memory lookup store
type cache map[Score]Addr

// Index represents an index of a datalog file
type Index struct {
	cache cache
	cur   int
}

// NewIndex returns a new index structure with the given name
func NewIndex(reader io.Reader) (*Index, error) {
	cache := make(cache)
	idx := &Index{cache: cache}
	blocks, err := scanReader(reader)
	if err != nil {
		return &Index{}, err
	}

	for block := range blocks {
		size := int(binary.BigEndian.Uint32(block[:intSize]))
		var score Score
		copy(score[:], block[intSize:])
		idx.Write(score, size+4)
	}
	return idx, nil
}

func scanReader(reader io.Reader) (<-chan []byte, error) {
	scanner := bufio.NewScanner(reader)
	blockSize := intSize + scoreSize
	scanner.Split(func(data []byte, eof bool) (int, []byte, error) {
		if eof {
			return 0, nil, io.EOF
		}
		if len(data) < blockSize {
			return 0, nil, nil
		}
		advance := intSize + int(binary.BigEndian.Uint32(data[:intSize]))
		if len(data) < advance {
			return 0, nil, nil
		}
		return advance, data, nil
	})
	blocks := make(chan []byte)
	go func() {
		defer close(blocks)
		for scanner.Scan() {
			block := make([]byte, blockSize)
			copy(block, scanner.Bytes())
			blocks <- block
		}
	}()
	return blocks, scanner.Err()
}

// Read reads address of data for a given score
func (idx *Index) Read(score Score) (a Addr, ok bool) {
	a, ok = idx.cache[score]
	return
}

// Write writes given score into index file and adds it to the cache
func (idx *Index) Write(score Score, size int) {
	if _, ok := idx.cache[score]; !ok {
		addr := Addr{pos: idx.cur, size: size}
		idx.cache[score] = addr
		idx.cur = addr.pos + addr.size
	}
}

// Len returns current length of cache
func (idx *Index) Len() int {
	return len(idx.cache)
}
