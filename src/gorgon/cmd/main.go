package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pavlosg/gorgon/src/gorgon"
	"github.com/pavlosg/gorgon/src/gorgon/jrpc"
	"github.com/pavlosg/gorgon/src/gorgon/log"
)

const exitUsage = 2

func usage() int {
	fmt.Println("Usage:", os.Args[0], "run|rpc")
	return exitUsage
}

func Main(db gorgon.Database, args []string) int {
	if len(args) == 0 {
		return usage()
	}
	command := args[0]
	args = args[1:]
	if command == "run" {
		return cmdRun(db, args)
	} else if command == "rpc" {
		return cmdRpc(args)
	}
	fmt.Println("Unknown command:", command)
	return usage()
}

func cmdRun(db gorgon.Database, args []string) int {
	opt := &gorgon.Options{
		Concurrency: 3,
		RpcPort:     9090,
	}
	if ret := parseOptions(args, opt); ret != 0 {
		return ret
	}
	if err := db.SetUp(opt); err != nil {
		return 1
	}
	defer db.TearDown()
	scenarios := db.Scenarios(opt)
	for _, scenario := range scenarios {
		runner := NewRunner(db, scenario, opt)
		if err := runner.SetUp(); err != nil {
			log.Error("Error in Runner.SetUp: %v", err)
			return 1
		}
		history, err := runner.Run()
		if err != nil {
			return 1
		}
		if err := runner.TearDown(); err != nil {
			log.Error("Error in Runner.TearDown: %v", err)
		}
		if err := runner.Check(history, ""); err != nil {
			log.Error("Error in Runner.Check: %v", err)
			return 1
		}
	}
	return 0
}

func parseOptions(args []string, opt *gorgon.Options) int {
	var flags Flags
	flags.Optional("--concurrency", "", &opt.Concurrency)
	flags.Optional("--rpc-port", "", &opt.RpcPort)
	workloadDuration := 30
	flags.Optional("--workload-duration", "", &workloadDuration)
	nodes := "localhost"
	flags.Optional("--nodes", "", &nodes)
	extras := ""
	flags.Optional("--extras", "", &extras)

	if !flags.Parse(args) {
		return exitUsage
	}

	if opt.Concurrency < 1 {
		fmt.Println("Invalid concurrency", opt.RpcPort)
		return exitUsage
	}
	if opt.RpcPort <= 0 || opt.RpcPort >= (1<<16) {
		fmt.Println("Invalid port", opt.RpcPort)
		return exitUsage
	}
	if workloadDuration < 10 {
		fmt.Println("Minimum workload duration 10s")
		return exitUsage
	}
	opt.WorkloadDuration = time.Duration(workloadDuration) * time.Second

	for _, node := range strings.Split(nodes, ",") {
		node = strings.TrimSpace(node)
		if len(node) == 0 {
			continue
		}
		opt.Nodes = append(opt.Nodes, node)
	}
	if len(opt.Nodes) == 0 {
		fmt.Println("Minimum one node")
		return exitUsage
	}

	opt.Extras = make(map[string]string)
	for _, pair := range strings.Split(extras, ";") {
		k, v, _ := strings.Cut(pair, "=")
		k = strings.TrimSpace(k)
		if len(k) == 0 {
			continue
		}
		v = strings.TrimSpace(v)
		opt.Extras[k] = v
	}
	return 0
}

func cmdRpc(args []string) int {
	var flags Flags
	port := 9090
	flags.Optional("--rpc-port", "", &port)
	if !flags.Parse(args) {
		return exitUsage
	}
	if port <= 0 || port >= (1<<16) {
		fmt.Println("Invalid port", port)
		return exitUsage
	}
	err := jrpc.Listen(fmt.Sprintf(":%v", port), []byte("password"))
	if err != nil {
		log.Error("rpc: %v", err)
		return 1
	}
	return 0
}
