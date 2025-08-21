package nemeses

import (
	"fmt"
	"net/rpc"
	"time"

	"github.com/pavlosg/gorgon/src/gorgon"
	"github.com/pavlosg/gorgon/src/gorgon/jrpc"
	"github.com/pavlosg/gorgon/src/gorgon/rpcs"
	"github.com/pavlosg/gorgon/src/gorgon/splitmix"
)

func NewKillNemesis(process string) gorgon.Generator {
	return &killNemesis{process: process}
}

type killNemesis struct {
	process string
	next    time.Time
	client  *rpc.Client
}

func (nemesis *killNemesis) Name() string {
	return fmt.Sprintf("Kill(%s)", nemesis.process)
}

func (*killNemesis) OnCall(client int, instruction gorgon.Instruction) error {
	return nil
}

func (*killNemesis) OnReturn(client int, instruction gorgon.Instruction, output gorgon.Output) error {
	return nil
}

func (nemesis *killNemesis) Next(client int) (gorgon.Instruction, error) {
	if client >= 0 || time.Until(nemesis.next) > 0 {
		return nil, nil
	}
	nemesis.next = nemesis.next.Add(8 * time.Second)
	return &rpcs.KillInstruction{Process: nemesis.process, Signal: 9}, nil
}

func (nemesis *killNemesis) SetUp(opt *gorgon.Options) error {
	node := opt.Nodes[splitmix.Rand.Intn(len(opt.Nodes))]
	client, err := jrpc.Dial(fmt.Sprintf("%s:%d", node, opt.RpcPort), []byte(opt.RpcPassword))
	if err != nil {
		return err
	}
	nemesis.next = time.Now().Add(4 * time.Second)
	nemesis.client = client
	return nil
}

func (nemesis *killNemesis) TearDown() error {
	if nemesis.client == nil {
		return nil
	}
	return nemesis.client.Close()
}

func (nemesis *killNemesis) Invoke(instruction gorgon.Instruction, getTime func() int64) (int64, gorgon.Output) {
	if instr, ok := instruction.(*rpcs.KillInstruction); ok {
		var reply string
		err := nemesis.client.Call("KillRpc.Pkill", instr, &reply)
		return getTime(), err
	}
	return -1, gorgon.ErrUnsupportedInstruction
}
