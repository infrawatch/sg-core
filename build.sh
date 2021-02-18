#!/bin/bash

base=$(pwd)

PLUGIN_DIR=${PLUGIN_DIR:-"/tmp/plugins/"}
CONTAINER_BUILD=${CONTAINER_BUILD:-false}

for i in plugins/transport/*; do 
  cd "$base/$i"
  echo "building $(basename $i).so"
  go build -o "$PLUGIN_DIR$(basename $i).so" -buildmode=plugin
done
cd "$base"
for i in plugins/handler/*; do 
  cd "$base/$i"
  echo "building $(basename $i).so"
  go build -o "$PLUGIN_DIR$(basename $i).so" -buildmode=plugin
done
cd "$base"
for i in plugins/application/*; do
  cd "$base/$i"
  echo "building $(basename $i).so"
  go build -o "$PLUGIN_DIR$(basename $i).so" -buildmode=plugin
done
cd "$base"

if $CONTAINER_BUILD; then
    echo "building sg-core for container"
    go build -o /tmp/sg-core cmd/*.go
else 
    go build -o sg-core cmd/*.go
fi
