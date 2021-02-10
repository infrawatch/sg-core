package lib

//AppConfig holds configuration for Elasticsearch client
type AppConfig struct {
	HostURL       string
	UseTLS        bool
	TLSServerName string
	TLSClientCert string
	TLSClientKey  string
	TLSCaCert     string
	UseBasicAuth  bool
	User          string
	Password      string
	ResetIndex    bool
}
