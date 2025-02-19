#!/bin/env bash
# CI script for UBI9 job
# purpose: verify the expected metric data is scraped by Prometheus

set -ex

dnf install -y jq hostname

PROMETHEUS_URL=http://127.0.0.1:9090
METRICS=$(curl -s "$PROMETHEUS_URL/api/v1/label/__name__/values"  | jq -r .data)

######################### gather collectd data #########################
collectd_found=""
for item in $METRICS; do
  if [[ $item == \"collectd_* ]]; then
    if [[ -z "$collectd_found" ]]; then
      collectd_found=$item
    else
      collectd_found="$collectd_found, $item"
    fi
  fi
done

############################### validate ###############################
echo "Collectd metrics stored: $collectd_found"

if [[ -z "$collectd_found" ]]; then
  echo "Missing expected metrics data"
  exit 1
fi
