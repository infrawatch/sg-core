#!/bin/bash

vol="prom-data"

if [ "$#" -eq 1 ]; then
    vol="$1"
    shift
fi

podman volume create "$vol" 
