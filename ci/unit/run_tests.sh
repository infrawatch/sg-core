#!/bin/env bash
# CI script for CentOS8 job
# purpose: runt unit test suite and submit code coverage

set -ex

# enable required repo(s)
curl -o /etc/yum.repos.d/CentOS-OpsTools.repo $OPSTOOLS_REPO
sed -i 's/gpgcheck=1/gpgcheck=0/g' /etc/yum.repos.d/CentOS-OpsTools.repo

# without glibc-langpack-en locale setting in CentOS8 is broken without this package
yum install -y git golang gcc make glibc-langpack-en qpid-proton-c-devel

export GOBIN=$GOPATH/bin
export PATH=$PATH:$GOBIN

go install golang.org/dl/go1.21.13@latest
go1.21.13 download


go1.21.13 test -v -coverprofile=profile.cov ./...
