package manager

import (
	"context"
	"fmt"
	"path/filepath"
	"plugin"
	"strconv"
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

// errors
var (
	// ErrAppNotReceiver return if application plugin does not implement any receiver. In this case, it will receive no messages from the internal buses
	ErrAppNotReceiver = errors.New("application plugin does not implement either application.MetricReceiver or application.EventReceiver")
)
var (
	transports        map[string]transport.Transport
	handlers          map[string][]handler.Handler
	applications      map[string]application.Application
	eventBus          bus.EventBus
	metricBus         bus.MetricBus
	pluginPath        string
	logger            *logging.Logger
	eventPublishFunc  bus.EventPublishFunc
	metricPublishFunc bus.MetricPublishFunc
)

func init() {
	transports = map[string]transport.Transport{}
	handlers = map[string][]handler.Handler{}
	applications = map[string]application.Application{}
	pluginPath = "/usr/lib64/sg-core"
	eventPublishFunc = eventBus.Publish
	metricPublishFunc = metricBus.Publish
}

// SetPluginDir set directory path containing plugin binaries
func SetPluginDir(path string) {
	pluginPath = path
}

// SetLogger set logger
func SetLogger(l *logging.Logger) {
	logger = l
}

// SetBlockingEventBus set the correct event bus publish function
func SetEventBusBlocking(block bool) {
	if block {
		eventPublishFunc = eventBus.PublishBlocking
	} else {
		eventPublishFunc = eventBus.Publish
	}
}

// InitTransport load tranpsort binary and initialize with config
func InitTransport(name string, config interface{}) (string, error) {
	n, err := initPlugin(name)
	if err != nil {
		return "", errors.Wrap(err, "failed initializing transport")
	}

	new, ok := n.(func(*logging.Logger) transport.Transport)
	if !ok {
		return "", fmt.Errorf("plugin %s constructor 'New' did not return type 'transport.Transport'", name)
	}

	// Append the current length of transports
	// to make each name unique
	uniqueName := name + strconv.Itoa(len(transports))
	transports[uniqueName] = new(logger)

	c, err := yaml.Marshal(config)
	if err != nil {
		return "", errors.Wrapf(err, "failed parsing transport config for '%s'", name)
	}

	err = transports[uniqueName].Config(c)
	if err != nil {
		return "", err
	}
	return uniqueName, nil
}

// InitApplication initialize application plugin with configuration
func InitApplication(name string, config interface{}) error {
	n, err := initPlugin(name)
	if err != nil {
		return errors.Wrap(err, "failed initializing application plugin")
	}

	new, ok := n.(func(*logging.Logger, bus.EventPublishFunc) application.Application)
	if !ok {
		return fmt.Errorf("plugin %s constructor 'New' did not return type 'application.Application'", name)
	}

	app := new(logger, eventBus.Publish)

	c, err := yaml.Marshal(config)
	if err != nil {
		return errors.Wrapf(err, "failed parsing application plugin config for '%s'", name)
	}

	err = app.Config(c)
	if err != nil {
		return err
	}

	// does it implement MetricReceiver?
	// does it implement EventReceiver?
	var mReceiver bool
	var eReceiver bool
	var itf interface{} = app
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

	applications[name] = app
	return nil
}

// SetTransportHandlers load handlers binaries for transport
func SetTransportHandlers(name string, handlerBlocks []struct {
	Name   string `validate:"required"`
	Config interface{}
}) error {
	for _, block := range handlerBlocks {
		n, err := initPlugin(block.Name)
		if err != nil {
			return errors.Wrap(err, "failed initializing handler")
		}

		new, ok := n.(func() handler.Handler)
		if !ok {
			return fmt.Errorf("handler %s constructor did not return type handler.Handler", block.Name)
		}
		h := new()

		configBlob, err := yaml.Marshal(block.Config)
		if err != nil {
			return errors.Wrapf(err, "failed parsing handler plugin config for '%s'", block.Name)
		}

		err = h.Config(configBlob)
		if err != nil {
			return err
		}

		handlers[name] = append(handlers[name], h)

		logger.Metadata(logging.Metadata{"transport pair": name, "handler": block.Name})
		logger.Info("initialized handler")
	}
	return nil
}

// RunTransports spins off tranpsort + handler processes
func RunTransports(ctx context.Context, wg *sync.WaitGroup, done chan bool, report bool) {
	for name, t := range transports {
		for _, h := range handlers[name] {
			wg.Add(1)
			go func(wg *sync.WaitGroup, h handler.Handler) {
				defer wg.Done()
				h.Run(ctx, metricPublishFunc, eventPublishFunc)
			}(wg, h)
		}

		wg.Add(1)
		go func(wg *sync.WaitGroup, t transport.Transport, name string) {
			defer wg.Done()
			t.Run(ctx, func(blob []byte) {
				for _, h := range handlers[name] {
					err := h.Handle(blob, report, metricPublishFunc, eventPublishFunc)
					if err != nil {
						logger.Metadata(logging.Metadata{"error": err, "handler": fmt.Sprintf("%s[%s]", h.Identify(), name)})
						logger.Debug("failed handling message")
					}
				}
			}, done)
		}(wg, t, name)
	}
}

// RunApplications spins off application processes
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
