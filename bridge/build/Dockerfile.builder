# --- build smart gateway ---
FROM centos:8 AS builder

RUN yum install epel-release -y && \
        yum update -y --setopt=tsflags=nodocs && \
        yum install qpid-proton-c-devel --setopt=tsflags=nodocs -y && \
        dnf install gcc make -y && \
        yum clean all

