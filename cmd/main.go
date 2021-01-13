package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sync"
	"syscall"

	"github.com/infrawatch/apputils/logging"
	log "github.com/infrawatch/apputils/logging"
	"github.com/infrawatch/apputils/system"
	"github.com/infrawatch/sg-core-refactor/cmd/manager"
	"github.com/infrawatch/sg-core-refactor/pkg/config"
)

func main() {
	configPath := flag.String("config", "/etc/sg-core.conf.yaml", "configuration file path")
	//logLevel := flag.String("logLevel", "ERROR", "log level")
	flag.Usage = func() {
		fmt.Printf("Usage: %s [OPTIONS]\n\nAvailable options:\n", os.Args[0])
		flag.PrintDefaults()

		fmt.Printf("\n\nDefault configurations:\n\n%s", string(configuration.Bytes()))
	}
	flag.Parse()

	logger, err := log.NewLogger(log.DEBUG, "console")
	if err != nil {
		fmt.Printf("failed initializing logger: %s", err)
	}
	logger.Timestamp = true

	file, err := os.Open(*configPath)
	if err != nil {
		logger.Metadata(log.Metadata{"error": err})
		logger.Error("failed opening configuration file")
		return
	}

	err = config.ParseConfig(file, &configuration)
	if err != nil {
		logger.Metadata(log.Metadata{"error": err})
		logger.Error("failed parsing config file")
		return
	}

	logger.SetLogLevel(map[string]logging.LogLevel{
		"error": logging.ERROR,
		"warn":  logging.WARN,
		"info":  logging.INFO,
		"debug": logging.DEBUG,
	}[configuration.LogLevel])

	manager.SetLogger(logger)
	manager.SetPluginDir(configuration.PluginDir)

	for _, tConfig := range configuration.Transports {
		err = manager.InitTransport(tConfig.Name, tConfig.Config)
		if err != nil {
			logger.Metadata(log.Metadata{"transport": tConfig.Name, "error": err})
			logger.Error("failed configuring transport")
			continue
		}
		err = manager.SetTransportHandlers(tConfig.Name, tConfig.Handlers)
		if err != nil {
			logger.Metadata(log.Metadata{"transport": tConfig.Name, "error": err})
			logger.Error("transport handlers failed to load")
			continue
		}
		logger.Metadata(log.Metadata{"transport": tConfig.Name})
		logger.Info("loaded transport")
	}

	for _, aConfig := range configuration.Applications {
		err = manager.InitApplication(aConfig.Name, aConfig.Config)
		if err != nil {
			logger.Metadata(log.Metadata{"application": aConfig.Name, "error": err})
			logger.Error("failed configuring application")
			continue
		}
		logger.Metadata(log.Metadata{"application": aConfig.Name})
		logger.Info("loaded application plugin")
	}

	if err != nil {
		return
	}

	ctx, cancelCtx := context.WithCancel(context.Background())
	wg := new(sync.WaitGroup)
	//run main processes

	pluginDone := make(chan bool) //notified if a plugin stops execution before main or interrupt recieved
	interrupt := make(chan bool)
	manager.RunTransports(ctx, wg, pluginDone)
	manager.RunApplications(ctx, wg, pluginDone)
	system.SpawnSignalHandler(interrupt, logger, syscall.SIGINT, syscall.SIGKILL)

	for {
		select {
		case <-pluginDone:
			goto done
		case <-interrupt:
			goto done
		}
	}

done:
	cancelCtx()
	wg.Wait()
	logger.Info("sg-core exited cleanly")
}
