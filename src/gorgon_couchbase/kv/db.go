package kv

import (
	"fmt"
	"strconv"

	"github.com/couchbase/gocb/v2"
	"github.com/pavlosg/gorgon/src/gorgon"
	"github.com/pavlosg/gorgon/src/gorgon/workloads"
)

func NewDatabase() gorgon.Database {
	return &database{}
}

type database struct {
	options    *gorgon.Options
	user       string
	pass       string
	port       int
	durability gocb.DurabilityLevel
}

func (*database) Name() string {
	return "couchbase"
}

func (db *database) SetUp(opt *gorgon.Options) error {
	db.options = opt
	db.user = "Administrator"
	db.pass = "password"
	db.port = 11210
	if user, ok := opt.Extras["db_user"]; ok {
		db.user = user
	}
	if pass, ok := opt.Extras["db_pass"]; ok {
		db.pass = pass
	}
	if port, ok := opt.Extras["db_port"]; ok {
		n, err := strconv.Atoi(port)
		if err != nil {
			return err
		}
		db.port = n
	}
	if durability, ok := opt.Extras["db_durability"]; ok {
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
	url := fmt.Sprintf("couchbase://%s:%d", db.options.Nodes[0], db.port)
	return NewClient(id, url, db.user, db.pass, db.durability), nil
}

func (*database) Scenarios(opt *gorgon.Options) []gorgon.Scenario {
	return []gorgon.Scenario{
		{Workload: workloads.NewGetSetWorkload(), Nemesis: nil},
		{Workload: workloads.NewGetSetWorkload(), Nemesis: NewKillNemesis("memcached")},
	}
}
