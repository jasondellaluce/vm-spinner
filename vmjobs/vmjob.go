package vmjobs

import (
	"fmt"
	"github.com/urfave/cli"
)

type VMOutput struct {
	VM   string
	Line string
}

type VMJob interface {
	// Cmd returns cmd to be used
	Cmd() string
	// Images returns list of images to be used
	Images() []string
	// Process processes each output line
	Process(VMOutput)
	// Done is called at the end of program, to let job flush its data if needed
	Done()
}

func NewVMJob(c *cli.Context) (VMJob, error) {
	jobType := c.Command.Name
	switch jobType {
	case "bpf":
		return newBpfJob(c)
	case "cmd":
		return newCmdLineJob(c)
	case "stdin":
		return newStdinJob(c)
	case "script":
		return newScriptJob(c)
	case "kmod":
		return newKmodJob(c)
	}
	return nil, fmt.Errorf("job '%s' not supported", jobType)
}
