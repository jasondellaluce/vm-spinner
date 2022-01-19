package vmjobs

import (
	"fmt"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli"
	"os"
	"strconv"
	"strings"
)

type kmodInfo struct {
	gcc        string
	linux      string
	kmod_built bool
}

type kmodJob struct {
	table       *tablewriter.Table
	checkoutCmd string
	images      []string
	kmodInfos   map[string]*kmodInfo
}

var kmodDefaultImages = []string{
	"generic/fedora33",
	"generic/fedora35",
	"ubuntu/focal64",
	"ubuntu/bionic64",
	"generic/debian10",
	"generic/centos8",
	"bento/amazonlinux-2",
}

func newKmodJob(c *cli.Context) (*kmodJob, error) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"VM", "GCC", "Linux", "Kmod_built"})

	var checkoutCmd string
	if c.IsSet("commithash") {
		checkoutCmd = fmt.Sprintf("git checkout %s", c.String("commithash"))
	}

	images := kmodDefaultImages
	if c.GlobalIsSet("images") {
		images = strings.Split(c.GlobalString("images"), ",")
	}
	return &kmodJob{
		table:       table,
		checkoutCmd: checkoutCmd,
		images:      images,
		kmodInfos:   initKmodInfoMap(images),
	}, nil
}

// Preinitialize map with meaningful values so that we will access it readonly,
// and there will be no need for concurrent access strategies
func initKmodInfoMap(images []string) map[string]*kmodInfo {
	kmodInfos := make(map[string]*kmodInfo)
	for _, image := range images {
		kmodInfos[image] = &kmodInfo{
			gcc:        "N/A",
			linux:      "N/A",
			kmod_built: false,
		}
	}
	return kmodInfos
}

func (j *kmodJob) Cmd() string {
	return fmt.Sprintf(`
#!/bin/sh

get_distribution() {
    lsb_dist=""
    # Every system that we officially support has /etc/os-release
    if [ -r /etc/os-release ]; then
        lsb_dist="$(. /etc/os-release && echo "$ID")"
    fi
    # Returning an empty string here should be alright since the
    # case statements don't act unless you provide an actual value
    echo "$lsb_dist"
}

install_deps() {
    lsb_dist=$( get_distribution )
    lsb_dist="$(echo "$lsb_dist" | tr '[:upper:]' '[:lower:]')"

    case "$lsb_dist" in
        ubuntu|debian) # OK ubuntu/focal64, OK ubuntu/bionic64, OK generic/debian10
            sudo apt update
            sudo apt install linux-headers-$(uname -r) git cmake build-essential pkg-config autoconf libtool libelf-dev -y
            ;;
        centos|rhel|amzn) # OK generic/centos8, OK bento/amazonlinux-2
            sudo yum makecache
            sudo yum install gcc gcc-c++ kernel-devel-$(uname -r) git cmake pkg-config autoconf libtool elfutils-libelf-devel -y
            ;;
        fedora) # OK generic/fedora33
            sudo dnf upgrade --refresh -y
            sudo dnf install gcc gcc-c++ kernel-headers git cmake pkg-config autoconf libtool elfutils-libelf-devel -y
            ;;
        arch*) # OK generic/arch libvirt
            sudo pacman -S linux-headers --noconfirm # without -Sy to avoid installing kernel-headers for an updated, non-running, kernel version
            sudo pacman -Sy git cmake base-devel elfutils --noconfirm
            ;;
        alpine) # OK generic/alpine314
            sudo apk update
            sudo apk add linux-virt-dev linux-headers g++ gcc cmake make git autoconf automake m4 libtool elfutils-dev libelf-static patch binutils
            need_musl=1
            ;;    
        opensuse-*) # ??
            sudo zypper refresh
            sudo zypper -n install kernel-default-devel gcc gcc-c++ git-core cmake patch which automake autoconf libtool libelf-devel
            ;;
        *)
            echo
            echo "ERROR: Unsupported distribution '$lsb_dist'"
            echo
            exit 1
            ;;
    esac
}

build_and_run() {
    git clone https://github.com/falcosecurity/libs.git && cd libs
	%s
    mkdir build && cd build

    if [ "$need_musl" -eq "1" ]
    then
        cmake -DUSE_BUNDLED_DEPS=on -DINSTALL_GTEST=off -DBUILD_GMOCK=off -DCREATE_TEST_TARGETS=off -DMUSL_OPTIMIZED_BUILD=On ../
    else 
        cmake -DUSE_BUNDLED_DEPS=on -DINSTALL_GTEST=off -DBUILD_GMOCK=off -DCREATE_TEST_TARGETS=off ../
    fi

    make driver
	echo "KMOD_DRIVER_BUILT: true"
}

set -e
need_musl=0
install_deps

echo "KMOD_DRIVER_GCC: $(gcc --version | head -n1 | awk -F' ' '{ print $3 }')"
echo "KMOD_DRIVER_LINUX: $(uname -r)"

build_and_run
`, j.checkoutCmd)
}

func (j *kmodJob) Images() []string {
	return j.images
}

func (j *kmodJob) Process(output VMOutput) {
	outputs := strings.Split(output.Line, ": ")
	info := j.kmodInfos[output.VM]
	switch outputs[0] {
	case "KMOD_DRIVER_GCC":
		info.gcc = outputs[1]
	case "KMOD_DRIVER_LINUX":
		info.linux = outputs[1]
	case "KMOD_DRIVER_BUILT":
		info.kmod_built, _ = strconv.ParseBool(outputs[1])
	}
}

func (j *kmodJob) Done() {
	for vm, info := range j.kmodInfos {
		j.table.Append([]string{vm, info.gcc, info.linux,
			strconv.FormatBool(info.kmod_built)})
	}
	j.table.Render()
}
