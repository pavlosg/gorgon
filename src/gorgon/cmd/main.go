package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/pavlosg/gorgon/src/gorgon"
	"github.com/pavlosg/gorgon/src/gorgon/log"
)

const fileTime = "2006-01-02-150405-0700"

const exitUsage = 2

func usage() int {
	fmt.Println("Usage:", os.Args[0], "run")
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
	}
	fmt.Println("Unknown command:", command)
	return usage()
}

func cmdRun(db gorgon.Database, args []string) int {
	opt := &gorgon.Options{
		Concurrency:      2,
		WorkloadDuration: 20 * time.Second,
		Nodes:            []string{"localhost"}, // []string{"n0", "n1", "n2", "n3", "n4"},
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
