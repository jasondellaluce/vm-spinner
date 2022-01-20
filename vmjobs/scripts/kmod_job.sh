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
    git clone https://github.com/%s/libs.git && cd libs
	  git checkout %s

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