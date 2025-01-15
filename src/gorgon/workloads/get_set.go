package workloads

import (
	"time"

	"github.com/pavlosg/gorgon/src/gorgon"
	"github.com/pavlosg/gorgon/src/gorgon/generators"
)

func NewGetSetWorkload() gorgon.Workload {
	return &getSetWorkload{}
}

type getSetWorkload struct {
	gen gorgon.Generator
}

func (*getSetWorkload) Name() string {
	return "GetSet"
}

func (w *getSetWorkload) Generator() gorgon.Generator {
	return w.gen
}

func (w *getSetWorkload) SetUp(opt *gorgon.Options, clients []gorgon.Client) error {
	keys := []string{"key0", "key1", "key2", "key3", "key4"}
	w.gen = generators.NewGetSetGenerator(keys)
	w.gen = generators.Stagger(w.gen, 10*time.Millisecond)
	clearOp := clients[0].Invoke(&gorgon.ClearDatabaseInstruction{}, func() int64 { return 0 })
	if err, ok := clearOp.Output.(error); ok {
		return err
	}
	return nil
}

func (w *getSetWorkload) TearDown() error {
	return nil
}
