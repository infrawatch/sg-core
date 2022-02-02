#!/bin/env bash
# CI script for UBI8 job
# purpose: spawn sg-bridge for message bus connection

set -ex

# enable required repo(s)
curl -o /etc/yum.repos.d/CentOS-OpsTools.repo $OPSTOOLS_REPO

dnf install -y git gcc make qpid-proton-c-devel

# install and start sg-bridge
BRANCH="$(echo ${GITHUB_REF#refs/heads/})"
git clone https://github.com/infrawatch/sg-bridge.git
pushd sg-bridge
git checkout $BRANCH || true
make

touch $BRIDGE_SOCKET
./bridge --amqp_url amqp://localhost:5666/rsyslog/logs --gw_unix=$BRIDGE_SOCKET
