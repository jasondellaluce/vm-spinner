package vmjobs

import (
	"bufio"
	"github.com/urfave/cli"
	"os"
)

func newScriptJob(c *cli.Context) (VMJob, error) {
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

	return cmdLineJobFromCmd(cmd)
}
