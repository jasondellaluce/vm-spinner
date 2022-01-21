package vmjobs

import (
	"bufio"
	"github.com/urfave/cli"
	"os"
)

type scriptJob struct {
	cmd string
}

func (j *scriptJob) String() string {
	return "script"
}

func (j *scriptJob) Desc() string {
	return "Run a simple script job read from file."
}

func (j *scriptJob) Flags() []cli.Flag {
	return []cli.Flag{
		cli.StringSliceFlag{
			Name:     "image,i",
			Usage:    imageParamDesc,
			Required: true,
		},
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
		return emptyCmdErr
	}
	return nil
}

func (j *scriptJob) Cmd() string {
	return j.cmd
}

func (j *scriptJob) Process(VMOutput) {

}

func (j *scriptJob) Done() {

}
