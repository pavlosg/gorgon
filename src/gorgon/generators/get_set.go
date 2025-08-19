package generators

import (
	"fmt"
	"math/rand"

	"github.com/pavlosg/gorgon/src/gorgon"
	"github.com/pavlosg/gorgon/src/gorgon/splitmix"
)

type GetInstruction struct {
	Key string
}

type SetInstruction struct {
	Key   string
	Value int
}

func (op *GetInstruction) GetKey() string {
	return op.Key
}

func (op *SetInstruction) GetKey() string {
	return op.Key
}

func (op *GetInstruction) String() string {
	return fmt.Sprintf("Get(%q)", op.Key)
}

func (op *SetInstruction) String() string {
	return fmt.Sprintf("Set(%q, %d)", op.Key, op.Value)
}

func NewGetSetGenerator(keys []string) gorgon.Generator {
	return (&getSetGenerator{keys: keys, rand: splitmix.NewRand()}).getNext
}

type getSetGenerator struct {
	keys []string
	rand *rand.Rand
	id   int
}

func (gen *getSetGenerator) getNext(client int) (gorgon.Instruction, error) {
	gen.id++
	id := gen.id
	key := gen.keys[gen.rand.Intn(len(gen.keys))]
	if client&1 != 0 || gen.rand.Int63()&1 != 0 {
		return &GetInstruction{Key: key}, nil
	}
	return &SetInstruction{Key: key, Value: id}, nil
}
