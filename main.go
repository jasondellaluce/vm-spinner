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

	"github.com/olekukonko/tablewriter"
	log "github.com/sirupsen/logrus"
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
			Name:  "log-json",
			Usage: "Whether to log output in json format.",
		},
		cli.StringFlag{
			Name:  "log-level",
			Usage: "Log level, between { trace, debug, info }. Defaults to debug.",
		},
		cli.StringFlag{
			Name:  "log-output",
			Usage: "Log output filename; by default stdout.",
		},
		cli.BoolFlag{
			Name:  "summary-matrix",
			Usage: "Print a summary matrix using the filtered (through --filter) line as results for each vm.",
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
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

	if c.Bool("summary-matrix") && len(c.String("filter")) == 0 {
		return fmt.Errorf("'--summary-matrix' requires '--filter' option")
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
			text := scanner.Text()
			// Skip shabang if a full script was used
			if !strings.HasPrefix(text, "#!") {
				cmd += scanner.Text() + "\n"
			}
		}
	}

	return cmd, nil
}

func initLog(c *cli.Context) error {
	// Log as JSON instead of the default ASCII formatter.
	if c.Bool("log-json") {
		log.SetFormatter(&log.JSONFormatter{})
	}

	out := os.Stdout
	if len(c.String("log-output")) > 0 {
		var err error
		out, err = os.Open(c.String("output"))
		if err != nil {
			return err
		}
	}
	log.SetOutput(out)

	switch c.String("log-level") {
	case "trace":
		log.SetLevel(log.TraceLevel)
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	default:
		log.SetLevel(log.DebugLevel)
	}
	return nil
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

	err = initLog(c)
	if err != nil {
		return err
	}

	// get the user-specified regex filter
	var filter *regexp.Regexp
	if len(c.String("filter")) > 0 {
		filter = regexp.MustCompile(c.String("filter"))
	}

	// Goroutine to handle result summary matrix, if needed
	var resWg sync.WaitGroup
	resCh := make(chan []string)
	if c.Bool("summary-matrix") {
		resWg.Add(1)
		go func() {
			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"VM", "RES"})
			for res := range resCh {
				table.Append(res)
			}
			table.Render() // Send output
			resWg.Done()
		}()
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
			channels := RunVirtualMachine(conf)
			for {
				logger := log.WithFields(log.Fields{"vm": name})
				var l string
				var lvl log.Level
				select {
				case <-channels.Done:
					return
				case l = <-channels.CmdOutput:
					lvl = log.InfoLevel
				case l = <-channels.Debug:
					lvl = log.TraceLevel
				case l = <-channels.Info:
					lvl = log.DebugLevel
				case err = <-channels.Error:
					lvl = log.ErrorLevel
					l = err.Error()
				}

				// print the line only if it matches the filter or if no filter is specified
				if len(l) > 0 && (filter == nil || filter.MatchString(l)) {
					logger.Log(lvl, l)

					if filter != nil && c.Bool("summary-matrix") {
						resCh <- []string{name, l }
					}
				}
			}
		}()
	}

	// wait for all workers
	wg.Wait()

	// Close summary matrix channel and wait
	// for it to eventually print the summary
	close(resCh)
	resWg.Wait()

	return nil
}
