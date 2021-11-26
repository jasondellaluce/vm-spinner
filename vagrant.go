package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

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

func destroyVagrantMachine(vagrant *vagrantutil.Vagrant, conf *VMConfig, debug, info chan<- string) error {
	sendStr(debug, "Destroying Vagrant VM for '"+conf.Name+"'")
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
	sendStr(debug, "Halting Vagrant VM for '"+conf.Name+"'")
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

func runVagrantMachine(conf *VMConfig, output, debug, info chan<- string) (resErr error) {
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
	defer func() {
		resErr = destroyVagrantMachine(vagrant, conf, debug, info)
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Start up the VM
	sendStr(debug, "Starting Vagrant VM for '"+conf.Name+"'")
	up, err := vagrant.Up()
	if err != nil {
		resErr = err
		return
	}
	defer func() {
		resErr = haltVagrantMachine(vagrant, conf, debug, info)
	}()

	killedBySignal := selectHandleSig(up, sigs, info)
	if killedBySignal {
		return
	}

	// Establish a SSH connection and run command
	sendStr(debug, "Running command with SSH for '"+conf.Name+"'")
	ssh, err := vagrant.SSH(conf.Command)
	if err != nil {
		resErr = err
		return
	}

	_ = selectHandleSig(ssh, sigs, output)
	return
}

// selectHandleSig returns whether a signal was received
func selectHandleSig(ch <-chan *vagrantutil.CommandOutput, sigCh chan os.Signal, out chan<- string) bool {
	for {
		select {
		case line, ok := <-ch:
			if !ok {
				return false
			}
			sendStr(out, line.Line)
		case <-sigCh:
			return true
		}
	}
}