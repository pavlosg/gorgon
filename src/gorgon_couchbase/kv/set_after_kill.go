package kv

import (
	"github.com/pavlosg/gorgon/src/gorgon"
	"github.com/pavlosg/gorgon/src/gorgon/generators"
	"github.com/pavlosg/gorgon/src/gorgon/rpcs"
)

func NewSetAfterKillGenerator() gorgon.Generator {
	return &setAfterKillGenerator{
		keys: []string{"key0", "key1", "key2", "key3", "key4", "key5", "key6", "key7"}}
}

type setAfterKillGenerator struct {
	keys   []string
	client int
	key    int
	val    int
}

func (gen *setAfterKillGenerator) Next(client int) (gorgon.Instruction, error) {
	if gen.client != client || client < 0 {
		return nil, nil
	}
	if gen.key >= len(gen.keys) {
		gen.key = 0
		return nil, nil
	}
	key := gen.keys[gen.key]
	gen.key++
	gen.val--
	return &generators.SetInstruction{Key: key, Value: gen.val}, nil
}

func (*setAfterKillGenerator) Name() string {
	return "SetAfterKill"
}

func (gen *setAfterKillGenerator) SetUp(opt *gorgon.Options) error {
	gen.client = -1
	return nil
}

func (*setAfterKillGenerator) TearDown() error {
	return nil
}

func (*setAfterKillGenerator) Invoke(instruction gorgon.Instruction, getTime func() int64) (int64, gorgon.Output) {
	return -1, gorgon.ErrUnsupportedInstruction
}

func (gen *setAfterKillGenerator) OnCall(client int, instruction gorgon.Instruction) error {
	if _, ok := instruction.(*rpcs.KillInstruction); ok && gen.client < 0 {
		gen.client = 1
	}
	return nil
}

func (gen *setAfterKillGenerator) OnReturn(client int, instruction gorgon.Instruction, output gorgon.Output) error {
	if _, ok := instruction.(*generators.SetInstruction); !ok {
		return nil
	}
	if err, ok := output.(error); ok && !gorgon.IsUnambiguousError(err) {
		if gen.client < 0 {
			gen.client = 1
		} else if gen.client == client {
			gen.client += 2
		}
	}
	return nil
}
