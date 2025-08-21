package kv

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/pavlosg/gorgon/src/gorgon"
	"github.com/pavlosg/gorgon/src/gorgon/generators"
)

func NewClient(id int, url, user, pass string) gorgon.Client {
	return &client{id: id, url: url, user: user, pass: pass}
}

type ClientConfig struct {
	Durability string
	Timeout    time.Duration
}

type client struct {
	id         int
	url        string
	user       string
	pass       string
	config     ClientConfig
	durability gocb.DurabilityLevel
	cluster    *gocb.Cluster
	collection *gocb.Collection
}

var errNilCollection = errors.New("nil collection")

func (client *client) Id() int {
	return client.id
}

func (client *client) Open(config string) error {
	if err := json.Unmarshal([]byte(config), &client.config); err != nil {
		return err
	}
	if len(client.config.Durability) != 0 {
		client.durability = parseDurabilityLevel(client.config.Durability)
		if client.durability == gocb.DurabilityLevelUnknown {
			return errors.New("kv: invalid durability level in config")
		}
	}
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

func (client *client) Invoke(instruction gorgon.Instruction, getTime func() int64) (retTime int64, output gorgon.Output) {
	switch instr := instruction.(type) {
	case *generators.GetInstruction:
		result, err := client.collection.Get(instr.Key, &gocb.GetOptions{Timeout: client.config.Timeout})
		retTime = getTime()
		if err != nil {
			if errors.Is(err, gocb.ErrDocumentNotFound) {
				output = nil
			} else {
				// Get is idempotent
				output = gorgon.WrapUnambiguousError(err)
			}
			return
		}
		val := 0
		err = result.Content(&val)
		if err != nil {
			output = gorgon.WrapUnambiguousError(err)
		} else {
			output = val
		}
		return
	case *generators.SetInstruction:
		_, err := client.collection.Upsert(instr.Key, instr.Value,
			&gocb.UpsertOptions{DurabilityLevel: client.durability, Timeout: client.config.Timeout})
		retTime = getTime()
		if err != nil {
			if errors.Is(err, gocb.ErrUnambiguousTimeout) ||
				errors.Is(err, gocb.ErrDurabilityImpossible) {
				output = gorgon.WrapUnambiguousError(err)
			} else {
				output = err
			}
		}
		return
	}
	return getTime(), gorgon.ErrUnsupportedInstruction
}

func parseDurabilityLevel(level string) gocb.DurabilityLevel {
	switch level {
	case "none":
		return gocb.DurabilityLevelNone
	case "majority":
		return gocb.DurabilityLevelMajority
	case "majorityPersistActive":
		return gocb.DurabilityLevelMajorityAndPersistOnMaster
	case "persistMajority":
		return gocb.DurabilityLevelPersistToMajority
	default:
		return gocb.DurabilityLevelUnknown
	}
}
