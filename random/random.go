package random

import (
	crand "crypto/rand" // for seed
	"encoding/binary"
	"math/rand" // for rand.New() and rand.Rand
)

// shared is used when Seed has been called, so a benchmark run is reproducible.
var shared *rand.Rand

// Seed makes every later NewSafeRand call return one shared, deterministic generator.
// Only safe single-threaded, which is what the benchmark uses.
func Seed(seed int64) {
	shared = rand.New(rand.NewSource(seed))
}

func NewSafeRand() *rand.Rand {
	if shared != nil {
		return shared
	}
	var b [8]byte
	_, err := crand.Read(b[:])
	if err != nil {
		panic(err)
	}
	seed := int64(binary.LittleEndian.Uint64(b[:]))
	return rand.New(rand.NewSource(seed))
}
