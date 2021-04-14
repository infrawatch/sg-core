#!/bin/env bash
# CI script for CentOS8 job
# purpose: spawn sg-core to process messages sent by rsyslog

set -ex

# enable required repo(s)
yum install -y epel-release

# Without glibc-langpack-en locale setting in CentOS8 is broken without this package
yum install -y git golang gcc make glibc-langpack-en qpid-proton-c-devel

export GOBIN=$GOPATH/bin
export PATH=$PATH:$GOBIN

SOCKET=/tmp/rsyslog-test-socket
BRIDGE_LOG=/var/log/sg-bridge.log

# install and start sg-bridge
git clone https://github.com/infrawatch/sg-bridge.git
pushd sg-bridge
make

touch $SOCKET
touch $BRIDGE_LOG
./bridge --amqp_url amqp://localhost:5666/rsyslog/logs --gw_unix=$SOCKET &>$BRIDGE_LOG &
sleep 1
cat $BRIDGE_LOG
popd

# install sg-core and start sg-core
mkdir -p /usr/lib64/sg-core
PLUGIN_DIR=/usr/lib64/sg-core/ ./build.sh

find / -iname "*sg-core"
find / -iname "*socket.so"

./sg-core -config ./ci/integration/logging/sg_config.yaml
