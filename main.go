package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"github.com/urfave/cli"
	"golang.org/x/sync/semaphore"
)

func defaultMemory() int {
	return 1024
}

func defaultParallelism() int {
	return runtime.NumCPU() / 2
}

func defaultNumCPUs() int {
	return runtime.NumCPU() / defaultParallelism()
}

func main() {
	app := cli.NewApp()
	app.Name = "vm-spinner"
	app.Usage = "Run your workloads on ephemeral Virtual Machines"
	app.Action = runApp
	app.UsageText = "vm-spinner [options...]"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:     "images,i",
			Usage:    "Comma-separated list of the VM image names to run the command on.",
			Required: true,
		},
		cli.StringFlag{
			Name:     "provider,p",
			Usage:    "Vagrant provider name.",
			Required: true,
		},
		cli.StringFlag{
			Name:  "filter",
			Usage: "If specified, only the output lines matching the filtering regex will be printed.",
		},
		cli.StringFlag{
			Name:  "cmdline,c",
			Usage: "The command that runs in each VM, specified as a command line parameter.",
		},
		cli.StringFlag{
			Name:  "cmdfile,f",
			Usage: "The command that runs in each VM, specified as a filepath.",
		},
		cli.BoolFlag{
			Name:  "cmdstdin",
			Usage: "The command that runs in each VM, specified through stdin.",
		},
		cli.IntFlag{
			Name:  "memory",
			Usage: "The amount of memory (in bytes) allocated for each VM.",
			Value: defaultMemory(),
		},
		cli.IntFlag{
			Name:  "cpus",
			Usage: "The number of cpus allocated for each VM.",
			Value: defaultNumCPUs(),
		},
		cli.IntFlag{
			Name:  "parallelism",
			Usage: "The number of VM to spawn in parallel.",
			Value: defaultParallelism(),
		},
		cli.BoolFlag{
			Name:  "info,I",
			Usage: "Include info-level lines in the output.",
		},
		cli.BoolFlag{
			Name:  "debug,D",
			Usage: "Include debug-level lines in the output.",
		},
	}

	app.Run(os.Args)
}

func validateParameters(c *cli.Context) error {
	if c.Int("cpus") > runtime.NumCPU() {
		return fmt.Errorf("number of CPUs for each VM (%d) exceeds the number of CPUs available (%d)", c.Int("cpus"), runtime.NumCPU())
	}

	if c.Int("parallelism") > runtime.NumCPU() {
		return fmt.Errorf("number of parallel VMs (%d) exceeds the number of CPUs available (%d)", c.Int("parallelism"), runtime.NumCPU())
	}

	if c.Int("parallelism")*c.Int("cpus") > runtime.NumCPU() {
		fmt.Printf("warning: number of parallel cpus (cpus * parallelism %d) exceeds the number of CPUs available (%d)\n", c.Int("parallelism")*c.Int("cpus"), runtime.NumCPU())
	}

	if len(c.String("cmdline")) == 0 && !c.Bool("cmdstdin") && len(c.String("cmdfile")) == 0 {
		return fmt.Errorf("one of the following must be specified: cmdline, cmdstdin, cmdfile")
	}

	return nil
}

func getCommand(c *cli.Context) (string, error) {
	errOverlap := fmt.Errorf("only one of the following must be specified: cmdline, cmdstdin, cmdfile")
	var cmd string

	if len(c.String("cmdline")) > 0 {
		cmd = c.String("cmdline")
	}

	if c.Bool("cmdstdin") {
		if len(cmd) > 0 {
			return "", errOverlap
		}
		cmd = ""
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			cmd += scanner.Text() + "\n"
		}
	}

	if len(c.String("cmdfile")) > 0 {
		if len(cmd) > 0 {
			return "", errOverlap
		}
		file, err := os.Open(c.String("cmdfile"))
		if err != nil {
			return "", err
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			cmd += scanner.Text() + "\n"
		}
	}

	return cmd, nil
}

func runApp(c *cli.Context) error {
	// Validate parameters
	err := validateParameters(c)
	if err != nil {
		return err
	}

	// read command to run on each VM
	command, err := getCommand(c)
	if err != nil {
		return err
	}

	// get the user-specified regex filter
	var filter *regexp.Regexp
	if len(c.String("filter")) > 0 {
		filter = regexp.MustCompile(c.String("filter"))
	}

	// prepare sync primitives.
	// the waitgrup is used to run all the VM in parallel, and to
	// join with each worker goroutine once their job is finished.
	// the semapthore is used to ensure that the parallelism upper
	// limit gets respected.
	var wg sync.WaitGroup
	sm := semaphore.NewWeighted(int64(c.Int("parallelism")))

	// iterate through all the specified VM images
	images := strings.Split(c.String("images"), ",")
	for i, image := range images {
		wg.Add(1)
		sm.Acquire(context.Background(), 1)
		imageName := image
		imageIndex := i

		// worker goroutine
		go func() {
			defer func() {
				sm.Release(1)
				wg.Done()
			}()

			// launch the VM for this image
			name := fmt.Sprintf("/tmp/%s-%d", imageName, imageIndex)
			conf := &VMConfig{
				Name:         name,
				BoxName:      imageName,
				ProviderName: c.String("provider"),
				CPUs:         c.Int("cpus"),
				Memory:       c.Int("memory"),
				Command:      command,
			}

			// select the VM outputs
			var line string
			channels := RunVirtualMachine(conf)
			for {
				line = ""
				select {
				case <-channels.Done:
					return
				case l := <-channels.CmdOutput:
					line = fmt.Sprintf("[OUTPUT %s] %s", name, l)
				case l := <-channels.Debug:
					if c.Bool("debug") {
						line = fmt.Sprintf("[DEBUG  %s] %s", name, l)
					}
				case l := <-channels.Info:
					if c.Bool("info") {
						line = fmt.Sprintf("[INFO   %s] %s", name, l)
					}
				case err := <-channels.Error:
					line = fmt.Sprintf("[ERROR  %s] %s", name, err.Error())
				}

				// print the line only if it matches the filter or if no filter is specified
				if len(line) > 0 && (filter == nil || filter.MatchString(err.Error())) {
					println(line)
				}
			}
		}()
	}

	// wait for all workers
	wg.Wait()
	return nil
}
