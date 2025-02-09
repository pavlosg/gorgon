package kv

import (
	"fmt"
	"net/rpc"
	"os/exec"
	"time"

	"github.com/pavlosg/gorgon/src/gorgon"
	"github.com/pavlosg/gorgon/src/gorgon/jrpc"
	"github.com/pavlosg/gorgon/src/gorgon/log"
	"github.com/pavlosg/gorgon/src/gorgon/splitmix"
)

type KillNemesis struct {
	process string
	options *gorgon.Options
	client  *rpc.Client
}

func NewKillNemesis(process string) *KillNemesis {
	return &KillNemesis{process: process}
}

func (nemesis *KillNemesis) Name() string {
	return fmt.Sprintf("Kill(%s)", nemesis.process)
}

func (nemesis *KillNemesis) SetUp(opt *gorgon.Options) error {
	nemesis.options = opt
	node := opt.Nodes[splitmix.Rand.Intn(len(opt.Nodes))]
	client, err := jrpc.Dial(fmt.Sprintf("%s:%d", node, opt.RpcPort), []byte("password"))
	if err != nil {
		return err
	}
	nemesis.client = client
	return nil
}

func (nemesis *KillNemesis) TearDown() error {
	if nemesis.client != nil {
		return nemesis.client.Close()
	}
	return nil
}

func (nemesis *KillNemesis) Run() error {
	deadline := time.Now().Add(nemesis.options.WorkloadDuration)
	time.Sleep(2 * time.Second)
	for time.Until(deadline) > 0 {
		var reply string
		arg := KillProcess{Name: nemesis.process, Signal: 9}
		err := nemesis.client.Call("KillRpc.Pkill", arg, &reply)
		if err != nil {
			return err
		}
		time.Sleep(6 * time.Second)
	}
	return nil
}

type KillRpc struct{}

type KillProcess struct {
	Name   string
	Signal uint
}

func (*KillRpc) Pkill(arg *KillProcess, reply *string) error {
	err := exec.Command("bash", "-c", fmt.Sprintf("kill -%v $(pgrep '%s' | head -n1)", arg.Signal, arg.Name)).Run()
	log.Info("Pkill(%v, %q) returned %v", arg.Signal, arg.Name, err)
	if err != nil {
		return err
	}
	*reply = "ok"
	return nil
}
