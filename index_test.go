package dieci

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestIndex for compliance to Indexer
func TestIndex(t *testing.T) {
	assert := require.New(t)
	name := randomName()
	err := createDatalogFile(name)
	assert.NoError(err)

	words := "The quick brown fox jumps over the lazy dog"
	var index []byte

	t.Run("store", func(t *testing.T) {
		idxRW := bytes.NewBuffer([]byte{})
		idx, err := NewIndex(idxRW)
		assert.NoError(err)
		for pos, word := range strings.Fields(words) {
			data := []byte(word)
			size := len(data)
			score := MakeScore(data)
			expAddr := Addr{pos, size}
			err := idx.Store(score, Addr{pos: pos, size: size})
			assert.NoError(err)
			addr, _ := idx.cache.Load(score)
			assert.Equal(expAddr, addr)
			err = idx.Store(score, Addr{pos: 0, size: 0})
			assert.NoError(err)
			addr, _ = idx.cache.Load(score)
			assert.Equal(expAddr, addr, "Should ignore update")
		}
		index = make([]byte, idxRW.Len())
		copy(index, idxRW.Bytes())
	})

	t.Run("open", func(t *testing.T) {
		tmp := make([]byte, len(index))
		copy(tmp, index)
		idxRW := bytes.NewBuffer(tmp)
		idx, err := NewIndex(idxRW)
		assert.NoError(err)
		assert.Equal(idx.Len(), len(strings.Fields(words)))
	})

	t.Run("load", func(t *testing.T) {
		tmp := make([]byte, len(index))
		copy(tmp, index)
		idxRW := bytes.NewBuffer(tmp)
		idx, err := NewIndex(idxRW)
		assert.NoError(err)
		for pos, word := range strings.Fields(words) {
			data := []byte(word)
			size := len(data)
			score := MakeScore(data)
			a, ok := idx.Load(score)
			assert.Equal(pos, a.pos, "Should return correct position")
			assert.Equal(size, a.size, "Should return correct size")
			assert.True(ok, "Should indicate that score exists")
		}
		score := MakeScore([]byte("missing"))
		a, ok := idx.Load(score)
		assert.Empty(a.pos, "Should return 0 position for missing score")
		assert.Empty(a.size, "Should return 0 size for missing score")
		assert.False(ok, "Should indicate that score doesn't exists")
	})

	// cleanup
	err = removeDatalogFile(name)
	assert.NoError(err)
}

// BenchmarkIndexLoad for iterative improvement of open
func BenchmarkIndexOpen(b *testing.B) {
	for n := 0; n < b.N; n++ {
		b.StopTimer()
		f, err := os.Open("testdata/words.idx")
		if err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
		_, err = NewIndex(f)
		if err != nil {
			b.Fatal(err)
		}
		f.Close()
	}
}
