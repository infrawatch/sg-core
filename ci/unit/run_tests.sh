#!/bin/env bash
# CI script for CentOS8 job
# purpose: runt unit test suite and submit code coverage

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

# without glibc-langpack-en locale setting in CentOS8 is broken without this package
yum install -y git golang gcc make glibc-langpack-en qpid-proton-c-devel

export GOBIN=$GOPATH/bin
export PATH=$PATH:$GOBIN

go mod tidy
go test -v -coverprofile=profile.cov ./...
