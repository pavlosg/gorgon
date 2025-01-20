package gorgon

import "time"

type Options struct {
	Extras           map[string]string
	Concurrency      int
	RpcPort          int
	WorkloadDuration time.Duration
	Nodes            []string
}
