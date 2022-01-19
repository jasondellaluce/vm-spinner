package vmjobs

import (
	"bufio"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"os"
)

func newScriptJob(c *cli.Context) (VMJob, error) {
	if !c.IsSet("file") {
		log.Fatalf("'file' argument required for script job.")
	}

	file, err := os.Open(c.String("file"))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var cmd string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		cmd += scanner.Text() + "\n"
	}

	return cmdLineJobFromCmd(cmd, c)
}
