package vmjobs

import (
	"bufio"
	"github.com/urfave/cli"
	"os"
)

type stdinJob struct {
	cmd string
}

func (j *stdinJob) String() string {
	return "stdin"
}

func (j *stdinJob) Desc() string {
	return "Run a simple cmd line job read from stdin."
}

func (j *stdinJob) Flags() []cli.Flag {
	return []cli.Flag{
		cli.StringSliceFlag{
			Name:     "image,i",
			Usage:    imageParamDesc,
			Required: true,
		},
	}
}

func (j *stdinJob) ParseCfg(_ *cli.Context) error {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		j.cmd += scanner.Text() + "\n"
	}
	if len(j.cmd) == 0 {
		return emptyCmdErr
	}
	return nil
}

func (j *stdinJob) Cmd() string {
	return j.cmd
}

func (j *stdinJob) Process(VMOutput) {

}

func (j *stdinJob) Done() {

}
