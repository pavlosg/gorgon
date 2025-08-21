package cmd

import (
	"errors"
	"fmt"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/anishathalye/porcupine"
	"github.com/pavlosg/gorgon/src/gorgon"
	"github.com/pavlosg/gorgon/src/gorgon/log"
)

type Runner struct {
	name     string
	db       gorgon.Database
	workload gorgon.Workload
	options  *gorgon.Options
	clients  []gorgon.Client
}

func NewRunner(db gorgon.Database, workload gorgon.Workload, opts *gorgon.Options) *Runner {
	var sb strings.Builder
	sb.WriteString(db.Name())
	for _, gen := range workload.Generators {
		sb.WriteByte('~')
		sb.WriteString(gen.Name())
	}
	return &Runner{sb.String(), db, workload, opts, nil}
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
	for _, gen := range runner.workload.Generators {
		if err := gen.SetUp(runner.options); err != nil {
			log.Error("[%s] Error in Generator.SetUp: %v", runner.name, err)
			return err
		}
	}
	runner.clients = clients
	clients = nil
	return nil
}

func (runner *Runner) Run() ([]gorgon.Operation, error) {
	stopFlag := &atomic.Bool{}
	defer stopFlag.Store(true)
	wg := &sync.WaitGroup{}
	genMutex := &sync.Mutex{}
	operationList := gorgon.NewOperationList()
	concurrency := runner.options.Concurrency
	log.Info("[%s] Starting workers", runner.name)
	deadline := time.Now().Add(runner.options.WorkloadDuration)
	for i := -1; i < concurrency; i++ {
		var client gorgon.Client
		if i >= 0 {
			client = runner.clients[i]
		}
		w := &worker{
			stopFlag:      stopFlag,
			wg:            wg,
			genMutex:      genMutex,
			generators:    runner.workload.Generators,
			client:        client,
			operations:    operationList,
			deadline:      deadline,
			stopAmbiguous: !runner.options.ContinueAmbiguousClient,
			name:          runner.name,
		}
		wg.Add(1)
		go w.run()
	}
	wg.Wait()
	log.Info("[%s] Workers finished", runner.name)
	return operationList.Extract(), nil
}

func (runner *Runner) TearDown() (retErr error) {
	for _, gen := range runner.workload.Generators {
		if err := gen.TearDown(); err != nil {
			log.Error("[%s] Error in Generator.TearDown: %v", runner.name, err)
			if retErr == nil {
				retErr = err
			}
		}
	}
	for i, client := range runner.clients {
		if client != nil {
			err := client.Close()
			if err != nil {
				log.Error("[%s] Client %d error: %v", runner.name, i, err)
				if retErr == nil {
					retErr = err
				}
			}
		}
	}
	return
}

func (runner *Runner) Check(history []gorgon.Operation, dir string) (err error) {
	const fileTime = "2006-01-02-150405-0700"
	model := runner.workload.Model
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
		result, info := porcupine.CheckOperationsVerbose(dmodel, hist, 40*time.Second)
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

type worker struct {
	stopFlag      *atomic.Bool
	wg            *sync.WaitGroup
	genMutex      *sync.Mutex
	generators    []gorgon.Generator
	client        gorgon.Client
	operations    *gorgon.OperationList
	deadline      time.Time
	stopAmbiguous bool
	name          string
}

func (w *worker) run() {
	defer w.wg.Done()
	id := -1
	if w.client != nil {
		id = w.client.Id()
	}
	for time.Until(w.deadline) >= 0 && !w.stopFlag.Load() {
		instr, gen, err := w.getNext(id)
		if err != nil {
			return
		}
		if instr == nil {
			time.Sleep(time.Millisecond)
			continue
		}

		if instr.ForSelf() {
			if err := w.onCall(id, instr); err != nil {
				return
			}
			_, output := gen.Invoke(instr, w.operations.GetTime)
			if err := w.onReturn(id, instr, output); err != nil {
				return
			}
			continue
		}

		if w.client == nil {
			panic(errors.New("worker has no client assigned"))
		}
		if err := w.onCall(id, instr); err != nil {
			return
		}
		op := gorgon.Operation{ClientId: id, Input: instr, Call: w.operations.GetTime()}
		retTime, output := w.client.Invoke(instr, w.operations.GetTime)
		if err := w.onReturn(id, instr, output); err != nil {
			return
		}

		op.Return = retTime
		op.Output = output
		if err, ok := output.(error); ok && !gorgon.IsUnambiguousError(err) {
			op.Return = -1
			if w.stopAmbiguous {
				log.Warning("[%s] Client %d returned ambiguous error: %T %v", w.name, id, err, err)
				w.operations.Append(op)
				return
			}
		}
		w.operations.Append(op)
	}
}

func (w *worker) getNext(id int) (gorgon.Instruction, gorgon.Generator, error) {
	w.genMutex.Lock()
	defer w.genMutex.Unlock()
	for i := len(w.generators) - 1; i >= 0; i-- {
		gen := w.generators[i]
		instr, err := gen.Next(id)
		if err != nil {
			log.Error("[%s] Generator %q failed: %v", w.name, gen.Name(), err)
			return nil, nil, err
		}
		if instr != nil {
			return instr, gen, nil
		}
	}
	return nil, nil, nil
}

func (w *worker) onCall(client int, instruction gorgon.Instruction) error {
	w.genMutex.Lock()
	defer w.genMutex.Unlock()
	for _, gen := range w.generators {
		if err := gen.OnCall(client, instruction); err != nil {
			log.Error("[%s] Generator %q OnCall failed: %v", w.name, gen.Name(), err)
			return err
		}
	}
	return nil
}

func (w *worker) onReturn(client int, instruction gorgon.Instruction, output gorgon.Output) error {
	w.genMutex.Lock()
	defer w.genMutex.Unlock()
	for _, gen := range w.generators {
		if err := gen.OnReturn(client, instruction, output); err != nil {
			log.Error("[%s] Generator %q OnReturn failed: %v", w.name, gen.Name(), err)
			return err
		}
	}
	return nil
}
