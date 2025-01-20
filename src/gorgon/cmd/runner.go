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

func (runner *Runner) Name() string {
	return runner.name
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
		client, err := runner.db.NewClient(i)
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
	log.Info("[%s] Nemesis SetUp", runner.name)
	if nemesis := runner.scenario.Nemesis; nemesis != nil {
		if err := nemesis.SetUp(runner.options); err != nil {
			return err
		}
	}
	runner.clients = clients
	clients = nil
	return nil
}

func (runner *Runner) Run() ([]gorgon.Operation, error) {
	workload := runner.scenario.Workload
	gen := generators.Synchronize(workload.Generator())
	log.Info("[%s] Starting workers", runner.name)
	wg := &sync.WaitGroup{}
	wgNemesis := &sync.WaitGroup{}
	deadline := time.Now().Add(runner.scenario.WorkloadDuration)
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
	if runner.scenario.Nemesis != nil {
		wgNemesis.Add(1)
		go runner.runNemesis(wgNemesis)
	}
	log.Info("[%s] Waiting for workers", runner.name)
	wg.Wait()
	log.Info("[%s] Workers finished", runner.name)
	wgNemesis.Wait()
	log.Info("[%s] Nemesis finished", runner.name)
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

func (runner *Runner) Check(history []gorgon.Operation, dir string) (err error) {
	const fileTime = "2006-01-02-150405-0700"
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
	now := time.Now()
	for i, part := range partitions {
		hist := make([]porcupine.Operation, len(part))
		for i := 0; i < len(part); i++ {
			op := part[i]
			hist[i] = porcupine.Operation{
				ClientId: op.ClientId,
				Input:    op.Input,
				Call:     op.Call,
				Output:   op.Output,
				Return:   op.Return,
			}
		}
		result, info := porcupine.CheckOperationsVerbose(dmodel, hist, 0)
		level := log.INFO
		if result != porcupine.Ok {
			level = log.WARNING
			filePath := path.Join(dir, EscapeFileName(fmt.Sprintf(
				"%s.%s.%d.html", now.Format(fileTime), runner.name, i)))
			visErr := porcupine.VisualizePath(dmodel, info, filePath)
			if visErr != nil && err == nil {
				err = visErr
			}
		}
		log.Log(level, "[%s] Checked partition %d - %s", runner.name, i, result)
	}
	return
}

func (runner *Runner) runNemesis(wg *sync.WaitGroup) {
	defer wg.Done()
	err := runner.scenario.Nemesis.Run()
	if err != nil {
		log.Error("[%s] Nemesis error: %v", runner.name, err)
	}
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
