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

func (st *stagger) Next(client int) (gorgon.Instruction, error) {
	now := time.Now()
	if now.Before(st.next) {
		return nil, nil
	}
	instr, err := st.gen.Next(client)
	if instr != nil {
		dur := time.Duration(st.rand.Int63n(int64(st.pace * 2)))
		next := st.next.Add(dur)
		if now.Sub(next) > st.pace*8 {
			next = now.Add(dur)
		}
		st.next = next
	}
	return instr, err
}

func (st *stagger) Name() string {
	return st.gen.Name()
}

func (st *stagger) SetUp(opt *gorgon.Options) error {
	return st.gen.SetUp(opt)
}

func (st *stagger) TearDown() error {
	return st.gen.TearDown()
}

func (st *stagger) Invoke(instruction gorgon.Instruction, getTime func() int64) (int64, gorgon.Output) {
	return st.gen.Invoke(instruction, getTime)
}

func (st *stagger) OnCall(client int, instruction gorgon.Instruction) error {
	return st.gen.OnCall(client, instruction)
}

func (st *stagger) OnReturn(client int, instruction gorgon.Instruction, output gorgon.Output) error {
	return st.gen.OnReturn(client, instruction, output)
}
