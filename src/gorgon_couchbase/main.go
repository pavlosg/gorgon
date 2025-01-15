package main

import (
	"log"
	"os"

	"github.com/pavlosg/gorgon/src/gorgon/cmd"
	"github.com/pavlosg/gorgon/src/gorgon_couchbase/kv"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	db := kv.NewDatabase()
	code := cmd.Main(db, os.Args[1:])
	if code != 0 {
		os.Exit(code)
	}
}
