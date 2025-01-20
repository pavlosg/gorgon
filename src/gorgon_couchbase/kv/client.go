package kv

import (
	"errors"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/pavlosg/gorgon/src/gorgon"
	"github.com/pavlosg/gorgon/src/gorgon/generators"
)

func NewClient(id int, url, user, pass string, durability gocb.DurabilityLevel) gorgon.Client {
	return &client{id: id, url: url, user: user, pass: pass, durability: durability}
}

type client struct {
	id         int
	url        string
	user       string
	pass       string
	durability gocb.DurabilityLevel
	cluster    *gocb.Cluster
	collection *gocb.Collection
}

var errNilCollection = errors.New("nil collection")

func (client *client) GetId() int {
	return client.id
}

func (client *client) Open() error {
	cluster, err := gocb.Connect(client.url, gocb.ClusterOptions{
		Username: client.user,
		Password: client.pass,
	})
	if err != nil {
		return err
	}
	defer func() {
		if client.cluster == nil {
			cluster.Close(nil)
		}
	}()
	bucket := cluster.Bucket("default")
	err = bucket.WaitUntilReady(5*time.Second, nil)
	if err != nil {
		return err
	}
	client.cluster = cluster
	client.collection = bucket.DefaultCollection()
	if client.collection == nil {
		return errNilCollection
	}
	return nil
}

func (client *client) Close() error {
	err := client.cluster.Close(nil)
	client.cluster = nil
	client.collection = nil
	return err
}

func (client *client) Invoke(instruction gorgon.Instruction, getTime func() int64) gorgon.Operation {
	op := gorgon.Operation{ClientId: client.id, Input: instruction, Call: getTime()}
	switch instr := instruction.(type) {
	case *generators.GetInstruction:
		result, err := client.collection.Get(instr.Key, nil)
		op.Return = getTime()
		if err != nil {
			if !errors.Is(err, gocb.ErrDocumentNotFound) {
				op.Output = err
				return op
			}
		} else {
			val := 0
			err = result.Content(&val)
			if err != nil {
				op.Output = err
				return op
			}
			op.Output = val
		}
		return op
	case *generators.SetInstruction:
		_, err := client.collection.Upsert(instr.Key, instr.Value, &gocb.UpsertOptions{DurabilityLevel: client.durability})
		op.Return = getTime()
		if err != nil {
			op.Output = err
			return op
		}
		return op
	case *gorgon.ClearDatabaseInstruction:
		op.Output = client.cluster.Buckets().FlushBucket("default", nil)
		time.Sleep(5 * time.Second)
		op.Return = getTime()
		return op
	}
	op.Return = getTime()
	op.Output = gorgon.ErrUnsupportedInstruction
	return op
}
