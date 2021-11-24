# vm-spinner

Run your workloads on ephemeral Virtual Machines.

### Descriprion

A simple tool that spawns an arbitrary number of VMs in parallel, runs the same workload on each of them, and collects their outputs.

This requires [Vagrant](https://www.vagrantup.com/) to be installed in your system, and to be properly configured with a supported provider.

### Examples
Printing `hello world` on an Ubuntu 20.04 VM using VirtualBox:
```bash
./vm-spinner -p "virtualbox" -i "ubuntu/focal64" -c "echo hello world" 
```

Creating a VM and installing Docker.
```bash
./vm-spinner -p "virtualbox" -i "ubuntu/focal64" -c "curl -fsSL https://get.docker.com -o get-docker.sh && sh ./get-docker.sh"   
```

Running a local script in two VM in parallel, by specifying the provisioned resources for each VM:
```bash
./vm-spinner -p "virtualbox" -i "ubuntu/focal64;ubuntu/bionic64" -f "./script.sh" --cpus=2 --parallelism=2 --memory=4096  
```