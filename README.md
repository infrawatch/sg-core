# Plugins
Default plugins exist in /plugins. Plugins can also be hosted as separate projects.

## Types
### Transport
Transports listen on an external protocol for incoming messages.

### Handler
Handlers receive message blobs from a transport plugin and parse them 
to be placed on the internal metrics or events bus. A handler can either be a 
metric or event handler, but not both. 


### Application
Applications receive both metrics and events and decide what to do
with them. Most application plugins interact with a storage backend such as 
Prometheus.

# Build
```bash
# build sg-core and plugins. Places plugin binaries in ./bin
./build.sh
```

# Configuration
Administrators must specify 3 sections in the yaml config:

1. Sg-core configuration options
2. Transport plugins and configurations
3. Application plugins and configurations

``` yaml
#1 sg-core configs
plugindir:
loglevel:

#2 Transports
transports:
  - name: <transport-0>
    handlers:
      - <handler-0>
      [...]
      - <handler-n>
    config:
      # plugin specific configuration
  [...]

  - name: <transport-n>
    handlers:
      [...]
    config:

#3 Applications
applications:
  - name: <application-0>
    config:
      # application plugin specific configurations
  [...]

  - name: <application-n>
    config:
```

Section one describes sg-core specific configurations. 

Section two describes any number of transport plugins that should be configured 
in a list. Each transport plugin can bind any number of message handlers to itself. 
Keep in mind that at this time, all handlers receive every message arriving on 
the transport. Thus, it is generally only viable to bind either all metric handlers
or all event handlers to any one transport. The `config` block is specific to 
the plugin being described. For example, TCP transport may require an address to
be specified here.

Section three describes application plugins. Just like the transport, more than
one application can be configured to run. Each application block contains a 
config block specific to that plugin.

## Example Configuration
This configuration assumes both a QPID Dispatch Router and Prometheus instance
are running on the localhost and listens for incoming messages on a unix socket
at `/tmp/smartgateway`. The setup expects incoming messages to be collectd
metrics, as can be seen by the type of handler bound to the socket transport.

```yaml
plugindir: bin/
loglevel: debug
transports:
  - name: socket
    handlers: 
      - collectd-metrics
    config:
      address: /tmp/smartgateway
applications: 
  - name: prometheus 
    config:
      host: localhost
      port: 8081
      withtimestamp: false
```

## Run
`./sg-core -config <path to config>`

## Docker/Podman
Build:
`podman build -t sg-core -f build/Dockerfile .`

Run:
`podman run -d -v /path/to/sg-core.conf.yaml:/etc/sg-core.conf.yaml:z --name sg-core sg-core`
