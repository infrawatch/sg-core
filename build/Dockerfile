# --- build smart gateway ---
FROM registry.access.redhat.com/ubi8:latest AS builder
ENV GOPATH=/go
ENV D=/go/src/github.com/infrawatch/sg-core

WORKDIR $D
COPY . $D/

# dependencies for qpid-proton-c
COPY build/repos/opstools.repo /etc/yum.repos.d/opstools.repo

RUN dnf install golang git qpid-proton-c-devel -y --setopt=tsflags=nodocs
RUN go install golang.org/dl/go1.21.13@latest && /go/bin/go1.21.13 download && PRODUCTION_BUILD=false CONTAINER_BUILD=true GOCMD=/go/bin/go1.21.13 ./build.sh

# --- end build, create smart gateway layer ---
FROM registry.access.redhat.com/ubi8-minimal:latest

LABEL io.k8s.display-name="Smart Gateway" \
      io.k8s.description="A component of the Service Telemetry Framework on the server side that ingests data from AMQP 1.x and provides a metrics scrape endpoint for Prometheus, and forwards events to ElasticSearch" \
      maintainer="Leif Madsen <leif+smartgatewayoperator@redhat.com>"

COPY --from=builder /tmp/sg-core /
COPY --from=builder /tmp/plugins/*.so /usr/lib64/sg-core/

ENTRYPOINT ["/sg-core"]
