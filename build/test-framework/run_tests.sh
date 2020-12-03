#!/bin/bash
set -ex

# bootstrap
mkdir -p /go/bin /go/src /go/pkg
export GOPATH=/go
export PATH=$PATH:$GOPATH/bin

# get dependencies
sed -i '/^tsflags=.*/a ip_resolve=4' /etc/yum.conf
dnf install -y git golang iproute
go get -u golang.org/x/tools/cmd/cover
GO111MODULES=off go get -u github.com/mattn/goveralls
go get -u golang.org/x/lint/golint
go get -u honnef.co/go/tools/cmd/staticcheck

# run code validation tools
echo " *** Running pre-commit code validation"
./build/test-framework/pre-commit

# run unit tests
echo " *** Running test suite"
go test -v ./...

set +e
echo " *** Running code coverage tooling"
go test ./... -race -covermode=atomic -coverprofile=coverage.txt

echo " *** Running Coveralls test coverage report"
goveralls -coverprofile=coverage.txt
