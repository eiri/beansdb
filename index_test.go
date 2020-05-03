package dieci

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestIndex for compliance to Indexer
func TestIndex(t *testing.T) {
	assert := require.New(t)

	var idxtests = []struct {
		in   string
		addr Addr
		ok   bool
	}{
		{"the", Addr{pos: 0, size: 15}, true},
		{"quick", Addr{pos: 15, size: 17}, true},
		{"brown", Addr{pos: 32, size: 17}, true},
		{"fox", Addr{pos: 49, size: 15}, true},
		{"jumps", Addr{pos: 64, size: 17}, true},
		{"over", Addr{pos: 81, size: 16}, true},
		{"missing", Addr{pos: 0, size: 0}, false},
		{"the", Addr{pos: 0, size: 15}, true},
		{"lazy", Addr{pos: 97, size: 16}, true},
		{"dog", Addr{pos: 113, size: 15}, true},
	}

	t.Run("open empty", func(t *testing.T) {
		reader := bytes.NewReader([]byte{})
		idx, err := NewIndex(reader)
		assert.NoError(err)
		assert.Equal(0, idx.cache.Len())
		assert.Equal(0, idx.cur)
	})

	t.Run("write", func(t *testing.T) {
		reader := bytes.NewReader([]byte{})
		idx, err := NewIndex(reader)
		assert.NoError(err)
		for _, tt := range idxtests {
			if !tt.ok {
				continue
			}
			data := []byte(tt.in)
			size := len(data)
			score := MakeScore(data)
			before := idx.Len()
			idx.Write(score, size)
			assert.GreaterOrEqual(idx.Len(), before)
			before = idx.Len()
			idx.Write(score, 0)
			assert.Equal(idx.Len(), before, "Should ignore same update")
		}
	})

	t.Run("open", func(t *testing.T) {
		reader := readerFromDatalog()
		idx, err := NewIndex(reader)
		assert.NoError(err)
		assert.Equal(8, idx.cache.Len())
		assert.Equal(int(reader.Size()), idx.cur)
	})

	t.Run("read", func(t *testing.T) {
		reader := readerFromDatalog()
		idx, err := NewIndex(reader)
		assert.NoError(err)
		for _, tt := range idxtests {
			score := MakeScore([]byte(tt.in))
			addr, ok := idx.Read(score)
			if tt.ok {
				assert.True(ok, "Should indicate that score exists")
			} else {
				assert.False(ok, "Should indicate that score is missing")
			}
			assert.Equal(tt.addr, addr, "Should return correct address")
		}
	})
}

// BenchmarkOpenIndex for iterative improvement of open
func BenchmarkOpenIndex(b *testing.B) {
	for n := 0; n < b.N; n++ {
		b.StopTimer()
		f, err := os.Open("testdata/words.data")
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

// FIXME! - this is a hack, I'm obviously need something better here
func readerFromDatalog() *bytes.Reader {
	dr := bytes.NewReader([]byte{})
	dw := bytes.NewBuffer([]byte{})
	idx, _ := NewIndex(dr)
	dl := NewDatalog(dr, dw, idx)
	for _, word := range []string{"the", "quick", "brown", "fox", "jumps", "over", "lazy", "dog"} {
		data := []byte(word)
		dl.Put(data)
	}
	return bytes.NewReader(dw.Bytes())
}
