# Multiple socket plugins
To run sg-core with multiple socket (or other - untested) plugins, create a configuration similar to the following:
```
transports:
    - name: socket
      config:
          path: "/tmp/smartgateway"
      handlers:
      - name: logs
        config:
            timestampField: "@timestamp"
            messageField: "message"
            severityField: "severity"
            hostnameField: "host"
    - name: socket
      config:
          path: "/tmp/logs"
      handlers:
      - name: logs
        config:
            timestampField: "@timestamp"
            messageField: "message"
            severityField: "severity"
            hostnameField: "host"
```
This will start 2 socket plugins in parallel, each listening on a different socket. Possible use case for this setup is
when running multiple sg-bridges, each listening on a different amqp address like this:
```
./bridge --amqp_url=amqp://127.0.0.1:5672/collectd --gw_unix=/tmp/smartgateway
```
```
./bridge --amqp_url=amqp://127.0.0.1:5672/logs --gw_unix=/tmp/logs
```
