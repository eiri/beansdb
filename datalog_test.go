package dieci

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestDataLog for compliance to Datalogger
func TestDataLog(t *testing.T) {
	assert := require.New(t)
	name := randomName()
	err := createDatalogFile(name)
	assert.NoError(err)

	words := "The quick brown fox jumps over the lazy dog"

	t.Run("open", func(t *testing.T) {
		missing := randomName()
		dl := NewDatalog(missing)
		err := dl.Open()
		assert.Error(err)
		dl = NewDatalog(name)
		err = dl.Open()
		assert.NoError(err)
		defer dl.Close()
		assert.Equal(0, dl.cur, "Cursor should be on 0")
	})

	t.Run("write", func(t *testing.T) {
		dl := NewDatalog(name)
		err := dl.Open()
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
	})

	t.Run("read", func(t *testing.T) {
		dl := NewDatalog(name)
		err := dl.Open()
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
		os.Remove(fmt.Sprintf("%s.idx", name))
		dl := NewDatalog(name)
		err := dl.Open()
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
		dl := NewDatalog(name)
		err := dl.Open()
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
		idxName := randomName()
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
		if idx.Len() != 235886 {
			b.Fatal("expected index cache to be fully propagated")
		}
		idxF.Close()
		os.Remove(idxName + ".idx")
	}
	dl.Close()
}

func createDatalogFile(name string) error {
	f, err := os.Create(fmt.Sprintf("%s.data", name))
	defer f.Close()
	return err
}

func removeDatalogFile(name string) error {
	os.Remove(fmt.Sprintf("%s.idx", name))
	return os.Remove(fmt.Sprintf("%s.data", name))
}

func randomName() string {
	buf := make([]byte, 16)
	rand.Read(buf)
	return hex.EncodeToString(buf)
}
