package vmjobs

import (
	"fmt"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli"
	"os"
	"strconv"
	"strings"
)

type bpfInfo struct {
	clang       string
	linux       string
	scap_built  bool
	probe_built bool
	res         string
}

type bpfJob struct {
	table       *tablewriter.Table
	checkoutCmd string
	images      []string
	bpfInfos    map[string]*bpfInfo
}

var bpfDefaultImages = []string{
	"generic/fedora33",
	"generic/fedora35",
	"ubuntu/focal64",
	"ubuntu/bionic64",
	"generic/debian10",
	"generic/centos8",
	"bento/amazonlinux-2",
}

func newBpfJob(c *cli.Context) (*bpfJob, error) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"VM", "Clang", "Linux", "Scap_built", "Probe_built", "Res"})

	var checkoutCmd string
	if c.IsSet("commithash") {
		checkoutCmd = fmt.Sprintf("git checkout %s", c.String("commithash"))
	}

	images := bpfDefaultImages
	if c.GlobalIsSet("images") {
		images = strings.Split(c.GlobalString("images"), ",")
	}
	return &bpfJob{
		table:       table,
		checkoutCmd: checkoutCmd,
		images:      images,
		bpfInfos:    initBpfInfoMap(images),
	}, nil
}

// Preinitialize map with meaningful values so that we will access it readonly,
// and there will be no need for concurrent access strategies
func initBpfInfoMap(images []string) map[string]*bpfInfo {
	bpfInfos := make(map[string]*bpfInfo)
	for _, image := range images {
		bpfInfos[image] = &bpfInfo{
			clang:       "N/A",
			linux:       "N/A",
			scap_built:  false,
			probe_built: false,
			res:         "N/A",
		}
	}
	return bpfInfos
}

func (j *bpfJob) Cmd() string {
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
            sudo apt install linux-headers-$(uname -r) git cmake build-essential pkg-config autoconf libtool libelf-dev llvm clang -y
            ;;
        centos|rhel|amzn) # OK generic/centos8, OK bento/amazonlinux-2
            sudo yum makecache
            sudo yum install gcc gcc-c++ kernel-devel-$(uname -r) git cmake pkg-config autoconf libtool elfutils-libelf-devel llvm clang -y
            ;;
        fedora) # OK generic/fedora33
            sudo dnf upgrade --refresh -y
            sudo dnf install gcc gcc-c++ kernel-headers git cmake pkg-config autoconf libtool elfutils-libelf-devel llvm clang -y
            ;;
        arch*) # OK generic/arch libvirt
            sudo pacman -S linux-headers --noconfirm # without -Sy to avoid installing kernel-headers for an updated, non-running, kernel version
            sudo pacman -Sy git cmake base-devel elfutils clang llvm --noconfirm
            ;;
        alpine) # OK generic/alpine314
            sudo apk update
            sudo apk add linux-virt-dev linux-headers g++ gcc cmake make git autoconf automake m4 libtool elfutils-dev libelf-static patch binutils clang llvm
            need_musl=1
            ;;    
        opensuse-*) # ??
            sudo zypper refresh
            sudo zypper -n install kernel-default-devel gcc gcc-c++ git-core cmake patch which automake autoconf libtool libelf-devel clang llvm
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
        cmake -DBUILD_BPF=ON -DUSE_BUNDLED_DEPS=on -DINSTALL_GTEST=off -DBUILD_GMOCK=off -DCREATE_TEST_TARGETS=off -DMUSL_OPTIMIZED_BUILD=On ../
    else 
        cmake -DBUILD_BPF=ON -DUSE_BUNDLED_DEPS=on -DINSTALL_GTEST=off -DBUILD_GMOCK=off -DCREATE_TEST_TARGETS=off ../
    fi

    make scap-open
	echo "BPF_VERIFIER_SCAP: true"

    make bpf
	echo "BPF_VERIFIER_PROBE: true"

	# Do not leave for verifier issues or timeout exit code (using "&& :")
    sudo BPF_PROBE=driver/bpf/probe.o timeout 5s ./libscap/examples/01-open/scap-open && :

    res=$?
    
    if [[ "$res" -eq "124" || "$res" -eq "143" ]]
    then
        # Timed out means no verifier issues.
        # See https://man7.org/linux/man-pages/man1/timeout.1.html
		# Some weird timeout version did not exit with 124 on timeout, 
		# but with 143 (ie: 128 + SIGTERM). Therefore, account for both.
		res=0
    fi
	echo "BPF_VERIFIER_TEST: $res"
}

set -e
need_musl=0
install_deps

echo "BPF_VERIFIER_CLANG: $(clang --version | head -n1 | awk -F' ' '{ print $3 }')"
echo "BPF_VERIFIER_LINUX: $(uname -r)"

build_and_run
`, j.checkoutCmd)
}

func (j *bpfJob) Images() []string {
	return j.images
}

func (j *bpfJob) Process(output VMOutput) {
	outputs := strings.Split(output.Line, ": ")
	info := j.bpfInfos[output.VM]
	switch outputs[0] {
	case "BPF_VERIFIER_CLANG":
		info.clang = outputs[1]
	case "BPF_VERIFIER_LINUX":
		info.linux = outputs[1]
	case "BPF_VERIFIER_SCAP":
		info.scap_built, _ = strconv.ParseBool(outputs[1])
	case "BPF_VERIFIER_PROBE":
		info.probe_built, _ = strconv.ParseBool(outputs[1])
	case "BPF_VERIFIER_TEST", "ERROR":
		info.res = outputs[1]
	}
}

func (j *bpfJob) Done() {
	for vm, info := range j.bpfInfos {
		j.table.Append([]string{vm, info.clang, info.linux,
			strconv.FormatBool(info.scap_built),
			strconv.FormatBool(info.probe_built),
			info.res})
	}
	j.table.Render()
}
