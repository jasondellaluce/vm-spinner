package vmjobs

import (
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"strings"
)

type cmdLineJob struct {
	cmd    string
	images []string
}

func newCmdLineJob(c *cli.Context) (VMJob, error) {
	if !c.IsSet("line") {
		log.Fatalf("'line' argument required for cmd job.")
	}
	return cmdLineJobFromCmd(c.String("line"), c)
}

func cmdLineJobFromCmd(cmd string, c *cli.Context) (VMJob, error) {
	if !c.GlobalIsSet("images") {
		log.Fatalf("'images' argument required for cmd job.")
	}
	return &cmdLineJob{cmd: cmd, images: strings.Split(c.GlobalString("images"), ",")}, nil
}

func (j *cmdLineJob) Cmd() string {
	return j.cmd
}

func (j *cmdLineJob) Images() []string {
	return j.images
}

func (j *cmdLineJob) Process(VMOutput) {

}

func (j *cmdLineJob) Done() {

}
