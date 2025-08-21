package workloads

import (
	"time"

	"github.com/pavlosg/gorgon/src/gorgon"
	"github.com/pavlosg/gorgon/src/gorgon/generators"
)

func GetSetWorkload() gorgon.Workload {
	keys := []string{"key0", "key1", "key2", "key3", "key4", "key5", "key6", "key7"}
	return gorgon.Workload{
		Model:      GetSetModel(),
		Generators: []gorgon.Generator{generators.Stagger(generators.NewGetSetGenerator(keys), time.Millisecond)},
	}
}

func GetSetModel() gorgon.Model {
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
			case *generators.GetInstruction:
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
			case *generators.SetInstruction:
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
