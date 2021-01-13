#!/bin/bash

for i in plugins/transport/*; do go build -o bin/ -buildmode=plugin "./$i/..."; done
for i in plugins/handler/*; do go build -o "bin/$(basename $i).so" -buildmode=plugin "./$i/main.go"; done
for i in plugins/application/*; do go build -o bin/ -buildmode=plugin "./$i/..."; done

go build -o sg-core cmd/*.go
