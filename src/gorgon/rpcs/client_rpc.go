package rpcs

import (
	"errors"
	"fmt"
	"net/rpc"
	"strconv"
	"sync"

	"github.com/pavlosg/gorgon/src/gorgon"
	"github.com/pavlosg/gorgon/src/gorgon/generators"
	"github.com/pavlosg/gorgon/src/gorgon/jrpc"
	"github.com/pavlosg/gorgon/src/gorgon/log"
)

type RpcOpenClient struct {
	Id     int
	Config string
}

type RpcGet struct {
	Id  int
	Key string
}

type RpcSet struct {
	Id    int
	Key   string
	Value int
}

type ClientRpc struct {
	db      gorgon.Database
	clients map[int]*lockableClient
	mutex   sync.Mutex
}

type lockableClient struct {
	client gorgon.Client
	mutex  sync.Mutex
}

func NewClientOverRpc(id int, node string, opt *gorgon.Options) gorgon.Client {
	return &clientOverRpc{id: id, node: node, opt: opt}
}

func NewClientRpc(db gorgon.Database) *ClientRpc {
	return &ClientRpc{db: db, clients: make(map[int]*lockableClient)}
}

func (rpc *ClientRpc) OpenClient(arg *RpcOpenClient, reply *string) error {
	rpc.mutex.Lock()
	defer rpc.mutex.Unlock()
	if _, ok := rpc.clients[arg.Id]; ok {
		return errors.New("ClientRpc: client already exists")
	}
	client, err := rpc.db.NewClient(arg.Id)
	if err != nil {
		return err
	}
	if err := client.Open(arg.Config); err != nil {
		return err
	}
	rpc.clients[arg.Id] = &lockableClient{client: client}
	*reply = "ok"
	log.Info("ClientRpc opened client %d with config %s", arg.Id, arg.Config)
	return nil
}

func (rpc *ClientRpc) CloseClient(id *int, reply *string) error {
	rpc.mutex.Lock()
	defer rpc.mutex.Unlock()
	if client, ok := rpc.clients[*id]; ok {
		client.mutex.Lock()
		defer client.mutex.Unlock()
		if err := client.client.Close(); err != nil {
			return err
		}
		delete(rpc.clients, *id)
		*reply = "ok"
		return nil
	}
	return errors.New("ClientRpc: client not found")
}

func (rpc *ClientRpc) Get(arg *RpcGet, reply *string) error {
	output := rpc.invoke(arg.Id, &generators.GetInstruction{Key: arg.Key})
	if output == nil {
		*reply = "null"
		return nil
	}
	switch v := output.(type) {
	case int:
		*reply = strconv.Itoa(v)
		return nil
	case error:
		return v
	default:
		log.Warning("ClientRpc.Get: unexpected output type %T", output)
		return errors.New("ClientRpc: unexpected output type")
	}
}

func (rpc *ClientRpc) Set(arg *RpcSet, reply *string) error {
	output := rpc.invoke(arg.Id, &generators.SetInstruction{Key: arg.Key, Value: arg.Value})
	if output == nil {
		*reply = "ok"
		return nil
	}
	if err, ok := output.(error); ok {
		if gorgon.IsUnambiguousError(err) {
			*reply = "unambiguous"
		}
		return err
	}
	return errors.New("ClientRpc: unexpected output type")
}

func (rpc *ClientRpc) ClearDatabase(id *int, reply *string) error {
	output := rpc.invoke(*id, &gorgon.ClearDatabaseInstruction{})
	if output == nil {
		*reply = "ok"
		return nil
	}
	if err, ok := output.(error); ok {
		return err
	}
	return errors.New("ClientRpc: unexpected output type")
}

func (rpc *ClientRpc) invoke(id int, instruction gorgon.Instruction) interface{} {
	var client *lockableClient
	rpc.mutex.Lock()
	if c, ok := rpc.clients[id]; ok {
		rpc.mutex.Unlock()
		client = c
	} else {
		rpc.mutex.Unlock()
		return errors.New("ClientRpc: client not found")
	}
	client.mutex.Lock()
	defer client.mutex.Unlock()
	op := client.client.Invoke(instruction, func() int64 { return 0 })
	return op.Output
}

type clientOverRpc struct {
	id     int
	node   string
	opt    *gorgon.Options
	client *rpc.Client
}

func (c *clientOverRpc) Open(config string) error {
	client, err := jrpc.Dial(fmt.Sprintf("%s:%d", c.node, c.opt.RpcPort), []byte(c.opt.RpcPassword))
	if err != nil {
		return err
	}
	err = client.Call("ClientRpc.OpenClient", &RpcOpenClient{Id: c.id, Config: config}, new(string))
	if err != nil {
		client.Close()
		return err
	}
	c.client = client
	return nil
}

func (c *clientOverRpc) Close() error {
	err := c.client.Call("ClientRpc.CloseClient", &c.id, new(string))
	if err != nil {
		return err
	}
	return c.client.Close()
}

func (c *clientOverRpc) Invoke(instruction gorgon.Instruction, getTime func() int64) gorgon.Operation {
	op := gorgon.Operation{ClientId: c.id, Input: instruction, Call: getTime()}
	switch instr := instruction.(type) {
	case *generators.GetInstruction:
		var reply string
		err := c.client.Call("ClientRpc.Get", &RpcGet{Id: c.id, Key: instr.Key}, &reply)
		op.Return = getTime()
		if err != nil {
			op.Output = err
		} else if reply == "null" {
			op.Output = nil
		} else if i, err := strconv.Atoi(reply); err == nil {
			op.Output = i
		} else {
			op.Output = reply
		}
	case *generators.SetInstruction:
		var reply string
		err := c.client.Call("ClientRpc.Set", &RpcSet{Id: c.id, Key: instr.Key, Value: instr.Value}, &reply)
		op.Return = getTime()
		if err != nil && reply == "unambiguous" {
			err = gorgon.WrapUnambiguousError(err)
			log.Warning("ClientRpc.Set: unambiguous error for key %q: %v", instr.Key, err)
		}
		op.Output = err
	case *gorgon.ClearDatabaseInstruction:
		var reply string
		op.Output = c.client.Call("ClientRpc.ClearDatabase", &c.id, &reply)
		op.Return = getTime()
	default:
		op.Return = getTime()
		op.Output = gorgon.ErrUnsupportedInstruction
	}
	return op
}
