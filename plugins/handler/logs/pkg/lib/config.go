package lib

// LogConfig holds configuration for logs handler
type LogConfig struct {
	MessageField    string `validate:"required" yaml:"messageField"`
	TimestampField  string `validate:"required" yaml:"timestampField"`
	HostnameField   string `validate:"required" yaml:"hostnameField"`
	SeverityField   string `yaml:"severityField"`
	CorrectSeverity bool   `yaml:"correctSeverity"`
}
