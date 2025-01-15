package generators

import (
	"sync"

	"github.com/pavlosg/gorgon/src/gorgon"
)

func Synchronize(gen gorgon.Generator) gorgon.Generator {
	return &synchronize{gen: gen}
}

type synchronize struct {
	gen   gorgon.Generator
	mutex sync.Mutex
}

func (syn *synchronize) NextInstruction() (gorgon.Instruction, error) {
	syn.mutex.Lock()
	instr, err := syn.gen.NextInstruction()
	syn.mutex.Unlock()
	return instr, err
}

func (syn *synchronize) Model() gorgon.Model {
	return syn.gen.Model()
}
