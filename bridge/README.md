# Bridge between AMQP and Smart Gateway

Connects to a QDR.  Creates and AQMP endpoint. Receives and processes collectd AMQP JSON messages.  Forwards the JSON messages to the Smart Gateway through a UDP / Unix socket.

## Build

```bash
make
```

## Usage

```bash
./bridge 127.0.0.1 5672 sg 0 127.0.0.1 30000
```
