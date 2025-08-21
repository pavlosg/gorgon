package rpcs

import (
	"fmt"
	"os/exec"

	"github.com/pavlosg/gorgon/src/gorgon/log"
)

type KillRpc struct{}

type KillInstruction struct {
	Process string
	Signal  uint
}

func (instr *KillInstruction) String() string {
	return fmt.Sprintf("Kill(%v, %q)", instr.Signal, instr.Process)
}

func (*KillInstruction) ForSelf() bool {
	return true
}

func (*KillRpc) Pkill(arg *KillInstruction, reply *string) error {
	err := exec.Command("pkill", fmt.Sprintf("-%v", arg.Signal), arg.Process).Run()
	log.Info("Pkill(%v, %q) returned %v", arg.Signal, arg.Process, err)
	if err != nil {
		return err
	}
	*reply = "ok"
	return nil
}
