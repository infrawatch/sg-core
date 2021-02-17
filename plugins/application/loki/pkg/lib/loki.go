package lib

import (
	"github.com/infrawatch/apputils/config"
	"github.com/infrawatch/apputils/connector"
	"github.com/infrawatch/apputils/logging"
)

type LokiConfig struct {
	Connection  string `validate:"required"`
	BatchSize   int
	MaxWaitTime int
}

func NewLokiClient(cfg *LokiConfig, logger *logging.Logger) (*connector.LokiConnector, error) {
	conf := createApputilsConfig(cfg, logger)
	client, err := connector.ConnectLoki(conf, logger)
	if err == nil {
		err = client.Connect()
	}
	return client, err
}

// This feels like a hack to me, we might want to come up
// with something else
func createApputilsConfig(cfg *LokiConfig, logger *logging.Logger) config.Config {
	elements := map[string][]config.Parameter{
		"loki": []config.Parameter{
			config.Parameter{Name: "connection", Tag: "", Default: cfg.Connection, Validators: []config.Validator{}},
			config.Parameter{Name: "batch_size", Tag: "", Default: cfg.BatchSize, Validators: []config.Validator{config.IntValidatorFactory()}},
			config.Parameter{Name: "max_wait_time", Tag: "", Default: cfg.MaxWaitTime, Validators: []config.Validator{config.IntValidatorFactory()}},
		},
	}
	conf := config.NewINIConfig(elements, logger)
	conf.Parse("/dev/null")
	return conf
}
