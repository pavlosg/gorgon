package cmd

import (
	"fmt"
	"path"
	"sync"
	"sync/atomic"
	"time"

	"github.com/anishathalye/porcupine"
	"github.com/pavlosg/gorgon/src/gorgon"
	"github.com/pavlosg/gorgon/src/gorgon/generators"
	"github.com/pavlosg/gorgon/src/gorgon/log"
	"github.com/pavlosg/gorgon/src/gorgon/nemeses"
)

type Runner struct {
	name     string
	db       gorgon.Database
	scenario gorgon.Scenario
	options  *gorgon.Options
	clients  []gorgon.Client
}

func NewRunner(db gorgon.Database, scenario gorgon.Scenario, opts *gorgon.Options) *Runner {
	if scenario.Nemesis == nil {
		scenario.Nemesis = &nemeses.NilNemesis{}
	}
	name := fmt.Sprintf("%s~%s~%s", db.Name(), scenario.Workload.Name(), scenario.Nemesis.Name())
	return &Runner{name, db, scenario, opts, nil}
}

func (runner *Runner) Name() string {
	return runner.name
}

func (runner *Runner) SetUp() error {
	log.Info("[%s] Database SetUp", runner.name)
	if err := runner.db.SetUp(); err != nil {
		return err
	}
	defer func() {
		if err := runner.db.TearDown(); err != nil {
			log.Error("[%s] Error in Database.TearDown: %v", runner.name, err)
		}
	}()
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
	config := runner.db.ClientConfig()
	for i := 0; i < concurrency; i++ {
		client, err := runner.db.NewClient(i)
		if err != nil {
			log.Error("[%s] Error creating new client: %v", runner.name, err)
			return err
		}
		err = client.Open(config)
		if err != nil {
			log.Error("[%s] Error opening client: %v", runner.name, err)
			return err
		}
		clients[i] = client
	}
	log.Info("[%s] Workload SetUp", runner.name)
	if err := runner.scenario.Workload.SetUp(runner.options, clients); err != nil {
		return err
	}
	log.Info("[%s] Nemesis SetUp", runner.name)
	if err := runner.scenario.Nemesis.SetUp(runner.options); err != nil {
		return err
	}
	runner.clients = clients
	clients = nil
	return nil
}

func (runner *Runner) Run() ([]gorgon.Operation, error) {
	workload := runner.scenario.Workload
	gen := generators.Synchronize(workload.Generator())
	stopFlag := &atomic.Bool{}
	defer stopFlag.Store(true)
	wg := &sync.WaitGroup{}
	wgNemesis := &sync.WaitGroup{}
	operationList := gorgon.NewOperationList()
	concurrency := runner.options.Concurrency
	log.Info("[%s] Starting workers", runner.name)
	for i := 0; i < concurrency; i++ {
		w := &worker{
			stopFlag:   stopFlag,
			wg:         wg,
			gen:        gen,
			client:     runner.clients[i],
			operations: operationList,
			name:       fmt.Sprintf("%s:%d", runner.name, i),
		}
		wg.Add(1)
		go w.run()
	}
	log.Info("[%s] Starting nemesis", runner.name)
	wgNemesis.Add(1)
	go runner.runNemesis(stopFlag, wgNemesis)
	wgNemesis.Wait()
	log.Info("[%s] Nemesis finished", runner.name)
	wg.Wait()
	log.Info("[%s] Workers finished", runner.name)
	return operationList.Extract(), nil
}

func (runner *Runner) TearDown() error {
	errNemesis := runner.scenario.Nemesis.TearDown()
	errWorkload := runner.scenario.Workload.TearDown()
	for i, client := range runner.clients {
		if client != nil {
			err := client.Close()
			if err != nil {
				log.Error("[%s] Client %d error: %v", runner.name, i, err)
			}
		}
	}
	if errNemesis != nil {
		return errNemesis
	}
	return errWorkload
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

func (runner *Runner) runNemesis(stopFlag *atomic.Bool, wg *sync.WaitGroup) {
	defer func() {
		stopFlag.Store(true)
		wg.Done()
	}()
	err := runner.scenario.Nemesis.Run()
	if err != nil {
		log.Error("[%s] Nemesis error: %v", runner.name, err)
	}
}

type worker struct {
	stopFlag   *atomic.Bool
	wg         *sync.WaitGroup
	gen        gorgon.Generator
	client     gorgon.Client
	operations *gorgon.OperationList
	name       string
}

func (w *worker) run() {
	defer w.wg.Done()
	for !w.stopFlag.Load() {
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
