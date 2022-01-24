# vm-spinner

Run your workloads on ephemeral Virtual Machines.

## Description

A simple tool that spawns an arbitrary number of VMs in parallel, runs the same workload on each of them, and collects their outputs.

This requires [Vagrant](https://www.vagrantup.com/) to be installed in your system, and to be properly configured with a supported provider.

## Jobs

vm-spinner uses so-called `jobs` to do its magic.  
Jobs implements a `VMJob` interface that defines their cmdline flags, name and description.  
Plus, they embed their private logic to pass ssh commands to the vms and parse their outputs.  

Moreover, vm-spinner also supports external plugins; they are go shared objects that implement the `VMJob` interface,  
and expose a `PluginJob` var.  
Here is a simple example:
```go
package main

import (
	"bufio"
	"os"
    	"github.com/urfave/cli"
)

type myJob struct {
	cmd string
}

var PluginJob myJob

func (j *myJob) String() string {
	return "testplugin"
}

func (j *myJob) Desc() string {
	return "Run a simple plugin job."
}

func (j *myJob) Flags() []cli.Flag {
	return nil
}

func (j *myJob) ParseCfg(_ *cli.Context) error {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		j.cmd += scanner.Text() + "\n"
	}
	return nil
}

func (j *myJob) Cmd() string {
	return j.cmd
}

func (j *myJob) Process(_, _ string) {}

func (j *myJob) Done() {}
```

You can see that the implementation is fairly simple.  
Just a couple of things to note:

* String() returns plugin name. It **must** be unique foreach plugin
* `github.com/urfave/cli` package is an hard dep
* When `nil` flags are returned, or if the list of flags does not contain an `image,i` flag, a default image flag is forced

### Examples

* Printing `hello world` on an Ubuntu 20.04 VM using VirtualBox (default provider):
```bash
vm-spinner cmd --line "echo hello world" -i "ubuntu/focal64"
```

* Creating a VM and installing Docker.
```bash
vm-spinner cmd --line "curl -fsSL https://get.docker.com -o get-docker.sh && sh ./get-docker.sh" -i "ubuntu/focal64"
```

* Running a local script in two VM in parallel, by specifying the provisioned resources for each VM:
```bash
vm-spinner --cpus=2 --parallelism=2 --memory=4096 cmd --file "./script.sh" -i "ubuntu/focal64" -i "ubuntu/bionic64"
```

* Running a plugin:
```bash
vm-spinner --plugin-dir /$HOME/plugins/ testplugin -i "ubuntu/focal64"
```
