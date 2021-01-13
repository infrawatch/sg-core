package main

import (
	"gopkg.in/yaml.v3"
)

type configT struct {
	PluginDir  string
	LogLevel   string `validate:"oneof=error warn info debug"`
	Transports []struct {
		Name     string `validate:"required"`
		Handlers []string
		Config   interface{}
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

var configuration configT = configT{
	PluginDir: "/usr/lib64/sg-core/",
	LogLevel:  "info",
}
