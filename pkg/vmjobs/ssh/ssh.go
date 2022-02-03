package ssh

import (
	"bufio"
	"fmt"
	"github.com/jasondellaluce/experiments/vm-spinner/pkg/vmjobs"
	"github.com/urfave/cli"
	"os"
	"strings"
)

type sshJob struct {
	scanner     *bufio.Scanner
	exitOnError bool
}

func init() {
	j := &sshJob{scanner: bufio.NewScanner(os.Stdin)}
	_ = vmjobs.RegisterJob(j.String(), j)
}

func (j *sshJob) String() string {
	return "ssh"
}

func (j *sshJob) Desc() string {
	return "Connect with ssh to a vm and run commands until 'exit' is sent."
}

func (j *sshJob) Flags() []cli.Flag {
	return []cli.Flag{
		cli.StringSliceFlag{
			Name:     "image,i",
			Usage:    "VM image to run the command on. Only one allowed.",
			Required: true,
		},
		cli.BoolFlag{
			Name:  "exit-on-error",
			Usage: "Whether the job should exit at first failed command.",
		},
	}
}

func (j *sshJob) ParseCfg(c *cli.Context) error {
	images := c.StringSlice("image")
	if len(images) > 1 {
		return fmt.Errorf("%v job can only work on single image", j)
	}

	j.exitOnError = c.Bool("exit-on-error")
	return nil
}

func (j *sshJob) Cmd() (string, bool) {
	fmt.Printf("> ")
	if j.scanner.Scan() {
		suffix := "\n"
		if !j.exitOnError {
			suffix = " || true\n"
		}
		text := j.scanner.Text()
		if !strings.HasPrefix(text, "exit") {
			return text + suffix, true
		}
	}
	return "", false
}
