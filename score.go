package dieci

import (
	"encoding/binary"
	"encoding/hex"

	"github.com/cespare/xxhash"
)

// scoreSize is the size of score in bytes
const scoreSize = 8

// Score is type alias for score representation
type Score [scoreSize]byte

func (s Score) String() string {
	return hex.EncodeToString(s[:])
}

func (s Score) UInt64() uint64 {
	return binary.BigEndian.Uint64(s[:])
}

// MakeScore creates a score for a given data block
func MakeScore(b []byte) Score {
	sum := xxhash.Sum64(b)
	score := Score{}
	binary.BigEndian.PutUint64(score[:], sum)
	return score
}
