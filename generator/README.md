# Generate collectd JSON AMQP messages

Connects to a QDR.  Generates collectd JSON formated metrics.  Sends the metrics to the bridge.

## Build

```bash
make
```

## Usage

```bash
./gen -v -n 4 127.0.0.1 5672
```
