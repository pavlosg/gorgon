package generators

import (
	"sort"

	"github.com/pavlosg/gorgon/src/gorgon"
)

func PartitionByKey(history []gorgon.Operation) (ret [][]gorgon.Operation) {
	operations := make(map[string][]gorgon.Operation)
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
		ops []gorgon.Operation
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
