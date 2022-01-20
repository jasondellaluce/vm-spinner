# vm-spinner

Run your workloads on ephemeral Virtual Machines.

### Descriprion

A simple tool that spawns an arbitrary number of VMs in parallel, runs the same workload on each of them, and collects their outputs.

This requires [Vagrant](https://www.vagrantup.com/) to be installed in your system, and to be properly configured with a supported provider.

### Examples
Printing `hello world` on an Ubuntu 20.04 VM using VirtualBox (default provider):
```bash
./vm-spinner cmd --line "echo hello world" -i "ubuntu/focal64"
```

Creating a VM and installing Docker.
```bash
./vm-spinner cmd --line "curl -fsSL https://get.docker.com -o get-docker.sh && sh ./get-docker.sh" -i "ubuntu/focal64"
```

Running a local script in two VM in parallel, by specifying the provisioned resources for each VM:
```bash
./vm-spinner --cpus=2 --parallelism=2 --memory=4096 script --file "./script.sh" -i "ubuntu/focal64" -i "ubuntu/bionic64"
```
