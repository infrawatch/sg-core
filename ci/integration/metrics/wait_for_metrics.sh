#!/bin/env bash
# CI helper: wait until sg-core is up and Prometheus has scraped handler metrics.
# Usage: wait_for_metrics.sh <metric_prefix>
# Example: wait_for_metrics.sh ceilometer_

set -e

METRIC_PREFIX=${1:?metric prefix required}
PROMETHEUS_URL=${PROMETHEUS_URL:-http://127.0.0.1:9090}
SG_CORE_URL=${SG_CORE_URL:-http://127.0.0.1:3000}
TIMEOUT=${METRICS_WAIT_TIMEOUT:-720}
POLL_INTERVAL=${METRICS_POLL_INTERVAL:-5}

found_metrics() {
  curl -sf "$PROMETHEUS_URL/api/v1/label/__name__/values" \
    | jq -e --arg prefix "$METRIC_PREFIX" '.data[] | select(startswith($prefix))' >/dev/null
}

sg_core_up() {
  curl -sf "$SG_CORE_URL/metrics" >/dev/null
}

elapsed=0
echo "Waiting for sg-core on ${SG_CORE_URL} ..."
while ! sg_core_up; do
  sleep "$POLL_INTERVAL"
  elapsed=$((elapsed + POLL_INTERVAL))
  if [[ $elapsed -ge $TIMEOUT ]]; then
    echo "ERROR: timeout waiting for sg-core (${TIMEOUT}s)"
    exit 1
  fi
done
echo "INFO: sg-core reachable after ${elapsed}s"

echo "Waiting for ${METRIC_PREFIX}* metrics in Prometheus ..."
while ! found_metrics; do
  sleep "$POLL_INTERVAL"
  elapsed=$((elapsed + POLL_INTERVAL))
  if [[ $elapsed -ge $TIMEOUT ]]; then
    echo "ERROR: timeout waiting for ${METRIC_PREFIX}* metrics in Prometheus (${TIMEOUT}s)"
    exit 1
  fi
done

echo "INFO: ${METRIC_PREFIX}* metrics present in Prometheus after ${elapsed}s"
curl -s "$PROMETHEUS_URL/api/v1/label/__name__/values" \
  | jq -r --arg prefix "$METRIC_PREFIX" '.data[] | select(startswith($prefix))'
