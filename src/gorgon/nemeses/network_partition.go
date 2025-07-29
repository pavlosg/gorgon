package nemeses

import (
	"fmt"
	"net/rpc"
	"os/exec"
	"strconv"
	"time"

	"github.com/pavlosg/gorgon/src/gorgon"
	"github.com/pavlosg/gorgon/src/gorgon/jrpc"
	"github.com/pavlosg/gorgon/src/gorgon/log"
	"github.com/pavlosg/gorgon/src/gorgon/splitmix"
)

type NetworkPartitionNemesis struct {
	allowedPorts []int
	options      *gorgon.Options
	client       *rpc.Client
	node         string
}

func NewNetworkPartitionNemesis(allowedPorts []int) *NetworkPartitionNemesis {
	return &NetworkPartitionNemesis{allowedPorts: allowedPorts}
}

func (*NetworkPartitionNemesis) Name() string {
	return "NetworkPartition"
}

func (nemesis *NetworkPartitionNemesis) SetUp(opt *gorgon.Options) error {
	nemesis.options = opt
	nemesis.node = opt.Nodes[splitmix.Rand.Intn(len(opt.Nodes))]
	client, err := jrpc.Dial(fmt.Sprintf("%s:%d", nemesis.node, opt.RpcPort), []byte("password"))
	if err != nil {
		return err
	}
	nemesis.client = client
	return nemesis.iptables("-A", "INPUT", "-p", "tcp", "--dport", strconv.Itoa(opt.RpcPort), "-j", "ACCEPT")
}

func (nemesis *NetworkPartitionNemesis) TearDown() error {
	if nemesis.client == nil {
		return nil
	}
	err := nemesis.iptables("-P", "INPUT", "ACCEPT")
	if err == nil {
		err = nemesis.iptables("-F")
	}
	nemesis.client.Close()
	return err
}

func (nemesis *NetworkPartitionNemesis) Run() error {
	deadline := time.Now().Add(nemesis.options.WorkloadDuration)
	time.Sleep(time.Until(deadline) / 4)
	for _, port := range nemesis.allowedPorts {
		err := nemesis.iptables("-A", "INPUT", "-p", "tcp", "--dport", strconv.Itoa(port), "-j", "ACCEPT")
		if err != nil {
			return err
		}
	}
	err := nemesis.iptables("-A", "INPUT", "-j", "DROP")
	if err != nil {
		return err
	}
	time.Sleep(time.Until(deadline) * 2 / 3)
	err = nemesis.iptables("-P", "INPUT", "ACCEPT")
	if err != nil {
		return err
	}
	err = nemesis.iptables("-F")
	if err != nil {
		return err
	}
	time.Sleep(time.Until(deadline))
	return nil
}

func (nemesis *NetworkPartitionNemesis) iptables(args ...string) error {
	var reply string
	return nemesis.client.Call("IpTablesRpc.IpTables", &args, &reply)
}

type IpTablesRpc struct{}

func (*IpTablesRpc) IpTables(arg *[]string, reply *string) error {
	err := exec.Command("iptables", (*arg)...).Run()
	log.Info("IpTables(%v) returned %v", *arg, err)
	if err != nil {
		return err
	}
	*reply = "ok"
	return nil
}
