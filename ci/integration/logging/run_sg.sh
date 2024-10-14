#!/bin/env bash
# CI script for UBI8 job
# purpose: spawn sg-core to process messages sent by rsyslog

set -ex

# enable required repo(s)
curl -o /etc/yum.repos.d/CentOS-OpsTools.repo $OPSTOOLS_REPO
sed -i 's/gpgcheck=1/gpgcheck=0/g' /etc/yum.repos.d/CentOS-OpsTools.repo
# Update to use the vault mirror since Centos 8s is EOL
sed -i 's/^#baseurl.*$/baseurl=http:\/\/vault.centos.org\/$releasever-stream\/opstools\/$basearch\/collectd-5/g' /etc/yum.repos.d/CentOS-OpsTools.repo
sed -i 's/^mirror/#mirror/g' /etc/yum.repos.d/CentOS-OpsTools.repo

dnf install -y git golang gcc make qpid-proton-c-devel

export GOBIN=$GOPATH/bin
export PATH=$PATH:$GOBIN

go install golang.org/dl/go1.21.13@latest
go1.21.13 download

# install sg-core and start sg-core
mkdir -p /usr/lib64/sg-core
PLUGIN_DIR=/usr/lib64/sg-core/ GOCMD=go1.21.13 BUILD_ARGS=-buildvcs=false ./build.sh

./sg-core -config ./ci/integration/logging/sg_config.yaml
