package main

import (
	"gopkg.in/yaml.v3"
)

type configT struct {
	PluginDir     string
	LogLevel      string `validate:"oneof=error warn info debug"`
	HandlerErrors bool
	Transports    []struct {
		Name     string `validate:"required"`
		Handlers []struct {
			Name   string `validate:"required"`
			Config interface{}
		} `validate:"dive"`
		Config interface{}
	} `validate:"dive"`
	Applications []struct {
		Name   string `validate:"required"`
		Config interface{}
	} `validate:"dive"`
}

func (ct *configT) Bytes() []byte {
	res, _ := yaml.Marshal(ct)
	return res
}

var configuration = configT{
	PluginDir:     "/usr/lib64/sg-core/",
	LogLevel:      "info",
	HandlerErrors: false,
}
