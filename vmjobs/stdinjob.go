package vmjobs

import (
	"bufio"
	"os"
)

func newStdinJob() (VMJob, error) {
	var cmd string
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		cmd += scanner.Text() + "\n"
	}
	return cmdLineJobFromCmd(cmd)
}
