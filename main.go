package main

import (
	"context"
	"fmt"
	"github.com/jasondellaluce/experiments/vm-spinner/vmjobs"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"

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
	// Each sub-command has its own "image" parameter, because some command
	// has a default value, therefore not needing a required flag,
	// while others have no default values.
	// Moreover, command may be run in parallel, therefore it is desired
	// to be able to specify different images for each job.
	app.Commands = []cli.Command{
		{
			Name:   vmjobs.VMJobBpf,
			Usage:  "Run bpf build + verifier job.",
			Action: runApp,
			Flags: []cli.Flag{
				cli.StringSliceFlag{
					Name:  "image,i",
					Usage: "VM image to run the command on. Specify it multiple times for multiple vms.",
					Value: &vmjobs.BpfDefaultImages,
				},
				cli.StringFlag{
					Name:  "forkname",
					Usage: "libs fork to clone from.",
					Value: "falcosecurity",
				},
				cli.StringFlag{
					Name:  "commithash",
					Usage: "libs commit hash to run the test against.",
					Value: "master",
				},
			},
		},
		{
			Name:   vmjobs.VMJobKmod,
			Usage:  "Run kmod build job.",
			Action: runApp,
			Flags: []cli.Flag{
				cli.StringSliceFlag{
					Name:  "image,i",
					Usage: "VM image to run the command on. Specify it multiple times for multiple vms.",
					Value: &vmjobs.KmodDefaultImages,
				},
				cli.StringFlag{
					Name:  "forkname",
					Usage: "libs fork to clone from.",
					Value: "falcosecurity",
				},
				cli.StringFlag{
					Name:  "commithash",
					Usage: "libs commit hash to run the test against.",
					Value: "master",
				},
			},
		},
		{
			Name:   vmjobs.VMJobCmd,
			Usage:  "Run a simple cmd line job.",
			Action: runApp,
			Flags: []cli.Flag{
				cli.StringSliceFlag{
					Name:     "image,i",
					Usage:    "VM image to run the command on. Specify it multiple times for multiple vms.",
					Required: true,
				},
				cli.StringFlag{
					Name:     "line",
					Usage:    "command that runs in each VM, as a command line parameter.",
					Required: true,
				},
			},
		},
		{
			Name:   vmjobs.VMJobStdin,
			Usage:  "Run a simple cmd line job read from stdin.",
			Action: runApp,
			Flags: []cli.Flag{
				cli.StringSliceFlag{
					Name:     "image,i",
					Usage:    "VM image to run the command on. Specify it multiple times for multiple vms.",
					Required: true,
				},
			},
		},
		{
			Name:   vmjobs.VMJobScript,
			Usage:  "Run a simple script job read from file.",
			Action: runApp,
			Flags: []cli.Flag{
				cli.StringSliceFlag{
					Name:     "image,i",
					Usage:    "VM image to run the command on. Specify it multiple times for multiple vms.",
					Required: true,
				},
				cli.StringFlag{
					Name:     "file",
					Usage:    "script that runs in each VM, as a filepath.",
					Required: true,
				},
			},
		},
	}
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "provider,p",
			Usage: "Vagrant provider name.",
			Value: "virtualbox",
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
			Name:  "log.json",
			Usage: "Whether to log output in json format.",
		},
		cli.StringFlag{
			Name:  "log.level",
			Usage: "Log level, between { trace, debug, info, error }.",
			Value: "debug",
		},
		cli.StringFlag{
			Name:  "log.output",
			Usage: "Log output filename. If empty, stdout will be used.",
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func validateParameters(c *cli.Context) error {
	if c.GlobalInt("cpus") > runtime.NumCPU() {
		return fmt.Errorf("number of CPUs for each VM (%d) exceeds the number of CPUs available (%d)", c.Int("cpus"), runtime.NumCPU())
	}

	if c.GlobalInt("parallelism") > runtime.NumCPU() {
		return fmt.Errorf("number of parallel VMs (%d) exceeds the number of CPUs available (%d)", c.Int("parallelism"), runtime.NumCPU())
	}

	if c.GlobalInt("parallelism")*c.GlobalInt("cpus") > runtime.NumCPU() {
		fmt.Printf("warning: number of parallel cpus (cpus * parallelism %d) exceeds the number of CPUs available (%d)\n", c.Int("parallelism")*c.Int("cpus"), runtime.NumCPU())
	}

	return nil
}

func initLog(c *cli.Context) error {
	// Log as JSON instead of the default ASCII formatter.
	if c.GlobalBool("log.json") {
		log.SetFormatter(&log.JSONFormatter{})
	}

	out := os.Stdout
	if len(c.GlobalString("log.output")) > 0 {
		var err error
		out, err = os.OpenFile(c.GlobalString("log-output"), os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
	}
	log.SetOutput(out)

	switch c.GlobalString("log.level") {
	case "trace":
		log.SetLevel(log.TraceLevel)
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	default:
		log.SetLevel(log.DebugLevel)
	}
	return nil
}

func runApp(c *cli.Context) error {
	err := validateParameters(c)
	if err != nil {
		return err
	}

	err = initLog(c)
	if err != nil {
		return err
	}

	job, err := vmjobs.NewVMJob(c)
	if err != nil {
		log.Fatal(err)
	}

	// Goroutine to handle result in job plugin
	var resWg sync.WaitGroup
	resCh := make(chan vmjobs.VMOutput)
	resWg.Add(1)
	go func() {
		for res := range resCh {
			job.Process(res)
		}
		resWg.Done()
	}()

	// Unlock sm.Acquire() call killing its context on external signals, allowing us
	// to avoid situations when some images are waiting on sm.Acquire() call,
	// and current images gets killed by an external signal (managed in vagrant.go),
	// we proceed to process subsequent images because main thread did not notice anything.
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	// prepare sync primitives.
	// the waitgrup is used to run all the VM in parallel, and to
	// join with each worker goroutine once their job is finished.
	// the semapthore is used to ensure that the parallelism upper
	// limit gets respected.
	var wg sync.WaitGroup
	sm := semaphore.NewWeighted(int64(c.GlobalInt("parallelism")))

	images := c.StringSlice("image")
	log.Infof("Running on %v images", images)
	for i, image := range images {
		smErr := sm.Acquire(ctx, 1)
		// Acquire may return non-nil err even if ctx.Done() is triggered
		if smErr != nil || ctx.Err() != nil {
			break
		}

		wg.Add(1)

		// launch the VM for this image
		name := fmt.Sprintf("/tmp/%s-%d", image, i)
		conf := &VMConfig{
			Name:         name,
			BoxName:      image,
			ProviderName: c.GlobalString("provider"),
			CPUs:         c.GlobalInt("cpus"),
			Memory:       c.GlobalInt("memory"),
			Command:      job.Cmd(),
		}

		// worker goroutine
		go func() {
			defer func() {
				sm.Release(1)
				wg.Done()
			}()

			// select the VM outputs
			channels := RunVirtualMachine(ctx, conf)
			for {
				logger := log.WithFields(log.Fields{"vm": conf.BoxName})
				select {
				case <-channels.Done:
					logger.Info("Job Finished.")
					return
				case l := <-channels.CmdOutput:
					logger.Info(l)
					resCh <- vmjobs.VMOutput{VM: conf.BoxName, Line: l}
				case l := <-channels.Debug:
					logger.Trace(l)
				case l := <-channels.Info:
					logger.Debug(l)
				case err := <-channels.Error:
					logger.Error(err.Error())
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

	// Notify job that we're done
	job.Done()

	return nil
}
