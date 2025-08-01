package kv

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/pavlosg/gorgon/src/gorgon"
	"github.com/pavlosg/gorgon/src/gorgon/log"
	"github.com/pavlosg/gorgon/src/gorgon/nemeses"
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
	replicas   int
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
	if replicas, ok := opt.Extras["db_replicas"]; ok {
		n, err := strconv.Atoi(replicas)
		if err != nil {
			return err
		}
		if n < 0 || n > 3 {
			return fmt.Errorf("kv: invalid number of replicas %d", n)
		}
		db.replicas = n
	} else {
		db.replicas = 1
	}

	for _, node := range opt.Nodes {
		if err := db.httpPost(node, "controller/hardResetNode", nil); err != nil {
			return err
		}
		if err := db.httpPost(node, "nodes/self/controller/settings", nil); err != nil {
			return err
		}
		if err := db.httpPost(node, "node/controller/rename", map[string]string{
			"hostname": node}); err != nil {
			return err
		}
	}
	if err := db.httpPost(opt.Nodes[0], "clusterInit", map[string]string{
		"hostname": opt.Nodes[0],
		"username": db.user,
		"password": db.pass,
		"services": "kv",
		"port":     "SAME"}); err != nil {
		return err
	}
	for i, node := range opt.Nodes {
		if i == 0 {
			continue
		}
		if err := db.httpPost(node, "node/controller/doJoinCluster", map[string]string{
			"hostname": opt.Nodes[0],
			"user":     db.user,
			"password": db.pass,
			"services": "kv"}); err != nil {
			return err
		}
	}
	var knownNodes strings.Builder
	for i, node := range opt.Nodes {
		if i == 0 {
			knownNodes.WriteString("ns_1@")
		} else {
			knownNodes.WriteString(",ns_1@")
		}
		knownNodes.WriteString(node)
	}
	if err := db.httpPost(opt.Nodes[0], "controller/rebalance", map[string]string{
		"knownNodes": knownNodes.String()}); err != nil {
		return err
	}
	// Wait for rebalance to complete
	for {
		time.Sleep(time.Second)
		bytes, err := db.httpGet(opt.Nodes[0], "pools/default/rebalanceProgress")
		if err != nil {
			return err
		}
		obj := make(map[string]interface{})
		if err := json.Unmarshal(bytes, &obj); err != nil {
			return fmt.Errorf("kv: cannot parse rebalance progress: %v", err)
		}
		status, ok := obj["status"].(string)
		if !ok {
			return fmt.Errorf("kv: cannot find rebalance status in %s", string(bytes))
		}
		if status == "none" {
			log.Info("Rebalance completed")
			break
		}
		log.Info("Rebalance in progress: %s", string(bytes))
	}
	if err := db.httpPost(opt.Nodes[0], "settings/autoFailover", map[string]string{
		"enabled":                            "true",
		"timeout":                            "15",
		"failoverPreserveDurabilityMajority": "true"}); err != nil {
		return err
	}
	if err := db.httpPost(opt.Nodes[0], "pools/default/buckets", map[string]string{
		"name":           "default",
		"ramQuota":       "1024",
		"evictionPolicy": "fullEviction",
		"replicaNumber":  strconv.Itoa(db.replicas),
		"flushEnabled":   "1"}); err != nil {
		return err
	}
	time.Sleep(5 * time.Second) // Wait for bucket creation

	return nil
}

func (db *database) TearDown() error {
	return nil
}

func (db *database) httpGet(node, endpoint string) ([]byte, error) {
	uri := fmt.Sprintf("http://%s:%s@%s:8091/%s", db.user, db.pass, node, endpoint)
	log.Info("HTTP GET %s", uri)
	resp, err := http.Get(uri)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, err
	}
	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP GET returned %d: %s", resp.StatusCode, string(bytes))
	}
	return bytes, nil
}

func (db *database) httpPost(node, endpoint string, form map[string]string) error {
	values := make(url.Values, len(form))
	for k, v := range form {
		values.Set(k, v)
	}
	uri := fmt.Sprintf("http://%s:%s@%s:8091/%s", db.user, db.pass, node, endpoint)
	log.Info("HTTP POST %s %s %s", node, endpoint, values.Encode())
	resp, err := http.PostForm(uri, values)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusAccepted {
			return nil
		}
		bytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("HTTP POST returned %d: %s", resp.StatusCode, string(bytes))
	}
	return nil
}

func (db *database) NewClient(id int) (gorgon.Client, error) {
	uri := fmt.Sprintf("couchbase://%s:%d", db.options.Nodes[0], db.port)
	return NewClient(id, uri, db.user, db.pass, db.durability), nil
}

func (*database) Scenarios(opt *gorgon.Options) []gorgon.Scenario {
	return []gorgon.Scenario{
		{Workload: workloads.NewGetSetWorkload(), Nemesis: nil},
		{Workload: workloads.NewGetSetWorkload(), Nemesis: NewKillNemesis("memcached")},
		{Workload: workloads.NewGetSetWorkload(), Nemesis: nemeses.NewNetworkPartitionNemesis([]int{11210})},
	}
}
