#!/bin/env bash
# CI script for UBI8 job
# purpose: spawn sg-core to process messages sent by rsyslog

set -ex

# enable required repo(s)
cat > /etc/yum.repos.d/fedora-eln.repo <<EOF
[centos-opstools]
name=opstools
baseurl=http://mirror.centos.org/centos/8/opstools/\$basearch/collectd-5/
gpgcheck=0
enabled=1
module_hotfixes=1
EOF

dnf install -y git golang gcc make qpid-proton-c-devel

export GOBIN=$GOPATH/bin
export PATH=$PATH:$GOBIN

# install sg-core and start sg-core
go mod tidy
mkdir -p /usr/lib64/sg-core
PLUGIN_DIR=/usr/lib64/sg-core/ ./build.sh

./sg-core -config ./ci/integration/logging/sg_config.yaml
