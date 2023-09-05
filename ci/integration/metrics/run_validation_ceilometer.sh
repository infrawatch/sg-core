#!/bin/env bash
# CI script for UBI8 job
# purpose: verify the expected metric data is scraped by Prometheus

set -ex

dnf install -y jq hostname

PROMETHEUS_URL=http://127.0.0.1:9090
METRICS=$(curl -s "$PROMETHEUS_URL/api/v1/label/__name__/values"  | jq -r .data)

######################### gather ceilometer data #########################
ceilo_found=""
for item in $METRICS; do
  if [[ $item == \"ceilometer_* ]]; then
    if [[ -z "$ceilo_found" ]]; then
      ceilo_found=$item
    else
      ceilo_found="$ceilo_found, $item"
    fi
  fi
done

############################### validate ###############################
echo "Ceilometer metrics stored: $ceilo_found"

if [[ -z "$ceilo_found" ]] ; then
  echo "Missing expected metrics data"
  exit 1
fi
