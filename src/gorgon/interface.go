package gorgon

import (
	"errors"
	"time"
)

var ErrUnsupportedInstruction = WrapUnambiguousError(errors.New("gorgon: unsupported instruction"))

type Instruction interface {
	String() string
	ForSelf() bool
}

type Client interface {
	Id() int
	Open(config string) error
	Invoke(instruction Instruction, getTime func() int64) (int64, Output)
	Close() error
}

type Generator interface {
	Name() string
	SetUp(opt *Options) error
	Next(client int) (Instruction, error)
	Invoke(instruction Instruction, getTime func() int64) (int64, Output)
	OnCall(client int, instruction Instruction) error
	OnReturn(client int, instruction Instruction, output Output) error
	TearDown() error
}

type Workload struct {
	Model
	Generators []Generator
}

func (w Workload) Add(generator Generator) Workload {
	w.Generators = append(w.Generators, generator)
	return w
}

type Database interface {
	Name() string
	SetOptions(opt *Options) error
	Workloads() []Workload
	SetUp() error
	NewClient(id int) (Client, error)
	ClientConfig() string
	TearDown() error
}

type Options struct {
	Args                    []string
	Nodes                   []string
	WorkloadDuration        time.Duration
	Concurrency             int
	ContinueAmbiguousClient bool
	RpcPort                 int
	RpcPassword             string
}

type Operation struct {
	ClientId int
	Input    Instruction
	Call     int64 // invocation timestamp
	Output   Output
	Return   int64 // response timestamp
}

type State = any

type Output = any

type Model struct {
	// Partition functions, such that a history is linearizable if and only
	// if each partition is linearizable. If left nil, this package will
	// skip partitioning.
	Partition func(history []Operation) [][]Operation
	// Initial state of the system.
	Init func() []State
	// Step function for the system. Returns all possible next states for
	// the given state, input, and output. If the system cannot step with
	// the given state/input to produce the given output, this function
	// should return an empty slice.
	Step func(state State, input Instruction, output Output) []State
	// Equality on states. If left nil, this package will use == as a
	// fallback.
	Equal func(state1, state2 State) bool
	// For visualization, describe an operation as a string.
	DescribeOperation func(input Instruction, output Output) string
	// For visualization purposes, describe a state as a string.
	DescribeState func(state State) string
}
