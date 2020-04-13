# --- build smart gateway ---
FROM centos:7 AS builder
ENV GOPATH=/go
ENV D=/go/src/github.com/infrawatch/sg

WORKDIR $D
COPY . $D/

RUN yum install epel-release -y && \
        yum update -y --setopt=tsflags=nodocs && \
        yum install git golang --setopt=tsflags=nodocs -y && \
        yum clean all && \
        go build -o smart_gateway cmd/server/server.go && \
        mv smart_gateway /tmp/

# --- end build, create smart gateway layer ---
FROM centos:7

LABEL io.k8s.display-name="Smart Gateway" \
      io.k8s.description="A component of the Service Telemetry Framework on the server side that ingests data from AMQP 1.x and provides a metrics scrape endpoint for Prometheus, and forwards events to ElasticSearch" \
      maintainer="Leif Madsen <leif@redhat.com>"

RUN yum install epel-release -y && \
        yum update -y --setopt=tsflags=nodocs && \
        yum clean all && \
        rm -rf /var/cache/yum

COPY --from=builder /tmp/smart_gateway /

ENTRYPOINT ["/smart_gateway"]