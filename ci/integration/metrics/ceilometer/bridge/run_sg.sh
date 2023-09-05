#!/bin/env bash
# CI script for UBI8 job
# purpose: spawn sg-core to process messages sent by rsyslog

set -ex

# enable required repo(s)
curl -o /etc/yum.repos.d/CentOS-OpsTools.repo $OPSTOOLS_REPO
sed -i 's/gpgcheck=1/gpgcheck=0/g' /etc/yum.repos.d/CentOS-OpsTools.repo

dnf install -y git golang gcc make qpid-proton-c-devel

export GOBIN=$GOPATH/bin
export PATH=$PATH:$GOBIN

go install golang.org/dl/go1.19@latest
go1.19 download

# install sg-core and start sg-core
mkdir -p /usr/lib64/sg-core
PLUGIN_DIR=/usr/lib64/sg-core/ GOCMD=go1.19 BUILD_ARGS=-buildvcs=false ./build.sh

./sg-core -config ./ci/integration/metrics/sg_config_bridge.yaml
