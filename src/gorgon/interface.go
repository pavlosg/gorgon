package gorgon

import (
	"errors"
	"time"
)

var ErrUnsupportedInstruction = errors.New("gorgon: unsupported instruction")

type Instruction interface {
	String() string
}

type InstructionPending struct{}

func (InstructionPending) String() string {
	return "InstructionPending"
}

type ClearDatabaseInstruction struct{}

func (ClearDatabaseInstruction) String() string {
	return "ClearDatabase"
}

type Client interface {
	Open() error
	Invoke(instruction Instruction, getTime func() int64) Operation
	Close() error
}

type Generator interface {
	NextInstruction() (Instruction, error)
	Model() Model
}

type Nemesis interface {
	Name() string
	SetUp(opt *Options) error
	Run() error
	TearDown() error
}

type Workload interface {
	Name() string
	SetUp(opt *Options, clients []Client) error
	Generator() Generator
	TearDown() error
}

type Scenario struct {
	Workload
	Nemesis
	WorkloadDuration time.Duration
}

type Database interface {
	Name() string
	Scenarios(opt *Options) []Scenario
	SetUp(opt *Options) error
	NewClient(id int) (Client, error)
	TearDown() error
}

type Operation struct {
	ClientId int
	Input    Instruction
	Call     int64 // invocation timestamp
	Output   interface{}
	Return   int64 // response timestamp
}

type State = interface{}

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
	Step func(state State, input Instruction, output interface{}) []State
	// Equality on states. If left nil, this package will use == as a
	// fallback.
	Equal func(state1, state2 State) bool
	// For visualization, describe an operation as a string.
	DescribeOperation func(input Instruction, output interface{}) string
	// For visualization purposes, describe a state as a string.
	DescribeState func(state State) string
}
