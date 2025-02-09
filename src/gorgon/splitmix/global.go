package splitmix

import "math/rand"

var Rand *rand.Rand

type globalSource struct {
	source *SplitMix
}

func (source *globalSource) Uint64() uint64 {
	return source.source.Uint64Atomic()
}

func (source *globalSource) Int63() int64 {
	return source.source.Int63Atomic()
}

func (source *globalSource) Seed(seed int64) {
	source.source.SeedAtomic(seed)
}

func init() {
	Rand = rand.New(&globalSource{New(NewSeed())})
}
