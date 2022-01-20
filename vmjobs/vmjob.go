package vmjobs

import (
	"fmt"
	"github.com/urfave/cli"
)

const (
	VMJobBpf    = "bpf"
	VMJobCmd    = "cmd"
	VMJobStdin  = "stdin"
	VMJobScript = "script"
	VMJobKmod   = "kmod"
)

type VMOutput struct {
	VM   string
	Line string
}

type VMJob interface {
	// Cmd returns cmd to be used
	Cmd() string
	// Process processes each output line
	Process(VMOutput)
	// Done is called at the end of program, to let job flush its data if needed
	Done()
}

func NewVMJob(c *cli.Context) (VMJob, error) {
	jobType := c.Command.Name
	switch jobType {
	case VMJobBpf:
		return newBpfJob(c)
	case VMJobCmd:
		return newCmdLineJob(c)
	case VMJobStdin:
		return newStdinJob()
	case VMJobScript:
		return newScriptJob(c)
	case VMJobKmod:
		return newKmodJob(c)
	}
	return nil, fmt.Errorf("job '%s' not supported", jobType)
}
