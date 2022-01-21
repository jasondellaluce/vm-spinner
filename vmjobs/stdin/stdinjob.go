package stdin

import (
	"bufio"
	"github.com/jasondellaluce/experiments/vm-spinner/vmjobs"
	"github.com/jasondellaluce/experiments/vm-spinner/vmjobs/cmd"
	"github.com/urfave/cli"
	"os"
)

type stdinJob struct {
	cmd string
}

func init() {
	j := &stdinJob{}
	_ = vmjobs.RegisterJob(j.String(), j)
}

func (j *stdinJob) String() string {
	return "stdin"
}

func (j *stdinJob) Desc() string {
	return "Run a simple cmd line job read from stdin."
}

func (j *stdinJob) Flags() []cli.Flag {
	return nil
}

func (j *stdinJob) ParseCfg(_ *cli.Context) error {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		j.cmd += scanner.Text() + "\n"
	}
	if len(j.cmd) == 0 {
		return cmd.EmptyCmdErr
	}
	return nil
}

func (j *stdinJob) Cmd() string {
	return j.cmd
}

func (j *stdinJob) Process(_, _ string) {

}

func (j *stdinJob) Done() {

}
