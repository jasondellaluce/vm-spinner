package vmjobs

import (
	"fmt"
	"github.com/urfave/cli"
)

type VMJobType int64

const (
	VMJobBpf VMJobType = iota
	VMJobCmd
	VMJobStdin
	VMJobScript
	VMJobKmod

	VMJobMax
)

type VMOutput struct {
	VM   string
	Line string
}

var imageParamDesc = "VM image to run the command on. Specify it multiple times for multiple vms."

type VMJob interface {
	fmt.Stringer // name for the job

	// Desc returns a job description that will be used as cmd line sub cmd description
	Desc() string
	// Flags returns list of cli.Flag supported specifically by the job
	Flags() []cli.Flag
	// ParseCfg is called when program starts on a job, to parse job specific config
	ParseCfg(c *cli.Context) error
	// Cmd returns cmd to be used
	Cmd() string
	// Process processes each output line
	Process(VMOutput)
	// Done is called at the end of program, to let job flush its data if needed
	Done()
}

func NewVMJob(jobType VMJobType) (VMJob, error) {
	switch jobType {
	case VMJobBpf:
		return &bpfJob{}, nil
	case VMJobCmd:
		return &cmdLineJob{}, nil
	case VMJobStdin:
		return &stdinJob{}, nil
	case VMJobScript:
		return &scriptJob{}, nil
	case VMJobKmod:
		return &kmodJob{}, nil
	}
	return nil, fmt.Errorf("job %v not supported", jobType)
}
