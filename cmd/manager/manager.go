package manager

import (
	"context"
	"fmt"
	"path/filepath"
	"plugin"
	"strings"
	"sync"

	"github.com/infrawatch/apputils/logging"
	"github.com/infrawatch/sg-core/pkg/application"
	"github.com/infrawatch/sg-core/pkg/bus"
	"github.com/infrawatch/sg-core/pkg/handler"
	"github.com/infrawatch/sg-core/pkg/transport"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

//errors
var (
	//ErrAppNotReceiver return if application plugin does not implement any receiver. In this case, it will receive no messages from the internal buses
	ErrAppNotReceiver = errors.New("application plugin does not implement either application.MetricReceiver or application.EventReceiver")
)
var (
	transports   map[string]transport.Transport
	handlers     map[string][]handler.Handler
	applications map[string]application.Application
	eventBus     bus.EventBus
	metricBus    bus.MetricBus
	pluginPath   string
	logger       *logging.Logger
)

func init() {
	transports = map[string]transport.Transport{}
	handlers = map[string][]handler.Handler{}
	applications = map[string]application.Application{}
	pluginPath = "/usr/lib64/sg-core"
}

//SetPluginDir set directory path containing plugin binaries
func SetPluginDir(path string) {
	pluginPath = path
}

//SetLogger set logger
func SetLogger(l *logging.Logger) {
	logger = l
}

//InitTransport load tranpsort binary and initialize with config
func InitTransport(name string, config interface{}) error {
	n, err := initPlugin(name)
	if err != nil {
		return errors.Wrap(err, "failed initializing transport")
	}

	new, ok := n.(func(*logging.Logger) transport.Transport)
	if !ok {
		return fmt.Errorf("plugin %s constructor 'New' did not return type 'transport.Transport'", name)
	}

	transports[name] = new(logger)

	c, err := yaml.Marshal(config)
	if err != nil {
		return errors.Wrapf(err, "failed parsing transport config for '%s'", name)
	}

	err = transports[name].Config(c)
	if err != nil {
		return err
	}
	return nil
}

//InitApplication initialize application plugin with configuration
func InitApplication(name string, config interface{}) error {
	n, err := initPlugin(name)
	if err != nil {
		return errors.Wrap(err, "failed initializing application plugin")
	}

	new, ok := n.(func(*logging.Logger) application.Application)
	if !ok {
		return fmt.Errorf("plugin %s constructor 'New' did not return type 'application.Application'", name)
	}

	applications[name] = new(logger)

	c, err := yaml.Marshal(config)
	if err != nil {
		return errors.Wrapf(err, "failed parsing application plugin config for '%s'", name)
	}

	err = applications[name].Config(c)
	if err != nil {
		return err
	}

	// does it implement MetricReceiver?
	// does it implement EventReceiver?
	var mReceiver bool
	var eReceiver bool
	var itf interface{} = applications[name]
	if r, ok := itf.(application.MetricReceiver); ok {
		mReceiver = true
		metricBus.Subscribe(r.ReceiveMetric)
	}

	if r, ok := itf.(application.EventReceiver); ok {
		eReceiver = true
		eventBus.Subscribe(r.ReceiveEvent)
	}

	if !(mReceiver || eReceiver) {
		return ErrAppNotReceiver
	}

	return nil
}

//SetTransportHandlers load handlers binaries for transport
func SetTransportHandlers(name string, handlerNames []string) error {
	for _, hName := range handlerNames {
		n, err := initPlugin(hName)
		if err != nil {
			return errors.Wrap(err, "failed initializing handler")
		}

		new, ok := n.(func() handler.Handler)
		if !ok {
			return fmt.Errorf("handler %s constructor did not return type handler.Handler", hName)
		}
		handlers[name] = append(handlers[name], new())

		logger.Metadata(logging.Metadata{"transport pair": name, "handler": hName})
		logger.Info("initialized handler")
	}
	return nil
}

//RunTransports spins off tranpsort + handler processes
func RunTransports(ctx context.Context, wg *sync.WaitGroup, done chan bool, report bool) {
	for name, t := range transports {
		for _, h := range handlers[name] {
			wg.Add(1)
			go func(wg *sync.WaitGroup, h handler.Handler) {
				defer wg.Done()
				h.Run(ctx, metricBus.Publish, eventBus.Publish)
			}(wg, h)
		}

		wg.Add(1)
		go func(wg *sync.WaitGroup, t transport.Transport, name string) {
			defer wg.Done()
			t.Run(ctx, func(blob []byte) {
				for _, h := range handlers[name] {
					err := h.Handle(blob, report, metricBus.Publish, eventBus.Publish)
					if err != nil {
						logger.Metadata(logging.Metadata{"error": err, "handler": fmt.Sprintf("%s[%s]", h.Identify(), name)})
						logger.Debug("failed handling message")
					}
				}
			}, done)
		}(wg, t, name)
	}
}

//RunApplications spins off application processes
func RunApplications(ctx context.Context, wg *sync.WaitGroup, done chan bool) {
	for _, a := range applications {
		wg.Add(1)
		go func(wg *sync.WaitGroup, a application.Application) {
			defer wg.Done()
			a.Run(ctx, done)
		}(wg, a)
	}
}

// helper functions

func initPlugin(name string) (plugin.Symbol, error) {
	bin := strings.Join([]string{name, "so"}, ".")
	path := filepath.Join(pluginPath, bin)
	p, err := plugin.Open(path)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open binary %s", path)
	}

	n, err := p.Lookup("New")
	return n, err
}
