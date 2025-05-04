package random

import (
	crand "crypto/rand" // für sicheren Seed
	"encoding/binary"
	"math/rand" // für rand.New() und rand.Rand
)

func NewSafeRand() *rand.Rand {
	var b [8]byte
	_, err := crand.Read(b[:])
	if err != nil {
		panic(err)
	}
	seed := int64(binary.LittleEndian.Uint64(b[:]))
	return rand.New(rand.NewSource(seed))
}
