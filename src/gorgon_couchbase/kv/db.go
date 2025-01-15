package kv

import (
	"fmt"

	"github.com/pavlosg/gorgon/src/gorgon"
	"github.com/pavlosg/gorgon/src/gorgon/workloads"
)

func NewDatabase() gorgon.Database {
	return &database{}
}

type database struct {
	clientId int
	options  *gorgon.Options
}

func (*database) Name() string {
	return "couchbase"
}

func (db *database) SetUp(opt *gorgon.Options) error {
	db.clientId = 0
	db.options = opt
	return nil
}

func (db *database) TearDown() error {
	return nil
}

func (db *database) NewClient() (gorgon.Client, error) {
	id := db.clientId
	db.clientId++
	url := fmt.Sprintf("couchbase://%s:12000", db.options.Nodes[0])
	return NewClient(id, url, "Administrator", "asdasd"), nil
}

func (*database) Scenarios(opt *gorgon.Options) []gorgon.Scenario {
	return []gorgon.Scenario{
		{Workload: workloads.NewGetSetWorkload(), Nemesis: nil},
	}
}
