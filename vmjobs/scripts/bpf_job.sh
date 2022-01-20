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
    git clone https://github.com/%s/libs.git && cd libs
	  git checkout %s

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