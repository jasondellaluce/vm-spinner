package main

import (
	"fmt"
	"os"

	"github.com/koding/vagrantutil"
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
	Name         string
	BoxName      string
	ProviderName string
	Memory       int
	CPUs         int
	Command      string
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

func RunVirtualMachine(conf *VMConfig) *VMChannels {
	output := make(chan string)
	debug := make(chan string)
	info := make(chan string)
	err := make(chan error)
	done := make(chan bool)

	go func() {
		vagrantErr := runVagrantMachine(conf, output, debug, info)
		if vagrantErr != nil {
			sendErr(err, vagrantErr)
		}
		done <- true
		close(done)
		close(output)
		close(debug)
		close(info)
		close(err)
		os.RemoveAll(conf.Name)
	}()

	return &VMChannels{
		CmdOutput: output,
		Debug:     debug,
		Info:      info,
		Error:     err,
		Done:      done,
	}
}

func runVagrantMachine(conf *VMConfig, output, debug, info chan<- string) (resErr error) {
	isUp := false

	// Create Vagrant config file
	sendStr(debug, "Initializing Vagrant configuration for '"+conf.Name+"'")
	vagrant, err := vagrantutil.NewVagrant(conf.Name)
	if err != nil {
		resErr = err
		return
	}

	// Create Vagrant VM
	sendStr(debug, "Creating Vagrant VM  for '"+conf.Name+"' on '"+conf.ProviderName+"' provider")
	vagrantfile := fmt.Sprintf(
		fmtVagrantfile,
		conf.BoxName,
		conf.ProviderName,
		conf.Memory,
		conf.CPUs,
	)
	err = vagrant.Create(vagrantfile)
	if err != nil {
		resErr = err
		return
	}

	// Once the VM is created, we need to destroy it.
	// If the VM has been started, it must be halted first
	defer func() {
		if isUp {
			sendStr(debug, "Halting Vagrant VM for '"+conf.Name+"'")
			halt, err := vagrant.Halt()
			if err != nil {
				resErr = err
				return
			}
			for line := range halt {
				if line.Error != nil {
					resErr = line.Error
					return
				}
				sendStr(info, line.Line)
			}
		}

		sendStr(debug, "Destroying Vagrant VM for '"+conf.Name+"'")
		destroy, err := vagrant.Destroy()
		if err != nil {
			resErr = err
			return
		}
		for line := range destroy {
			if line.Error != nil {
				resErr = line.Error
				return
			}
			sendStr(info, line.Line)
		}
	}()

	// Start up the VM
	sendStr(debug, "Starting Vagrant VM for '"+conf.Name+"'")
	up, err := vagrant.Up()
	if err != nil {
		resErr = err
		return
	}
	isUp = true
	for line := range up {
		if line.Error != nil {
			resErr = line.Error
			return
		}
		sendStr(info, line.Line)
	}

	// Establish a SSH connection and run command
	sendStr(debug, "Running command with SSH for '"+conf.Name+"'")
	ssh, err := vagrant.SSH(conf.Command)
	if err != nil {
		resErr = err
		return
	}
	for line := range ssh {
		sendStr(output, line.Line)
	}

	return
}