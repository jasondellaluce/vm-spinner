package script

import (
	"bufio"
	"github.com/jasondellaluce/experiments/vm-spinner/vmjobs"
	"github.com/jasondellaluce/experiments/vm-spinner/vmjobs/cmd"
	"github.com/urfave/cli"
	"os"
)

type scriptJob struct {
	cmd string
}

func init() {
	j := &scriptJob{}
	_ = vmjobs.RegisterJob(j.String(), j)
}

func (j *scriptJob) String() string {
	return "script"
}

func (j *scriptJob) Desc() string {
	return "Run a simple script job read from file."
}

func (j *scriptJob) Flags() []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:     "file",
			Usage:    "script that runs in each VM, as a filepath.",
			Required: true,
		},
	}
}

func (j *scriptJob) ParseCfg(c *cli.Context) error {
	file, err := os.Open(c.String("file"))
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		j.cmd += scanner.Text() + "\n"
	}

	if len(j.cmd) == 0 {
		return cmd.EmptyCmdErr
	}
	return nil
}

func (j *scriptJob) Cmd() string {
	return j.cmd
}

func (j *scriptJob) Process(_, _ string) {

}

func (j *scriptJob) Done() {

}
