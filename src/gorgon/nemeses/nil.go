package nemeses

import (
	"time"

	"github.com/pavlosg/gorgon/src/gorgon"
)

type NilNemesis struct {
	duration time.Duration
}

func (*NilNemesis) Name() string {
	return "nil"
}

func (nemesis *NilNemesis) SetUp(opt *gorgon.Options) error {
	nemesis.duration = opt.WorkloadDuration
	return nil
}

func (*NilNemesis) TearDown() error {
	return nil
}

func (nemesis *NilNemesis) Run() error {
	time.Sleep(nemesis.duration)
	return nil
}
