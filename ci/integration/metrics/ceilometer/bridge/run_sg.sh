#!/bin/env bash
# CI script for UBI9 job
# purpose: spawn sg-core to process messages sent by rsyslog

set -ex

# enable required repo(s)
curl -o /etc/yum.repos.d/CentOS-OpsTools.repo $OPSTOOLS_REPO

dnf install -y git golang gcc make qpid-proton-c-devel

export GOBIN=$GOPATH/bin
export PATH=$PATH:$GOBIN

bash ./ci/integration/metrics/bootstrap_go.sh

# install sg-core and start sg-core
mkdir -p /usr/lib64/sg-core
PLUGIN_DIR=/usr/lib64/sg-core/ GOCMD=go1.25.13 BUILD_ARGS=-buildvcs=false ./build.sh

./sg-core -config ./ci/integration/metrics/ceilometer/bridge/sg_config.yaml
