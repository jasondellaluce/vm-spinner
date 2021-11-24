package main

import (
	"context"
	"fmt"
	"os"
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
	app.Action = run
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:     "cmd,c",
			Usage:    "The command that runs in each VM",
			Required: true,
		},
		cli.StringFlag{
			Name:     "images,i",
			Usage:    "Comma-separated list of the VM image names to run the command on",
			Required: true,
		},
		cli.StringFlag{
			Name:     "provider,p",
			Usage:    "Vagrant provider name",
			Required: true,
		},
		cli.IntFlag{
			Name:  "memory",
			Usage: "The amount of memory (in bytes) allocated for each VM",
			Value: defaultMemory(),
		},
		cli.IntFlag{
			Name:  "cpus",
			Usage: "The number of cpus allocated for each VM",
			Value: defaultNumCPUs(),
		},
		cli.IntFlag{
			Name:  "parallelism",
			Usage: "The number of VM to spawn in parallel",
			Value: defaultParallelism(),
		},
	}

	app.Run(os.Args)
}

func run(c *cli.Context) error {
	if c.Int("cpus") > runtime.NumCPU() {
		return fmt.Errorf("number of CPUs for each VM (%d) exceeds the number of CPUs available (%d)", c.Int("cpus"), runtime.NumCPU())
	}

	if c.Int("parallelism") > runtime.NumCPU() {
		return fmt.Errorf("number of parallel VMs (%d) exceeds the number of CPUs available (%d)", c.Int("parallelism"), runtime.NumCPU())
	}

	if c.Int("parallelism")*c.Int("cpus") >= runtime.NumCPU() {
		fmt.Printf("warning: number of parallel cpus (cpus * parallelism %d) exceeds the number of CPUs available (%d)\n", c.Int("parallelism")*c.Int("cpus"), runtime.NumCPU())
	}

	var wg sync.WaitGroup
	sm := semaphore.NewWeighted(int64(c.Int("parallelism")))
	ctx := context.Background()
	images := strings.Split(c.String("images"), ",")
	for i, image := range images {
		wg.Add(1)
		sm.Acquire(ctx, 1)
		imageName := image
		imageIndex := i
		go func() {
			defer func() {
				sm.Release(1)
				wg.Done()
			}()
			name := fmt.Sprintf("/tmp/%s-%d", imageName, imageIndex)
			conf := &VMConfig{
				Name:         name,
				BoxName:      imageName,
				ProviderName: c.String("provider"),
				CPUs:         c.Int("cpus"),
				Memory:       c.Int("memory"),
				Command:      c.String("cmd"),
			}

			channels := RunVirtualMachine(conf)
			for {
				select {
				case <-channels.Done:
					return
				case line := <-channels.CmdOutput:
					println(fmt.Sprintf("[OUTPUT %s] %s", name, line))
				case line := <-channels.Debug:
					println(fmt.Sprintf("[DEBUG  %s] %s", name, line))
				case line := <-channels.Info:
					println(fmt.Sprintf("[INFO   %s] %s", name, line))
				case err := <-channels.Error:
					println(fmt.Sprintf("[ERROR  %s] %s", name, err.Error()))
				}
			}
		}()
	}

	wg.Wait()
	return nil
}
