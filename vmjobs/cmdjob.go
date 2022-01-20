package vmjobs

import (
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

type cmdLineJob struct {
	cmd string
}

func newCmdLineJob(c *cli.Context) (VMJob, error) {
	return cmdLineJobFromCmd(c.String("line"))
}

func cmdLineJobFromCmd(cmd string) (VMJob, error) {
	if len(cmd) == 0 {
		log.Fatalf("provided command is empty.")
	}
	return &cmdLineJob{cmd: cmd}, nil
}

func (j *cmdLineJob) Cmd() string {
	return j.cmd
}

func (j *cmdLineJob) Process(VMOutput) {

}

func (j *cmdLineJob) Done() {

}
