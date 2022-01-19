package vmjobs

import (
	"bufio"
	"github.com/urfave/cli"
	"os"
)

func newStdinJob(c *cli.Context) (VMJob, error) {
	var cmd string
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		cmd += scanner.Text() + "\n"
	}
	return cmdLineJobFromCmd(cmd, c)
}
