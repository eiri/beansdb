package dieci_test

import (
	"crypto/rand"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/eiri/dieci"
)

type kv struct {
	score dieci.Score
	data  []byte
}

var kvs []kv
var storeName string

// TestNew to ensure we can create a new storage
func TestNew(t *testing.T) {
	s, err := dieci.New()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	storeName = s.Name()
	_, err = os.Stat(fmt.Sprintf("%s.data", storeName))
	if err != nil {
		t.Fatal(err)
	}
}

// TestOpen to ensure we can open an existing storage
func TestOpen(t *testing.T) {
	s, err := dieci.Open(storeName)
	if err != nil {
		t.Fatal(err)
	}
	s.Close()
}

// BenchmarkOpen for iterative improvement of open
func BenchmarkOpen(b *testing.B) {
	for n := 0; n < b.N; n++ {
		s, err := dieci.Open("testdata/words")
		if err != nil {
			b.Fatal(err)
		}
		s.Close()
	}
}

// TestWrite to ensure we can write in the store
func TestWrite(t *testing.T) {
	kvs = make([]kv, 5)
	s, err := dieci.Open(storeName)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	dataFileName := fmt.Sprintf("%s.data", storeName)
	for i, docSize := range []int{2100, 1200, 4200, 500, 1700} {
		doc := make([]byte, docSize)
		_, err = rand.Read(doc)
		if err != nil {
			t.Fatal(err)
		}
		score, err := s.Write(doc)
		if err != nil {
			t.Fatal(err)
		}
		kvs[i] = kv{score: score, data: doc}
		// test deduplication
		statBefore, _ := os.Stat(dataFileName)
		score2, err := s.Write(doc)
		if err != nil {
			t.Fatal(err)
		}
		statAfter, _ := os.Stat(dataFileName)
		if score != score2 {
			t.Errorf("Expecting score be the same %s != %s", score, score2)
		}
		if statBefore.Size() != statAfter.Size() {
			t.Errorf("Expecting store size be the same")
		}
	}
}

// BenchmarkWrite for iterative improvement or writes
func BenchmarkWrite(b *testing.B) {
	s, err := dieci.New()
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		b.StopTimer()
		docSize := 1024
		doc := make([]byte, docSize)
		_, err = rand.Read(doc)
		if err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
		_, err = s.Write(doc)
		if err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	s.Delete()

}

// TestRead to ensure we can read from the store
func TestRead(t *testing.T) {
	s, err := dieci.Open(storeName)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for _, i := range [5]int{1, 2, 0, 4, 3} {
		kv := kvs[i]
		doc, err := s.Read(kv.score)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(doc, kv.data) {
			t.Error("Expecting store to return stored data")
		}
	}
}

// BenchmarkRead for iterative improvement of reads
func BenchmarkRead(b *testing.B) {
	s, err := dieci.Open("testdata/words")
	if err != nil {
		b.Fatal(err)
	}
	score := dieci.MakeScore([]byte("witchwork"))
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, err := s.Read(score)
		if err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	s.Close()
}

// TestWriteRead to ensure we can read back written
func TestWriteRead(t *testing.T) {
	for i := 0; i < 5; i++ {
		// write doc
		s, err := dieci.Open(storeName)
		if err != nil {
			t.Fatal(err)
		}
		before := make([]byte, 1024)
		rand.Read(before)
		score, err := s.Write(before)
		if err != nil {
			t.Fatal(err)
		}
		after, err := s.Read(score)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(after, before) {
			t.Error("Expecting store to return stored data")
		}
		s.Close()
	}
}

// TestDelete to ensure we can delete the store
func TestDelete(t *testing.T) {
	dataFileName := fmt.Sprintf("%s.data", storeName)
	_, err := os.Stat(dataFileName)
	if err != nil {
		t.Fatal(err)
	}
	s, err := dieci.Open(storeName)
	if err != nil {
		t.Fatal(err)
	}
	s.Delete()
	_, err = os.Stat(dataFileName)
	if !os.IsNotExist(err) {
		t.Error("Expecting store file do not exist")
	}
}
