package main

import (
	"flag"
	"log"
	"net/rpc"
	"os"
	"time"

	"github.com/pavlosg/gorgon/src/gorgon/cmd"
	"github.com/pavlosg/gorgon/src/gorgon/generators"
	"github.com/pavlosg/gorgon/src/gorgon/nemeses"
	"github.com/pavlosg/gorgon/src/gorgon/rpcs"
	"github.com/pavlosg/gorgon/src/gorgon_couchbase/kv"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	db := kv.NewDatabase(kv.DatabaseConfig{
		User:          flag.String("user", "Administrator", "Couchbase username"),
		Pass:          flag.String("pass", "password", "Couchbase password"),
		Port:          flag.Int("port", 11210, "Couchbase port"),
		Replicas:      flag.Int("replicas", 1, "Number of Couchbase replicas (0-3)"),
		Durability:    flag.String("durability", "none", "Couchbase durability level"),
		Timeout:       flag.Duration("timeout", 5*time.Second, "Couchbase operation timeout"),
		ClientOverRpc: flag.Bool("client-over-rpc", false, "Use RPC for client operations"),
	})

	rpc.Register(rpcs.NewClientRpc(db))
	rpc.Register(&nemeses.IpTablesRpc{})
	rpc.Register(&kv.KillRpc{})

	rpcs.RegisterInstruction(&generators.GetInstruction{})
	rpcs.RegisterInstruction(&generators.SetInstruction{})

	code := cmd.Main(db)
	if code != 0 {
		os.Exit(code)
	}
}
