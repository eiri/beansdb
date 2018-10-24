package beansdb_test

import (
	"crypto/rand"
	"github.com/eiri/beansdb"
	"os"
	"reflect"
	"testing"
)

type kv struct {
	score beansdb.Score
	data  []byte
}

var kvs []kv
var store string

// TestNew to ensure we can create a new storage
func TestNew(t *testing.T) {
	s, err := beansdb.New()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	store = s.Name()
	_, err = os.Stat(store)
	if err != nil {
		t.Fatal(err)
	}
}

// TestOpen to ensure we can open an existing storage
func TestOpen(t *testing.T) {
	_, err := os.Stat(store)
	if err != nil {
		t.Fatal(err)
	}
	s, err := beansdb.Open(store)
	if err != nil {
		t.Fatal(err)
	}
	s.Close()
}

// BenchmarkOpen for an iterative improvement
func BenchmarkOpen(b *testing.B) {
	for n := 0; n < b.N; n++ {
		s, err := beansdb.Open("testdata/words.data")
		if err != nil {
			b.Fatal(err)
		}
		s.Close()
	}
}

// TestWrite to ensure we can write in the store
func TestWrite(t *testing.T) {
	kvs = make([]kv, 5)
	s, err := beansdb.Open(store)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for i, docSize := range []int{21, 12, 42, 5, 17} {
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
		statBefore, _ := s.Stat()
		score2, err := s.Write(doc)
		if err != nil {
			t.Fatal(err)
		}
		statAfter, _ := s.Stat()
		if score != score2 {
			t.Errorf("Expecting score be the same %s != %s", score, score2)
		}
		if statBefore.Size() != statAfter.Size() {
			t.Errorf("Expecting store size be the same")
		}
	}
}

// BenchmarkWrite for an iterative improvement
func BenchmarkWrite(b *testing.B) {
	s, err := beansdb.New()
	if err != nil {
		b.Fatal(err)
	}
	defer s.Delete()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		b.StopTimer()
		docSize := 1024
		doc := make([]byte, docSize)
		_, err = rand.Read(doc)
		b.StartTimer()
		if err == nil {
			_, err = s.Write(doc)
		} else {
			b.Fatal(err)
		}
	}
}

// TestRead to ensure we can read from the store
func TestRead(t *testing.T) {
	s, err := beansdb.Open(store)
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

// BenchmarkRead for an iterative improvement
func BenchmarkRead(b *testing.B) {
	s, err := beansdb.Open("testdata/words.data")
	if err != nil {
		b.Fatal(err)
	}
	defer s.Close()
	score := s.MakeScore([]byte("witchwork"))
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, err := s.Read(score)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// TestDelete to ensure we can delete the store
func TestDelete(t *testing.T) {
	_, err := os.Stat(store)
	if err != nil {
		t.Fatal(err)
	}
	s, err := beansdb.Open(store)
	if err != nil {
		t.Fatal(err)
	}
	s.Delete()
	_, err = os.Stat(store)
	if !os.IsNotExist(err) {
		t.Error("Expecting store file do not exist")
	}
}
