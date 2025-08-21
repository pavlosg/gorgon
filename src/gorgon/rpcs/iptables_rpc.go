package rpcs

import (
	"os/exec"

	"github.com/pavlosg/gorgon/src/gorgon/log"
)

type IpTablesRpc struct{}

func (*IpTablesRpc) IpTables(arg *[]string, reply *string) error {
	err := exec.Command("iptables", (*arg)...).Run()
	log.Info("IpTables(%v) returned %v", *arg, err)
	if err != nil {
		return err
	}
	*reply = "ok"
	return nil
}
