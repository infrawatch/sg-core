# --- build smart gateway ---
FROM registry.access.redhat.com/ubi8 AS builder
ENV GOPATH=/go
ENV D=/go/src/github.com/infrawatch/sg-core

WORKDIR $D
COPY . $D/

RUN dnf install golang git -y --setopt=tsflags=nodocs
RUN go get -d ./...
RUN go build -o /tmp/smart_gateway cmd/*.go && \
      for i in plugins/transport/*; do go build -o "/tmp/$(basename $i).so" -buildmode=plugin "./$i/..."; done && \
      for i in plugins/handler/*; do go build -o "/tmp/$(basename $i).so" -buildmode=plugin "./$i/main.go"; done && \
      for i in plugins/application/*; do go build -o "/tmp/$(basename $i).so" -buildmode=plugin "./$i/..."; done

# --- end build, create smart gateway layer ---
FROM registry.access.redhat.com/ubi8

LABEL io.k8s.display-name="Smart Gateway" \
      io.k8s.description="A component of the Service Telemetry Framework on the server side that ingests data from AMQP 1.x and provides a metrics scrape endpoint for Prometheus, and forwards events to ElasticSearch" \
      maintainer="Leif Madsen <leif+smartgatewayoperator@redhat.com>"

COPY --from=builder /tmp/smart_gateway /
COPY --from=builder /tmp/*.so /usr/lib64/sg-core/

ENTRYPOINT ["/smart_gateway"]
