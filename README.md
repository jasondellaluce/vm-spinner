# vm-spinner

Run your workloads on ephemeral Virtual Machines.

## Description

A simple tool that spawns an arbitrary number of VMs in parallel, runs the same workload on each of them, and collects their outputs.

This requires [Vagrant](https://www.vagrantup.com/) to be installed in your system, and to be properly configured with a supported provider.

## Jobs

vm-spinner uses so-called `jobs` to do its magic.  
Jobs implements a `VMJob` interface that defines their name, description, and command to be run.  
Moreover, there are other 2 interfaces that might be implemented:  
* `VMJobProcessor`: to embed private logic to process output from command being run
* `VMJobConfigurator`: to embed private logic to define and parse plugin specific flags. This adds an hard dep on `github.com/urfave/cli` package.  

All these interfaces can be found in the [vmjob](pkg/vmjobs/vmjob.go) file.

Finally, vm-spinner also supports external plugins; they are go shared objects that implement the `VMJob` interface(and eventually `VMJobProcessor` and `VMJobConfigurator`),   
and expose a `PluginJob` var.  
Here is a simple example:

```go
package main

type myJob struct {
	cmd string
}

// PluginJob symbol needs to be exported because it will be loaded by plugin framework
var PluginJob myJob

func (j *myJob) String() string {
	return "testplugin"
}

func (j *myJob) Desc() string {
	return "Run a simple plugin job."
}

func (j *myJob) Cmd() (string, bool) {
	return `echo "I am a plugin"`, false
}
```

You can see that the implementation is fairly simple.  
Just a couple of things to note:

* String() returns plugin name. It **must** be unique foreach plugin
* When `VMJobConfigurator` interface is not implemented, or if the list of plugin flags does not contain an `image,i` flag, a default image flag is enforced by the framework

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
