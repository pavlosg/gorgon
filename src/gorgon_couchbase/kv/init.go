package kv

import "net/rpc"

func init() {
	rpc.Register(&KillRpc{})
}
