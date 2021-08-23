#!/bin/env bash
# CI script for UBI8 job
# purpose: spawn rsyslog with omamqp1 plugin and simulate log records creation

set -ex

EXIT_CODE=0

# Locale setting in CentOS8 is broken
dnf install -q -y glibc-langpack-en rsyslog-omamqp1

# Generate log records for verification
touch /tmp/test.log
while true
do
  echo "[$(date +'%Y-%m-%d %H:%M')] WARNING Something bad might happen" >> /tmp/test.log
  echo "[$(date +'%Y-%m-%d %H:%M')] :ERROR: Something bad happened" >> /tmp/test.log
  echo "[$(date +'%Y-%m-%d %H:%M')] [DEBUG] Wubba lubba dub dub" >> /tmp/test.log
  sleep 10
done &

# launch rsyslog
echo "Launching rsyslog"
rsyslogd -n
