#!/bin/env bash
# CI helper: install and download the Go toolchain used to build sg-core plugins.

set -ex

export GOBIN=${GOBIN:-$GOPATH/bin}
export PATH=$PATH:$GOBIN

MAX_ATTEMPTS=5
attempt=1
while [[ $attempt -le $MAX_ATTEMPTS ]]; do
  if go install golang.org/dl/go1.25.13@latest && go1.25.13 download; then
    exit 0
  fi
  echo "Go 1.25.13 bootstrap attempt ${attempt}/${MAX_ATTEMPTS} failed"
  attempt=$((attempt + 1))
  sleep 10
done

echo "ERROR: failed to bootstrap Go 1.25.13 after ${MAX_ATTEMPTS} attempts"
exit 1
