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
            sudo apt install linux-headers-"$(uname -r)" git cmake build-essential pkg-config autoconf libtool libelf-dev -y
            if [ "$is_bpf" = true ]
            then
                sudo apt install llvm clang -y
            fi
            ;;
        centos|rhel|amzn) # OK generic/centos8, OK bento/amazonlinux-2
            sudo yum makecache
            sudo yum install gcc gcc-c++ kernel-devel-"$(uname -r)" git cmake pkg-config autoconf libtool elfutils-libelf-devel llvm clang -y
            if [ "$is_bpf" = true ]
            then
                sudo yum install llvm clang -y
            fi
            ;;
        fedora) # OK generic/fedora33
            sudo dnf upgrade --refresh -y
            sudo dnf install gcc gcc-c++ kernel-headers git cmake pkg-config autoconf libtool elfutils-libelf-devel llvm clang -y
            if [ "$is_bpf" = true ]
            then
                sudo dnf install llvm clang -y
            fi
            ;;
        arch*) # OK generic/arch libvirt
            sudo pacman -Sy
            sudo pacman -S linux-headers git cmake base-devel elfutils --noconfirm
            if [ "$is_bpf" = true ]
            then
                sudo pacman -S llvm clang --noconfirm
            fi
            ;;
        alpine) # OK generic/alpine314
            sudo apk update
            sudo apk add linux-virt-dev linux-headers g++ gcc cmake make git autoconf automake m4 libtool elfutils-dev libelf-static patch binutils
            need_musl=true
            if [ "$is_bpf" = true ]
            then
                sudo apk add llvm clang
            fi
            ;;
        opensuse-*) # ??
            sudo zypper refresh
            sudo zypper -n install kernel-default-devel gcc gcc-c++ git-core cmake patch which automake autoconf libtool libelf-devel
            if [ "$is_bpf" = true ]
            then
                sudo zypper -n install llvm clang
            fi
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

    if [ "$need_musl" = true ]
    then
        cmake -DBUILD_BPF=ON -DUSE_BUNDLED_DEPS=on -DINSTALL_GTEST=off -DBUILD_GMOCK=off -DCREATE_TEST_TARGETS=off -DMUSL_OPTIMIZED_BUILD=On ../
    else
        cmake -DBUILD_BPF=ON -DUSE_BUNDLED_DEPS=on -DINSTALL_GTEST=off -DBUILD_GMOCK=off -DCREATE_TEST_TARGETS=off ../
    fi

    if [ "$is_bpf" = true ]
    then
        make scap-open
	      echo "SCAP_BUILT: true"

        make bpf
	      echo "PROBE_BUILT: true"

	      # Do not leave for verifier issues or timeout exit code (using "&& :")
        sudo BPF_PROBE=driver/bpf/probe.o timeout 5s ./libscap/examples/01-open/scap-open && :

        res=$?

        if [ "$res" -eq "124" ] || [ "$res" -eq "143" ]
        then
            # Timed out means no verifier issues.
            # See https://man7.org/linux/man-pages/man1/timeout.1.html
            # Some weird timeout version did not exit with 124 on timeout,
            # but with 143 (ie: 128 + SIGTERM). Therefore, account for both.
            res=0
        fi
        echo "VERIFIER_TEST: $res"
    else
	      make driver
	      echo "DRIVER_BUILT: true"
	  fi
}

set -e
need_musl=false
is_bpf=%v

install_deps

echo "GCC_VERSION: $(gcc --version | head -n1 | awk -F' ' '{ print $3 }')"
echo "CLANG_VERSION: $(clang --version | head -n1 | awk -F' ' '{ print $3 }')"
echo "LINUX_VERSION: $(uname -r)"

build_and_run