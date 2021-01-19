# Plugins

1. Transport
2. Handler
3. Application

## Philosophy
Plugins should be objects with a constructor such that more than one can be 
created if there exists more than one configuration for that plugin.

# Development
Sg-core begins by loading plugin shared object files (those that have been configured) and calling the New() function. The New() function is different for each type:

Plugin Type | Initializer Function
-|-
Transport | `func New(* logging.Logger) transport.Transport`
Handler | `func New() handler.MetricHandler` or `func New() handler.EventHandler`
Application | `func New(* logging.Logger) application.Application`

Both transport and application plugins contain a Run() function which encompass their primary process. Because these processes are run in a separate goroutine, a golang context is provided to synchronize with the rest of sg-core.

A plugin's Run() function should listen for close signals on the context and exit when it is received. Additionally, if a critical error occurs, the plugin should pass `true` to the boolean channel. This will signal the sg-core to perform a clean exit.

```go
func (t *TCP) Run(ctx context.Context, wg *sync.WaitGroup, w transport.WriteFn, done chan bool) transport.Transport {
    defer wg.Done()

    go func() {
        // receive messages from transport protocol

        if err != nil {  // some error or exit condition 
            done <- true // signal to sg-core that an unrecoverable event occured and that a clean exit should happen
            return
        }
    }

    <-ctx.Done() //wait for exit signal
    //cleanup resources
}
```


## Configurations
Plugins should not read their own cofiguration files. The sg-core reads the plugin configuration from the `config` block and passes it into the plugin's `Config()` method as a byte slice. Most plugin's configuration validation should be done with the ParseConfig() function in the `pkg/config` package. ParseConfig() validates objects using the [validator](https://pkg.go.dev/gopkg.in/go-playground/validator.v9) library and provides descriptive error messages for failed configurations. Typically, the plugin specifies a configuration object with yaml and validator tags and passes an instance of it into the ParseConfig() method for unmarshalling and validation.

Here is the pattern typically followed by most application and transport plugins for receiving and validating a configuration:

```go
package main


[...]

type configT struct {
    Address string `yaml:"address" validate:"required"` //required parameter
    Instance string `yaml:"instance" validate:"oneof=inst0 inst1 inst2 inst3"` //must be element in set
    Port int `yaml:"port"`
}

type Socket struct {
    conf configT
    logger *logging.Logger
}

func (s *Socket) Config(c []byte) error {
    s.conf = configT{}
    err := config.ParseConfig(bytes.NewReader(c), &s.conf)
    if err != nil {
        return err // sg-core will log configuration error message and exit
    }
    return nil
}

[...]

func New(l *logging.Logger) transport.Transport {
    return &Socket{
        logger: l,
        conf: configT{
            Port: 9090 // set defaults. Only overriden if input data for Config() contains the port option
        }
    }
}

```

## Transports

Transport plugins listen on an external messaging protocal and deliver received messages to handlers that have been bound to it per the administrator's configuration. They receive a configuration block from the main configuration file and write to handlers by involing transport.WriteFn in Run().

Transports should contain the minimal amount of code necessary to fulfill this functionality. 

Transport plugin objects must implement the the Transport interface:
```go
type Transport interface {
	Config([]byte) error
	Run(context.Context, *sync.WaitGroup, transport.WriteFn, chan bool)
	Listen(data.Event)
}
```

## Handlers

Handlers parse incoming blobs from the transport into objects and delivers those objects to the internal buses. There are two types of handlers: metric handlers and event handlers. Metric handlers deliver metric objects to the internal metrics bus while event handlers deliver event objects to the internal events bus. These metrics and events are then consumed by the application plugins.

Handlers must remain simple: they take in no configuration and should only handle one message type. See the [collectd-metrics](https://github.com/pleimer/sg-core-refactor/tree/master/plugins/handler/collectd-metrics) plugin for an example. Additionally, handlers should not print logs (this is why no logger is passed into the New() function). If errors occur while parsing messages, handlers should create their own metrics or events recording the error(s) and write them to the bus. The `collectd-metrics` handler iterates a counter every time a parsing error occurs and submits the number as a metric to the metric bus.


Handler plugin objects must imeplement either the MetricHandler interface or the EventHandler interface:
```go
type MetricHandler interface {
	Handle([]byte) []data.Metric
}

type EventHandler interface {
	Handle([]byte) (data.Event, error)
}
```

## Applications

The purpose of application plugins are to provide the business logic for interfacing with external programs like a database. They receive both metrics and events and must decide what to do with them. For example, the [prometheus](https://github.com/pleimer/sg-core-refactor/tree/master/plugins/application/prometheus) plugin receives metrics from the internal metrics bus and stores them into Prometheus.

Application plugins must implement the Application interface:
```go
type Application interface {
	Config([]byte) error
	Run(context.Context, *sync.WaitGroup, chan data.Event, chan []data.Metric, chan bool)
}
```

## Examples
Examples of the implementation of each type of plugin can be found in the [plugins](https://github.com/pleimer/sg-core-refactor/tree/master/plugins) directory.
