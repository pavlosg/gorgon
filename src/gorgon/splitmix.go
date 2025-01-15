package gorgon

import (
	crypto_rand "crypto/rand"
	"encoding/binary"
	"math/rand"
	"sync/atomic"
)

func NewSeed() int64 {
	var seed [8]byte
	crypto_rand.Read(seed[:])
	return int64(binary.LittleEndian.Uint64(seed[:]))
}

func NewSplitMixRand() *rand.Rand {
	return rand.New(NewSplitMixRandomSource(NewSeed()))
}

func NewSplitMixRandomSource(seed int64) rand.Source {
	return &splitmix{state: splitmixTransform(uint64(seed))}
}

func NewSplitMixRandomSource64(seed int64) rand.Source64 {
	return &splitmix{state: splitmixTransform(uint64(seed))}
}

const maxInt63 = (1 << 63) - 1

type splitmix struct {
	state uint64
}

func (rng *splitmix) Uint64() uint64 {
	return splitmixTransform(atomic.AddUint64(&rng.state, 0x9e3779b97f4a7c15))
}

func (rng *splitmix) Int63() int64 {
	return int64(rng.Uint64() & maxInt63)
}

func (rng *splitmix) Seed(seed int64) {
	atomic.StoreUint64(&rng.state, splitmixTransform(uint64(seed)))
}

func splitmixTransform(z uint64) uint64 {
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31)
}
