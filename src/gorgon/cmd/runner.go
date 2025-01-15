package cmd

import (
	"fmt"
	"path"
	"sync"
	"time"

	"github.com/anishathalye/porcupine"
	"github.com/pavlosg/gorgon/src/gorgon"
	"github.com/pavlosg/gorgon/src/gorgon/generators"
	"github.com/pavlosg/gorgon/src/gorgon/log"
)

type Runner struct {
	name     string
	db       gorgon.Database
	scenario gorgon.Scenario
	options  *gorgon.Options
	clients  []gorgon.Client
}

func NewRunner(db gorgon.Database, scenario gorgon.Scenario, opts *gorgon.Options) *Runner {
	nemesis := "nil"
	if scenario.Nemesis != nil {
		nemesis = scenario.Nemesis.Name()
	}
	name := fmt.Sprintf("%s~%s~%s", db.Name(), scenario.Workload.Name(), nemesis)
	return &Runner{name, db, scenario, opts, nil}
}

func (runner *Runner) SetUp() error {
	workload := runner.scenario.Workload
	log.Info("[%s] Creating clients", runner.name)
	concurrency := runner.options.Concurrency
	clients := make([]gorgon.Client, concurrency)
	defer func() {
		for _, client := range clients {
			if client != nil {
				client.Close()
			}
		}
	}()
	for i := 0; i < concurrency; i++ {
		client, err := runner.db.NewClient()
		if err != nil {
			log.Error("[%s] Error creating new client: %v", runner.name, err)
			return err
		}
		err = client.Open()
		if err != nil {
			log.Error("[%s] Error opening client: %v", runner.name, err)
			return err
		}
		clients[i] = client
	}
	log.Info("[%s] Workload SetUp", runner.name)
	if err := workload.SetUp(runner.options, clients); err != nil {
		return err
	}
	runner.clients = clients
	clients = nil
	return nil
}

func (runner *Runner) Run() ([]porcupine.Operation, error) {
	workload := runner.scenario.Workload
	gen := generators.Synchronize(workload.Generator())
	log.Info("[%s] Starting workers", runner.name)
	wg := &sync.WaitGroup{}
	deadline := time.Now().Add(runner.options.WorkloadDuration)
	operationList := gorgon.NewOperationList()
	concurrency := runner.options.Concurrency
	for i := 0; i < concurrency; i++ {
		w := &worker{
			wg:         wg,
			gen:        gen,
			client:     runner.clients[i],
			operations: operationList,
			deadline:   deadline,
			name:       fmt.Sprintf("%s:%d", runner.name, i),
		}
		wg.Add(1)
		go w.run()
	}
	log.Info("[%s] Waiting for workers", runner.name)
	wg.Wait()
	log.Info("[%s] Workers finished", runner.name)
	return operationList.Extract(), nil
}

func (runner *Runner) TearDown() error {
	workload := runner.scenario.Workload
	err := workload.TearDown()
	for i, client := range runner.clients {
		if client != nil {
			err := client.Close()
			if err != nil {
				log.Error("[%s] Client %d error: %v", runner.name, i, err)
			}
		}
	}
	return err
}

func (runner *Runner) Check(history []porcupine.Operation, dir string) (err error) {
	workload := runner.scenario.Workload
	model := workload.Generator().Model()
	ndmodel := porcupine.NondeterministicModel{
		Init: model.Init,
		Step: func(state, input, output interface{}) []interface{} {
			return model.Step(state, input.(gorgon.Instruction), output)
		},
		Equal: model.Equal,
		DescribeOperation: func(input, output interface{}) string {
			return model.DescribeOperation(input.(gorgon.Instruction), output)
		},
		DescribeState: model.DescribeState,
	}
	dmodel := ndmodel.ToModel()
	partitions := model.Partition(history)
	for i, hist := range partitions {
		result, info := porcupine.CheckOperationsVerbose(dmodel, hist, 0)
		level := log.INFO
		if result != porcupine.Ok {
			level = log.WARNING
			filePath := path.Join(dir, fmt.Sprintf("%s.%d.html", runner.name, i))
			visErr := porcupine.VisualizePath(dmodel, info, filePath)
			if visErr != nil && err == nil {
				err = visErr
			}
		}
		log.Log(level, "[%s] Checked partition %d - %s", runner.name, i, result)
	}
	return
}

type worker struct {
	wg         *sync.WaitGroup
	gen        gorgon.Generator
	client     gorgon.Client
	operations *gorgon.OperationList
	deadline   time.Time
	name       string
}

func (w *worker) run() {
	defer w.wg.Done()
	for time.Now().Before(w.deadline) {
		instr, err := w.gen.NextInstruction()
		if err != nil {
			log.Error("[%s] Generator.NextInstruction(): %v", w.name, err)
			return
		}
		if instr == nil {
			return
		}
		if _, ok := instr.(gorgon.InstructionPending); ok {
			time.Sleep(time.Millisecond)
			continue
		}
		op := w.client.Invoke(instr, w.operations.GetTime)
		w.operations.Append(op)
	}
}
