package rpcs

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/rpc"
	"reflect"
	"strconv"
	"sync"

	"github.com/pavlosg/gorgon/src/gorgon"
	"github.com/pavlosg/gorgon/src/gorgon/jrpc"
	"github.com/pavlosg/gorgon/src/gorgon/log"
)

func RegisterInstruction(instrunction gorgon.Instruction) {
	rtype := reflect.TypeOf(instrunction)
	for rtype.Kind() == reflect.Pointer {
		rtype = rtype.Elem()
	}
	instructions[rtype.PkgPath()+"."+rtype.Name()] = rtype
}

func NewClientOverRpc(id int, node string, opt *gorgon.Options) gorgon.Client {
	return &clientOverRpc{id: id, node: node, opt: opt}
}

func NewClientRpc(db gorgon.Database) *ClientRpc {
	return &ClientRpc{db: db, clients: make(map[int]*lockableClient)}
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

func (rpc *ClientRpc) Invoke(arg *RpcInvoke, reply *RpcInvokeReply) error {
	var instruction gorgon.Instruction
	if rtype, ok := instructions[arg.Instructon]; ok {
		instr := reflect.New(rtype).Interface()
		if err := json.Unmarshal([]byte(arg.Value), instr); err != nil {
			return fmt.Errorf("ClientRpc.Invoke: error unmarshalling instruction %s: %v", arg.Instructon, err)
		}
		instruction = instr.(gorgon.Instruction)
	} else {
		return fmt.Errorf("ClientRpc.Invoke: unknown instruction type %s", arg.Instructon)
	}
	output := rpc.invoke(arg.Id, instruction)
	if output == nil {
		reply.Type = "nil"
		reply.Value = "null"
		return nil
	}
	switch v := output.(type) {
	case int:
		reply.Type = "int"
		reply.Value = strconv.Itoa(v)
	case string:
		reply.Type = "string"
		reply.Value = v
	case error:
		if gorgon.IsUnambiguousError(v) {
			reply.Type = "unambiguous_error"
		} else {
			reply.Type = "error"
		}
		reply.Value = v.Error()
	default:
		return fmt.Errorf("ClientRpc.Invoke: unexpected output type %T", output)
	}
	return nil
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
	instructionJson, err := json.Marshal(instruction)
	if err != nil {
		log.Error("ClientOverRpc.Invoke: failed to marshal instruction %T", instruction)
		op.Output = errors.New("ClientOverRpc: failed to marshal instruction")
		op.Return = getTime()
		return op
	}
	var reply RpcInvokeReply
	rtype := reflect.TypeOf(instruction)
	for rtype.Kind() == reflect.Pointer {
		rtype = rtype.Elem()
	}
	arg := RpcInvoke{Id: c.id, Instructon: rtype.PkgPath() + "." + rtype.Name(), Value: string(instructionJson)}
	err = c.client.Call("ClientRpc.Invoke", &arg, &reply)
	op.Return = getTime()
	if err != nil {
		op.Output = err
		return op
	}
	switch reply.Type {
	case "nil":
		op.Output = nil
	case "int":
		i, err := strconv.Atoi(reply.Value)
		if err != nil {
			op.Output = fmt.Errorf("ClientOverRpc.Invoke: expected int, got %s", reply.Value)
			return op
		}
		op.Output = i
	case "string":
		op.Output = reply.Value
	case "unambiguous_error":
		op.Output = gorgon.WrapUnambiguousError(errors.New(reply.Value))
	case "error":
		op.Output = errors.New(reply.Value)
	default:
		op.Output = fmt.Errorf("ClientOverRpc.Invoke: unexpected reply type %s", reply.Type)
	}
	return op
}

type RpcOpenClient struct {
	Id     int
	Config string
}

type RpcInvoke struct {
	Id         int
	Instructon string
	Value      string
}

type RpcInvokeReply struct {
	Type  string
	Value string
}

var instructions = make(map[string]reflect.Type)
