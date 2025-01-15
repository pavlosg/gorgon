package generators

import (
	"sort"

	"github.com/anishathalye/porcupine"
)

func PartitionByKey(history []porcupine.Operation) (ret [][]porcupine.Operation) {
	operations := make(map[string][]porcupine.Operation)
	for _, op := range history {
		instr, ok := op.Input.(interface{ GetKey() string })
		if !ok {
			continue
		}
		key := instr.GetKey()
		operations[key] = append(operations[key], op)
	}
	type keyOps struct {
		key string
		ops []porcupine.Operation
	}
	var list []keyOps
	for key, ops := range operations {
		list = append(list, keyOps{key, ops})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].key < list[j].key })
	for _, ops := range list {
		ret = append(ret, ops.ops)
	}
	return
}
