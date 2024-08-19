#!/bin/env bash
# CI script for CentOS9 job
# purpose: run unit test suite and submit code coverage

set -ex

# enable required repo(s)
curl -o /etc/yum.repos.d/CentOS-OpsTools.repo $OPSTOOLS_REPO

# without glibc-langpack-en locale setting in CentOS8 is broken without this package
yum install -y git golang gcc make glibc-langpack-en qpid-proton-c-devel

export GOBIN=$GOPATH/bin
export PATH=$PATH:$GOBIN

go install golang.org/dl/go1.22.0@latest
go1.22.0 download

go1.22.0 test -v -coverprofile=profile.cov ./...
