#!/bin/env bash
# CI script for UBI8 job
# purpose: spawn sg-bridge for message bus connection

set -ex

CHANNEL=$QDR_CHANNEL_CEILOMTR
CHANNEL=${CHANNEL:-$QDR_CHANNEL_COLLECTD}

# enable required repo(s)
cat > /etc/yum.repos.d/fedora-eln.repo <<EOF
[centos-opstools]
name=opstools
baseurl=http://mirror.centos.org/centos/8/opstools/\$basearch/collectd-5/
gpgcheck=0
enabled=1
module_hotfixes=1
EOF

dnf install -y git gcc make qpid-proton-c-devel

# install and start sg-bridge
BRANCH="$(echo ${GITHUB_REF#refs/heads/})"
git clone https://github.com/infrawatch/sg-bridge.git
pushd sg-bridge
git checkout $BRANCH || true
make

touch $BRIDGE_SOCKET
./bridge --amqp_url amqp://localhost:5666/$CHANNEL --gw_unix=$BRIDGE_SOCKET
