package kv

import (
	"fmt"

	"github.com/couchbase/gocb/v2"
	"github.com/pavlosg/gorgon/src/gorgon"
	"github.com/pavlosg/gorgon/src/gorgon/workloads"
)

func NewDatabase() gorgon.Database {
	return &database{}
}

type database struct {
	options    *gorgon.Options
	durability gocb.DurabilityLevel
}

func (*database) Name() string {
	return "couchbase"
}

func (db *database) SetUp(opt *gorgon.Options) error {
	db.options = opt
	if durability, ok := opt.Extras["kv_durability"]; ok {
		switch durability {
		case "":
			db.durability = gocb.DurabilityLevelUnknown
		case "none":
			db.durability = gocb.DurabilityLevelNone
		case "majority":
			db.durability = gocb.DurabilityLevelMajority
		case "majority_and_persist_on_master":
			db.durability = gocb.DurabilityLevelMajorityAndPersistOnMaster
		case "persist_to_majority":
			db.durability = gocb.DurabilityLevelPersistToMajority
		default:
			return fmt.Errorf("kv: invalid durability %q", durability)
		}
	}
	return nil
}

func (db *database) TearDown() error {
	return nil
}

func (db *database) NewClient(id int) (gorgon.Client, error) {
	url := fmt.Sprintf("couchbase://%s:12000", db.options.Nodes[0])
	return NewClient(id, url, "Administrator", "asdasd", db.durability), nil
}

func (*database) Scenarios(opt *gorgon.Options) []gorgon.Scenario {
	dur := opt.WorkloadDuration
	return []gorgon.Scenario{
		{Workload: workloads.NewGetSetWorkload(), Nemesis: nil, WorkloadDuration: dur},
		{Workload: workloads.NewGetSetWorkload(), Nemesis: NewKillNemesis("memcached"), WorkloadDuration: dur},
	}
}
