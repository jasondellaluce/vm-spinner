package cmd

import (
	"bufio"
	"github.com/jasondellaluce/experiments/vm-spinner/vmjobs"
	"github.com/urfave/cli"
	"os"
)

type cmdLineJob struct {
	cmd string
}

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
			Name:  "line",
			Usage: "command that runs in each VM, as a command line parameter.",
		},
		cli.StringFlag{
			Name:  "file",
			Usage: "script that runs in each VM, as a filepath.",
		},
	}
}

func (j *cmdLineJob) ParseCfg(c *cli.Context) error {
	var (
		err  error
		file = os.Stdin
	)
	switch {
	case c.IsSet("line"):
		j.cmd = c.String("line")
	case c.IsSet("file"):
		file, err = os.Open(c.String("file"))
		if err != nil {
			return err
		}
		defer file.Close()
		fallthrough
	default:
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			j.cmd += scanner.Text() + "\n"
		}
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
