#!/bin/env bash
# CI script for UBI9 job
# purpose: verify the expected metric data is scraped by Prometheus

set -ex

METRICS_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)

dnf install -y jq hostname

bash "$METRICS_ROOT/wait_for_metrics.sh" collectd_
