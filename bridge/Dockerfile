# --- build smart gateway ---
FROM centos:7 AS builder

ENV D=/home/bridge

WORKDIR $D
COPY . $D/

RUN yum install epel-release -y && \
        yum -y install centos-release-scl && \
        yum update -y --setopt=tsflags=nodocs && \
        yum install qpid-proton-c-devel --setopt=tsflags=nodocs -y && \
        yum install devtoolset-8-gcc* devtoolset-8-make -y && \
        yum clean all

SHELL [ "/usr/bin/scl", "enable", "devtoolset-8", "bash"] 
RUN     gcc -v
RUN     make && \
        mv bridge /tmp/

# --- end build, create smart gateway layer ---
FROM centos:7

LABEL io.k8s.display-name="Service Assurance Smart Bridge" \
      io.k8s.description="A component of the Service Assurance Framework on the server side that ingests data from AMQP 1.x and provides a metrics scrape endpoint for Prometheus, and forwards events to ElasticSearch" \
      maintainer="Leif Madsen <leif@redhat.com>"

RUN yum install epel-release -y && \
        yum update -y --setopt=tsflags=nodocs && \
        yum install qpid-proton-c --setopt=tsflags=nodocs -y && \
        yum clean all && \
        rm -rf /var/cache/yum

COPY --from=builder /tmp/bridge /

ENTRYPOINT ["/bridge"]

