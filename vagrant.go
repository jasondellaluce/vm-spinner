package main

import (
	"context"
	"fmt"
	"github.com/jasondellaluce/experiments/vm-spinner/vmjobs"
	"github.com/koding/vagrantutil"
	"os"
)

const fmtVagrantfile = `
Vagrant.configure("2") do |config|
  config.vm.box = "%s"
  config.vm.synced_folder ".", "/vagrant", disabled: true
  config.vm.provider "%s" do |vb|
    vb.memory = "%d"
    vb.cpus = "%d"
  end
end
`

type VMConfig struct {
	Path         string
	BoxName      string
	ProviderName string
	Memory       int
	CPUs         int
	Job          vmjobs.VMJob
}

type VMChannels struct {
	CmdOutput <-chan string
	Debug     <-chan string
	Info      <-chan string
	Error     <-chan error
	Done      <-chan bool
}

func sendStr(c chan<- string, v string) {
	select {
	case c <- v:
	default:
	}
}

func sendErr(c chan<- error, v error) {
	select {
	case c <- v:
	default:
	}
}

func RunVirtualMachine(ctx context.Context, conf *VMConfig) *VMChannels {
	output := make(chan string)
	debug := make(chan string)
	info := make(chan string)
	err := make(chan error)
	done := make(chan bool)

	go func() {
		vagrantErr := runVagrantMachine(ctx, conf, output, debug, info)
		if vagrantErr != nil {
			sendErr(err, vagrantErr)
		}
		done <- true
		close(done)
		close(output)
		close(debug)
		close(info)
		close(err)
		os.RemoveAll(conf.Path)
	}()

	return &VMChannels{
		CmdOutput: output,
		Debug:     debug,
		Info:      info,
		Error:     err,
		Done:      done,
	}
}

func destroyVagrantMachine(vagrant *vagrantutil.Vagrant, conf *VMConfig, debug, info chan<- string) error {
	sendStr(debug, "Destroying Vagrant VM for '"+conf.BoxName+"'")
	destroy, err := vagrant.Destroy()
	if err != nil {
		return err
	}
	for line := range destroy {
		if line.Error != nil {
			return line.Error
		}
		sendStr(info, line.Line)
	}
	return nil
}

func haltVagrantMachine(vagrant *vagrantutil.Vagrant, conf *VMConfig, debug, info chan<- string) error {
	sendStr(debug, "Halting Vagrant VM for '"+conf.BoxName+"'")
	halt, err := vagrant.Halt()
	if err != nil {
		return err
	}
	for line := range halt {
		if line.Error != nil {
			return line.Error
		}
		sendStr(info, line.Line)
	}
	return nil
}

func runVagrantMachine(ctx context.Context, conf *VMConfig, output, debug, info chan<- string) (resErr error) {
	var (
		vagrant *vagrantutil.Vagrant
		up      <-chan *vagrantutil.CommandOutput
	)
	// Create Vagrant config file
	sendStr(debug, "Initializing Vagrant configuration for '"+conf.BoxName+"'")
	vagrant, resErr = vagrantutil.NewVagrant(conf.Path)
	if resErr != nil {
		return
	}

	// Create Vagrant VM
	sendStr(debug, "Creating Vagrant VM  for '"+conf.BoxName+"' on '"+conf.ProviderName+"' provider")
	vagrantfile := fmt.Sprintf(
		fmtVagrantfile,
		conf.BoxName,
		conf.ProviderName,
		conf.Memory,
		conf.CPUs,
	)
	resErr = vagrant.Create(vagrantfile)
	if resErr != nil {
		return
	}
	defer func() {
		resErr = destroyVagrantMachine(vagrant, conf, debug, info)
	}()

	// Start up the VM
	sendStr(debug, "Starting Vagrant VM for '"+conf.BoxName+"'")
	up, resErr = vagrant.Up()
	if resErr != nil {
		return
	}
	defer func() {
		resErr = haltVagrantMachine(vagrant, conf, debug, info)
	}()

	killedBySignal := selectHandleSig(up, ctx, info)
	if killedBySignal {
		return
	}

	// Establish an SSH connection and run command
	sendStr(debug, "Running command with SSH for '"+conf.BoxName+"'")

	for {
		cmd, hasMore := conf.Job.Cmd()
		killedBySignal, resErr = callSSHCmd(ctx, vagrant, cmd, output)
		if !hasMore || killedBySignal || resErr != nil {
			break
		}
	}
	return
}

func callSSHCmd(ctx context.Context, vagrant *vagrantutil.Vagrant, cmd string, output chan<- string) (bool, error) {
	ssh, err := vagrant.SSH(cmd)
	if err != nil {
		return false, err
	}
	return selectHandleSig(ssh, ctx, output), nil
}

// selectHandleSig returns whether ctx done was received
func selectHandleSig(ch <-chan *vagrantutil.CommandOutput, ctx context.Context, out chan<- string) bool {
	for {
		select {
		case line, ok := <-ch:
			if !ok {
				return false
			}
			sendStr(out, line.Line)
		case <-ctx.Done():
			return true
		}
	}
}
