package lib

// HandlerConfig contains validateable configuration
type HandlerConfig struct {
	StrictSource string `yaml:"strictSource" validate:"oneof=generic collectd ceilometer"`
}
