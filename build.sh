#!/bin/bash

# Build locally:
# ./build.sh
#
# Build for container:
# CONTAINER_BUILD=true ./build.sh
#
# Production build (omits test plugin binaries to minimize image size and builds for container)
# PRODUCTION_BUILD=true ./build.sh

base=$(pwd)

GOCMD=${GOCMD:-"go"}
PLUGIN_DIR=${PLUGIN_DIR:-"/tmp/plugins/"}
CONTAINER_BUILD=${CONTAINER_BUILD:-false}

PRODUCTION_BUILD=${PRODUCTION_BUILD:-false}
if $PRODUCTION_BUILD; then
    CONTAINER_BUILD=true
fi

# add plugins here that should be omitted in production build
# to keep the image size as small as possible
if $PRODUCTION_BUILD; then
  OMIT_TRANSPORTS=(
      "dummy-alertmanager"
      "dummy-events"
      "dummy-metrics"
      "dummy-logs"
  )

  OMIT_HANDLERS=(

  )

  OMIT_APPLICATIONS=(
  )
fi



search_list() {
  #arges: search_string, list_to_search
  arrName=$2[@]
  arr=("${!arrName}")
  for entry in "${arr[@]}"; do
    if [ $entry == $1 ]; then
      return 1
    fi
  done
  return 0
}

build_plugins() {
  # build transports
  cd "$base"
  for i in plugins/transport/*; do
    cd "$base/$i"
    search_list "$(basename $i)" OMIT_TRANSPORTS
    if [ $? -ne 1 ]; then
      echo "building $(basename $i).so"
      $GOCMD build -o "$PLUGIN_DIR$(basename $i).so" -buildmode=plugin
    fi
  done

  # build handlers
  cd "$base"
  for i in plugins/handler/*; do
    cd "$base/$i"
    search_list "$(basename $i)" OMIT_HANDLERS
    if [ $? -ne 1 ]; then
      echo "building $(basename $i).so"
      $GOCMD build -o "$PLUGIN_DIR$(basename $i).so" -buildmode=plugin
    fi
  done

  # build applications
  cd "$base"
  for i in plugins/application/*; do
    cd "$base/$i"
    search_list "$(basename $i)" OMIT_APPLICATIONS
    if [ $? -ne 1 ]; then
      echo "building $(basename $i).so"
      $GOCMD build -o "$PLUGIN_DIR$(basename $i).so" -buildmode=plugin
    fi
  done
}

build_core() {
  # build sg-core
  cd "$base"
  if $CONTAINER_BUILD; then
      echo "building sg-core for container"
      $GOCMD build -o /tmp/sg-core cmd/*.go
  else
      $GOCMD build -o sg-core cmd/*.go
  fi
}

build_plugins
build_core
