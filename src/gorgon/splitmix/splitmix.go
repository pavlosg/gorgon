package splitmix

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

func New(seed int64) *SplitMix {
	return &SplitMix{state: splitmixTransform(uint64(seed))}
}

func NewRand() *rand.Rand {
	return rand.New(New(NewSeed()))
}

type SplitMix struct {
	state uint64
}

func (rng *SplitMix) Uint64() uint64 {
	rng.state += splitmixIncrement
	return splitmixTransform(rng.state)
}

func (rng *SplitMix) Int63() int64 {
	return int64(rng.Uint64() & maxInt63)
}

func (rng *SplitMix) Uint64Atomic() uint64 {
	return splitmixTransform(atomic.AddUint64(&rng.state, splitmixIncrement))
}

func (rng *SplitMix) Int63Atomic() int64 {
	return int64(rng.Uint64Atomic() & maxInt63)
}

func (rng *SplitMix) Seed(seed int64) {
	rng.state = splitmixTransform(uint64(seed))
}

func (rng *SplitMix) SeedAtomic(seed int64) {
	atomic.StoreUint64(&rng.state, splitmixTransform(uint64(seed)))
}

func splitmixTransform(z uint64) uint64 {
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31)
}

const splitmixIncrement = 0x9e3779b97f4a7c15

const maxInt63 = (1 << 63) - 1
