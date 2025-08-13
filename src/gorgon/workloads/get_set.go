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
	keys := []string{"key0", "key1", "key2", "key3", "key4", "key5", "key6", "key7"}
	w.gen = generators.NewGetSetGenerator(keys)
	w.gen = generators.Stagger(w.gen, time.Millisecond/4)
	return nil
}

func (w *getSetWorkload) TearDown() error {
	return nil
}
