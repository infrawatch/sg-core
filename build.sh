#!/bin/bash

base=$(pwd)
for i in plugins/transport/*; do 
  cd "$base/$i"
  go build -o "/tmp/plugins/$(basename $i).so" -buildmode=plugin
done
cd "$base"
for i in plugins/handler/*; do 
  cd "$base/$i"
  go build -o "/tmp/plugins/$(basename $i).so" -buildmode=plugin
done
cd "$base"
for i in plugins/application/*; do
  cd "$base/$i"
  go build -o "/tmp/plugins/$(basename $i).so" -buildmode=plugin
done
cd "$base"
go build -o sg-core cmd/*.go
