package dieci

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestDataLog for compliance to Datalogger
func TestDataLog(t *testing.T) {
	assert := require.New(t)
	name := RandomName()
	err := CreateDatalogFile(name)
	assert.NoError(err)

	words := "The quick brown fox jumps over the lazy dog"
	var index []byte

	t.Run("open", func(t *testing.T) {
		missing := RandomName()
		irw := bytes.NewBuffer([]byte{})
		dl, err := NewDatalog(missing, irw)
		assert.NoError(err)
		err = dl.Open()
		assert.Error(err)
		dl, err = NewDatalog(name, irw)
		assert.NoError(err)
		err = dl.Open()
		assert.NoError(err)
		defer dl.Close()
		assert.Equal(0, dl.cur, "Cursor should be on 0")
	})

	t.Run("write", func(t *testing.T) {
		irw := bytes.NewBuffer([]byte{})
		dl, err := NewDatalog(name, irw)
		assert.NoError(err)
		err = dl.Open()
		assert.NoError(err)
		defer dl.Close()
		prevCur := dl.cur
		for _, word := range strings.Fields(words) {
			data := []byte(word)
			expectedScore := MakeScore(data)
			score, err := dl.Write(data)
			assert.NoError(err)
			assert.Equal(expectedScore, score)
			assert.True(dl.cur > prevCur, "Cursor should move")
			prevCur = dl.cur
		}
		index = make([]byte, irw.Len())
		copy(index, irw.Bytes())
	})

	t.Run("read", func(t *testing.T) {
		tmp := make([]byte, len(index))
		copy(tmp, index)
		irw := bytes.NewBuffer(tmp)
		dl, err := NewDatalog(name, irw)
		assert.NoError(err)
		err = dl.Open()
		assert.NoError(err)
		defer dl.Close()
		stat, err := dl.rwc.Stat()
		assert.NoError(err)
		end := int(stat.Size())
		assert.EqualValues(end, dl.cur, "Cursor should be at EOF")
		for _, word := range strings.Fields(words) {
			expectedData := []byte(word)
			score := MakeScore(expectedData)
			data, err := dl.Read(score)
			assert.NoError(err)
			assert.Equal(expectedData, data)
		}
	})

	t.Run("rebuild index", func(t *testing.T) {
		irw := bytes.NewBuffer([]byte{})
		dl, err := NewDatalog(name, irw)
		assert.NoError(err)
		err = dl.Open()
		assert.NoError(err)
		defer dl.Close()
		stat, err := dl.rwc.Stat()
		assert.NoError(err)
		end := int(stat.Size())
		assert.EqualValues(end, dl.cur, "Cursor should be at EOF")
		for _, word := range strings.Fields(words) {
			expectedData := []byte(word)
			score := MakeScore(expectedData)
			data, err := dl.Read(score)
			assert.NoError(err)
			assert.Equal(expectedData, data)
		}
	})

	t.Run("close", func(t *testing.T) {
		irw := bytes.NewBuffer([]byte{})
		dl, err := NewDatalog(name, irw)
		assert.NoError(err)
		err = dl.Open()
		assert.NoError(err)
		defer dl.Close()
		stat, err := dl.rwc.Stat()
		assert.NoError(err)
		end := int(stat.Size())
		assert.Equal(end, dl.cur, "Cursor should be at EOF")
		err = dl.Close()
		assert.NoError(err)
		assert.Equal(0, dl.cur, "Cursor should reset")
		err = dl.Close()
		assert.Error(err, "Should return error on attempt to close again")
	})

	err = removeDatalogFile(name)
	assert.NoError(err)
}

// BenchmarkRebuildIndex isolated
func BenchmarkRebuildIndex(b *testing.B) {
	// open data file
	name := "testdata/words"
	f, err := os.Open(name + ".data")
	if err != nil {
		b.Fatal(err)
	}
	stat, err := f.Stat()
	if err != nil {
		b.Fatal(err)
	}
	dl := &Datalog{name: name, cur: int(stat.Size()), rwc: f}
	for n := 0; n < b.N; n++ {
		// create an empty index and set it to datalog
		idxName := RandomName()
		idxF, err := os.Create(idxName + ".idx")
		if err != nil {
			b.Fatal(err)
		}
		idx, err := NewIndex(idxF)
		if err != nil {
			b.Fatal(err)
		}
		dl.index = idx
		// isolated test
		b.ResetTimer()
		err = dl.RebuildIndex()
		if err != nil {
			b.Fatal(err)
		}
		b.StopTimer()
		if len(idx.cache) != 235886 {
			b.Fatal("expected index cache to be fully propagated")
		}
		idxF.Close()
		os.Remove(idxName + ".idx")
	}
	dl.Close()
}

func removeDatalogFile(name string) error {
	return os.Remove(name + ".data")
}
