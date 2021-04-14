#!/bin/env bash
# CI script for CentOS8 job
# purpose: spawn rsyslog with omamqp1 plugin and simulate log records creation

set -ex

EXIT_CODE=0

# enable ELN repo
cat > /etc/yum.repos.d/fedora-eln.repo <<EOF
[eln-baseos]
name=Fedora - ELN BaseOS - Developmental packages for the next Enterprise Linux release
baseurl=https://odcs.fedoraproject.org/composes/production/latest-Fedora-ELN/compose/BaseOS/\$basearch/os/
#metalink=https://mirrors.fedoraproject.org/metalink?repo=eln&arch=\$basearch
enabled=1
gpgcheck=0
skip_if_unavailable=False
EOF

# Locale setting in CentOS8 is broken
yum install -y glibc-langpack-en rsyslog-omamqp1

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
cat /etc/rsyslog.d/*
rsyslogd -n
