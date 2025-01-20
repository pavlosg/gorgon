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
	return &getSetGenerator{keys: keys, rand: splitmix.NewRand()}
}

type getSetGenerator struct {
	keys []string
	rand *rand.Rand
	id   int
}

func (gen *getSetGenerator) NextInstruction() (gorgon.Instruction, error) {
	gen.id++
	id := gen.id
	key := gen.keys[gen.rand.Intn(len(gen.keys))]
	if gen.rand.Int63()&3 != 0 {
		return &GetInstruction{Key: key}, nil
	}
	return &SetInstruction{Key: key, Value: id}, nil
}

func (*getSetGenerator) Model() gorgon.Model {
	return gorgon.Model{
		Init: func() []gorgon.State { return []gorgon.State{IntMap{}} },
		Equal: func(s1, s2 gorgon.State) bool {
			return s1.(IntMap).Equals(s2.(IntMap))
		},
		DescribeState: func(state gorgon.State) string {
			return state.(IntMap).String()
		},
		DescribeOperation: DescribeOperation,
		Partition:         PartitionByKey,
		Step: func(state gorgon.State, input gorgon.Instruction, output interface{}) []gorgon.State {
			stateMap := state.(IntMap)
			switch instr := input.(type) {
			case *GetInstruction:
				if _, ok := output.(error); ok {
					return []gorgon.State{state}
				}
				if val, ok := stateMap.Get(instr.Key); ok {
					if i, ok := output.(int); ok && val == i {
						return []gorgon.State{state}
					}
					return nil
				}
				if output == nil {
					return []gorgon.State{state}
				}
				return nil
			case *SetInstruction:
				stateMap = stateMap.Put(instr.Key, instr.Value)
				if output != nil {
					if _, ok := output.(error); ok {
						return []gorgon.State{state, stateMap}
					}
					return nil
				}
				return []gorgon.State{stateMap}
			}
			return nil
		},
	}
}
