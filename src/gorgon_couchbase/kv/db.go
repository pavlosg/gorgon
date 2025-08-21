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
	"github.com/pavlosg/gorgon/src/gorgon/rpcs"
	"github.com/pavlosg/gorgon/src/gorgon/workloads"
)

type DatabaseConfig struct {
	User          *string
	Pass          *string
	Port          *int
	Replicas      *int
	Durability    *string
	Timeout       *time.Duration
	ClientOverRpc *bool
}

func NewDatabase(config DatabaseConfig) gorgon.Database {
	return &database{config: config}
}

type database struct {
	config     DatabaseConfig
	options    *gorgon.Options
	durability gocb.DurabilityLevel
}

func (*database) Name() string {
	return "couchbase"
}

func (db *database) SetOptions(opt *gorgon.Options) error {
	db.options = opt
	if durability := *db.config.Durability; len(durability) != 0 {
		db.durability = parseDurabilityLevel(durability)
		if db.durability == gocb.DurabilityLevelUnknown {
			return fmt.Errorf("kv: invalid durability level %q", durability)
		}
	}
	if n := *db.config.Replicas; n < 0 || n > 3 {
		return fmt.Errorf("kv: invalid number of replicas %d", n)
	}
	return nil
}

func (db *database) SetUp() error {
	opt := db.options
	user := *db.config.User
	pass := *db.config.Pass
	replicas := *db.config.Replicas

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
		"username": user,
		"password": pass,
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
			"user":     user,
			"password": pass,
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
		"replicaNumber":  strconv.Itoa(replicas),
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
	uri := fmt.Sprintf("http://%s:%s@%s:8091/%s", *db.config.User, *db.config.Pass, node, endpoint)
	log.Info("HTTP GET %s %s", node, endpoint)
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
	uri := fmt.Sprintf("http://%s:%s@%s:8091/%s", *db.config.User, *db.config.Pass, node, endpoint)
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
	nodes := db.options.Nodes
	if *db.config.ClientOverRpc {
		return rpcs.NewClientOverRpc(id, nodes[id%len(nodes)], db.options), nil
	}
	uri := fmt.Sprintf("couchbase://%s:%d", strings.Join(nodes, ","), *db.config.Port)
	return NewClient(id, uri, *db.config.User, *db.config.Pass), nil
}

func (db *database) ClientConfig() string {
	config := ClientConfig{
		Durability: *db.config.Durability,
		Timeout:    *db.config.Timeout}
	configJson, err := json.Marshal(config)
	if err != nil {
		panic(err)
	}
	return string(configJson)
}

func (db *database) Workloads() []gorgon.Workload {
	return []gorgon.Workload{
		workloads.GetSetWorkload(),
		workloads.GetSetWorkload().Add(nemeses.NewKillNemesis("memcached")).Add(NewSetAfterKillGenerator()),
		workloads.GetSetWorkload().Add(nemeses.NewNetworkPartitionNemesis(8091)).Add(NewPartitionAwareGetSetGenerator()),
	}
}
