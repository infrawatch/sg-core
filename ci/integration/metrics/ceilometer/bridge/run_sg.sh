#!/bin/env bash
# CI script for UBI9 job
# purpose: spawn sg-core to process messages sent by rsyslog

set -ex

# enable required repo(s)
curl -o /etc/yum.repos.d/centos9-caracal.repo $OPENSTACK_REPO

dnf install -y git golang gcc make qpid-proton-c-devel

export GOBIN=$GOPATH/bin
export PATH=$PATH:$GOBIN

go install golang.org/dl/go1.21.13@latest
go1.21.13 download

# install sg-core and start sg-core
mkdir -p /usr/lib64/sg-core
PLUGIN_DIR=/usr/lib64/sg-core/ GOCMD=go1.21.13 BUILD_ARGS=-buildvcs=false ./build.sh

./sg-core -config ./ci/integration/metrics/ceilometer/bridge/sg_config.yaml
