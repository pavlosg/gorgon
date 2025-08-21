package kv

import (
	"math/rand"
	"time"

	"github.com/pavlosg/gorgon/src/gorgon"
	"github.com/pavlosg/gorgon/src/gorgon/generators"
	"github.com/pavlosg/gorgon/src/gorgon/nemeses"
	"github.com/pavlosg/gorgon/src/gorgon/splitmix"
)

func NewPartitionAwareGetSetGenerator() gorgon.Generator {
	return generators.Stagger(&partitionAwareGenerator{
		keys: []string{"key0", "key1", "key2", "key3", "key4", "key5", "key6", "key7"},
		rand: splitmix.NewRand()}, 10*time.Millisecond)
}

type partitionAwareGenerator struct {
	keys     []string
	rand     *rand.Rand
	numNodes int
	node     int
	val      int
	start    time.Time
}

func (gen *partitionAwareGenerator) Next(client int) (gorgon.Instruction, error) {
	if client < 0 || gen.numNodes == 0 || gen.node < 0 || time.Until(gen.start) > 0 {
		return nil, nil
	}
	key := gen.keys[gen.rand.Intn(len(gen.keys))]
	vb := getVbid([]byte(key), 1024)
	if vb < gen.node*1024/gen.numNodes || vb >= (gen.node+1)*1024/gen.numNodes {
		return nil, nil
	}
	if client%gen.numNodes == gen.node {
		return &generators.GetInstruction{Key: key}, nil
	}
	gen.val--
	return &generators.SetInstruction{Key: key, Value: gen.val}, nil
}

func (*partitionAwareGenerator) Name() string {
	return "PartitionAwareGetSet"
}

func (gen *partitionAwareGenerator) SetUp(opt *gorgon.Options) error {
	gen.numNodes = len(opt.Nodes)
	gen.node = -1
	return nil
}

func (*partitionAwareGenerator) TearDown() error {
	return nil
}

func (*partitionAwareGenerator) Invoke(instruction gorgon.Instruction, getTime func() int64) (int64, gorgon.Output) {
	return -1, gorgon.ErrUnsupportedInstruction
}

func (*partitionAwareGenerator) OnCall(client int, instruction gorgon.Instruction) error {
	return nil
}

func (gen *partitionAwareGenerator) OnReturn(client int, instruction gorgon.Instruction, output gorgon.Output) error {
	if instr, ok := instruction.(*nemeses.PartitionNodeInstruction); ok {
		if instr.Heal {
			gen.node = -1
		} else {
			gen.node = instr.Node
			gen.start = time.Now().Add(20 * time.Second)
		}
	}
	return nil
}
