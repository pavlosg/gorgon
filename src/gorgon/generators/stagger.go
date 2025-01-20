package generators

import (
	"math/rand"
	"time"

	"github.com/pavlosg/gorgon/src/gorgon"
	"github.com/pavlosg/gorgon/src/gorgon/splitmix"
)

func Stagger(gen gorgon.Generator, pace time.Duration) gorgon.Generator {
	return &stagger{gen, pace, time.Now(), splitmix.NewRand()}
}

type stagger struct {
	gen  gorgon.Generator
	pace time.Duration
	next time.Time
	rand *rand.Rand
}

func (st *stagger) NextInstruction() (gorgon.Instruction, error) {
	if time.Now().Before(st.next) {
		return gorgon.InstructionPending{}, nil
	}
	instr, err := st.gen.NextInstruction()
	if err == nil {
		dur := time.Duration(st.rand.Int63n(int64(st.pace * 2)))
		st.next = st.next.Add(dur)
	}
	return instr, err
}

func (st *stagger) Model() gorgon.Model {
	return st.gen.Model()
}
