package vmjobs

import (
	"fmt"
	"github.com/urfave/cli"
)

type cmdLineJob struct {
	cmd string
}

var emptyCmdErr = fmt.Errorf("provided command is empty")

func (j *cmdLineJob) String() string {
	return "cmd"
}

func (j *cmdLineJob) Desc() string {
	return "Run a simple cmd line job."
}

func (j *cmdLineJob) Flags() []cli.Flag {
	return []cli.Flag{
		cli.StringSliceFlag{
			Name:     "image,i",
			Usage:    imageParamDesc,
			Required: true,
		},
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
		return emptyCmdErr
	}
	return nil
}

func (j *cmdLineJob) Cmd() string {
	return j.cmd
}

func (j *cmdLineJob) Process(VMOutput) {

}

func (j *cmdLineJob) Done() {

}
