package random

import (
	crand "crypto/rand" // for seed
	"encoding/binary"
	"math/rand" // for rand.New() and rand.Rand
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
