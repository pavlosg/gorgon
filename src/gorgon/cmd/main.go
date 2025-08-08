package cmd

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pavlosg/gorgon/src/gorgon"
	"github.com/pavlosg/gorgon/src/gorgon/jrpc"
	"github.com/pavlosg/gorgon/src/gorgon/log"
)

const exitUsage = 2

func Main(db gorgon.Database) int {
	var filter Filter
	opt := &gorgon.Options{
		WorkloadDuration: time.Minute,
		Concurrency:      3,
		RpcPort:          9090,
	}
	ret := parseOptions(opt, &filter)
	if ret != 0 {
		return ret
	}
	if err := db.SetOptions(opt); err != nil {
		log.Error("Error in Database.SetOptions: %v", err)
		return 1
	}
	switch flag.Arg(0) {
	case "run":
		return cmdRun(db, opt, &filter)
	case "rpc":
		return cmdRpc(opt)
	}
	return usage()
}

func usage() int {
	fmt.Println("Usage:", os.Args[0], "[options] run|rpc [args...]")
	return exitUsage
}

func cmdRun(db gorgon.Database, opt *gorgon.Options, filter *Filter) int {
	scenarios := db.Scenarios()
	for _, scenario := range scenarios {
		runner := NewRunner(db, scenario, opt)
		if !filter.Match(runner.Name()) {
			continue
		}
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

func cmdRpc(opt *gorgon.Options) int {
	err := jrpc.Listen(fmt.Sprintf(":%v", opt.RpcPort), []byte(opt.RpcPassword))
	if err != nil {
		log.Error("rpc: %v", err)
		return 1
	}
	return 0
}

func parseOptions(opt *gorgon.Options, filter *Filter) int {
	matchPattern := "*"
	excludePattern := ""
	nodes := "localhost"

	flag.StringVar(&matchPattern, "gorgon-filter", matchPattern, "Wildcard pattern for scenarios to run")
	flag.StringVar(&excludePattern, "gorgon-exclude", excludePattern, "Wildcard pattern for scenarios to exclude")
	flag.StringVar(&nodes, "gorgon-nodes", nodes, "Comma-separated list of nodes")
	flag.DurationVar(&opt.WorkloadDuration, "gorgon-workload-duration", opt.WorkloadDuration, "Intended workload/nemesis duration")
	flag.IntVar(&opt.Concurrency, "gorgon-concurrency", opt.Concurrency, "Number of clients to use")
	flag.IntVar(&opt.RpcPort, "gorgon-rpc-port", opt.RpcPort, "RPC port to connect")
	flag.StringVar(&opt.RpcPassword, "gorgon-rpc-password", opt.RpcPassword, "RPC password")

	flag.Parse()
	if flag.NArg() == 0 {
		return usage()
	}

	opt.Args = flag.Args()[1:]

	*filter = MakeFilter(matchPattern, excludePattern)
	if opt.Concurrency < 1 {
		fmt.Println("Invalid concurrency", opt.Concurrency)
		return exitUsage
	}
	if opt.RpcPort <= 0 || opt.RpcPort >= (1<<16) {
		fmt.Println("Invalid port", opt.RpcPort)
		return exitUsage
	}
	if opt.WorkloadDuration < 10*time.Second {
		fmt.Println("Minimum workload duration 10s")
		return exitUsage
	}

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

	return 0
}
