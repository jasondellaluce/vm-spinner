package cmd

import (
	"fmt"
	"github.com/jasondellaluce/experiments/vm-spinner/vmjobs"
	"github.com/urfave/cli"
)

type cmdLineJob struct {
	cmd string
}

var EmptyCmdErr = fmt.Errorf("provided command is empty")

func init() {
	j := &cmdLineJob{}
	_ = vmjobs.RegisterJob(j.String(), j)
}

func (j *cmdLineJob) String() string {
	return "cmd"
}

func (j *cmdLineJob) Desc() string {
	return "Run a simple cmd line job."
}

func (j *cmdLineJob) Flags() []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:     "line",
			Usage:    "command that runs in each VM, as a command line parameter.",
			Required: true,
		},
	}
}

func (j *cmdLineJob) ParseCfg(c *cli.Context) error {
	j.cmd = c.String("line")
	if len(j.cmd) == 0 {
		return EmptyCmdErr
	}
	return nil
}

func (j *cmdLineJob) Cmd() string {
	return j.cmd
}

func (j *cmdLineJob) Process(_, _ string) {

}

func (j *cmdLineJob) Done() {

}
