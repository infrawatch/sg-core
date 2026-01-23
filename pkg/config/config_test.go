package config

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testConfig struct {
	Host     string `validate:"required"`
	Port     int    `validate:"required"`
	Optional string
}

type nestedConfig struct {
	Database struct {
		Host string `validate:"required"`
		Port int    `validate:"required"`
	} `validate:"required"`
	Server struct {
		Name string `validate:"required"`
	} `validate:"required"`
}

func TestParseConfig(t *testing.T) {
	t.Run("parse valid YAML config", func(t *testing.T) {
		yaml := `
host: localhost
port: 8080
optional: value
`
		var cfg testConfig
		err := ParseConfig(bytes.NewReader([]byte(yaml)), &cfg)
		require.NoError(t, err)
		assert.Equal(t, "localhost", cfg.Host)
		assert.Equal(t, 8080, cfg.Port)
		assert.Equal(t, "value", cfg.Optional)
	})

	t.Run("parse minimal valid config", func(t *testing.T) {
		yaml := `
host: localhost
port: 8080
`
		var cfg testConfig
		err := ParseConfig(bytes.NewReader([]byte(yaml)), &cfg)
		require.NoError(t, err)
		assert.Equal(t, "localhost", cfg.Host)
		assert.Equal(t, 8080, cfg.Port)
		assert.Empty(t, cfg.Optional)
	})

	t.Run("parse null config", func(t *testing.T) {
		yaml := "null\n"
		var cfg testConfig
		err := ParseConfig(bytes.NewReader([]byte(yaml)), &cfg)
		require.NoError(t, err)
		assert.Empty(t, cfg.Host)
		assert.Equal(t, 0, cfg.Port)
	})

	t.Run("error on missing required field", func(t *testing.T) {
		yaml := `
host: localhost
`
		var cfg testConfig
		err := ParseConfig(bytes.NewReader([]byte(yaml)), &cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing or incorrect configuration fields")
		assert.Contains(t, err.Error(), "port")
	})

	t.Run("error on missing multiple required fields", func(t *testing.T) {
		yaml := `
optional: value
`
		var cfg testConfig
		err := ParseConfig(bytes.NewReader([]byte(yaml)), &cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing or incorrect configuration fields")
		assert.Contains(t, err.Error(), "host")
		assert.Contains(t, err.Error(), "port")
	})

	t.Run("error on invalid YAML", func(t *testing.T) {
		yaml := `
host: localhost
port: invalid
	bad indentation
`
		var cfg testConfig
		err := ParseConfig(bytes.NewReader([]byte(yaml)), &cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unmarshalling config yaml")
	})

	t.Run("error on malformed YAML", func(t *testing.T) {
		yaml := `{{{invalid yaml`
		var cfg testConfig
		err := ParseConfig(bytes.NewReader([]byte(yaml)), &cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unmarshalling config yaml")
	})

	t.Run("parse nested config", func(t *testing.T) {
		yaml := `
database:
  host: db.example.com
  port: 5432
server:
  name: web-server
`
		var cfg nestedConfig
		err := ParseConfig(bytes.NewReader([]byte(yaml)), &cfg)
		require.NoError(t, err)
		assert.Equal(t, "db.example.com", cfg.Database.Host)
		assert.Equal(t, 5432, cfg.Database.Port)
		assert.Equal(t, "web-server", cfg.Server.Name)
	})

	t.Run("error on missing nested required field", func(t *testing.T) {
		yaml := `
database:
  host: db.example.com
server:
  name: web-server
`
		var cfg nestedConfig
		err := ParseConfig(bytes.NewReader([]byte(yaml)), &cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing or incorrect configuration fields")
	})

	t.Run("parse config with numbers", func(t *testing.T) {
		yaml := `
host: 192.168.1.1
port: 9090
`
		var cfg testConfig
		err := ParseConfig(bytes.NewReader([]byte(yaml)), &cfg)
		require.NoError(t, err)
		assert.Equal(t, "192.168.1.1", cfg.Host)
		assert.Equal(t, 9090, cfg.Port)
	})

	t.Run("parse config with boolean and special characters", func(t *testing.T) {
		type specialConfig struct {
			Host    string `validate:"required"`
			Enabled bool
			Path    string
		}
		yaml := `
host: "host-name.with-dashes"
enabled: true
path: "/var/lib/data"
`
		var cfg specialConfig
		err := ParseConfig(bytes.NewReader([]byte(yaml)), &cfg)
		require.NoError(t, err)
		assert.Equal(t, "host-name.with-dashes", cfg.Host)
		assert.True(t, cfg.Enabled)
		assert.Equal(t, "/var/lib/data", cfg.Path)
	})

	t.Run("parse empty config", func(t *testing.T) {
		yaml := ``
		var cfg testConfig
		err := ParseConfig(bytes.NewReader([]byte(yaml)), &cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing or incorrect configuration fields")
	})

	t.Run("parse config with extra fields", func(t *testing.T) {
		yaml := `
host: localhost
port: 8080
extra_field: ignored
another_field: also_ignored
`
		var cfg testConfig
		err := ParseConfig(bytes.NewReader([]byte(yaml)), &cfg)
		require.NoError(t, err)
		assert.Equal(t, "localhost", cfg.Host)
		assert.Equal(t, 8080, cfg.Port)
	})

	t.Run("error reading from bad reader", func(t *testing.T) {
		badReader := &errorReader{}
		var cfg testConfig
		err := ParseConfig(badReader, &cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "while reading configuration")
	})
}

func TestSetCamelCase(t *testing.T) {
	t.Run("convert simple field name", func(t *testing.T) {
		input := "Config.Host"
		result := setCamelCase(input)
		assert.Equal(t, "host", result)
	})

	t.Run("convert nested field name", func(t *testing.T) {
		input := "Config.Database.Host"
		result := setCamelCase(input)
		assert.Equal(t, "database.host", result)
	})

	t.Run("convert deeply nested field name", func(t *testing.T) {
		input := "Config.Server.Database.Connection.Host"
		result := setCamelCase(input)
		assert.Equal(t, "server.database.connection.host", result)
	})

	t.Run("convert field with already lowercase first letter", func(t *testing.T) {
		input := "config.host"
		result := setCamelCase(input)
		assert.Equal(t, "host", result)
	})

	t.Run("convert single segment", func(t *testing.T) {
		input := "Config"
		result := setCamelCase(input)
		assert.Equal(t, "", result)
	})

	t.Run("convert field with uppercase words", func(t *testing.T) {
		input := "Config.HTTPServer.Port"
		result := setCamelCase(input)
		assert.Equal(t, "hTTPServer.port", result)
	})

	t.Run("convert multiple segments", func(t *testing.T) {
		input := "TestConfig.Host.Name"
		result := setCamelCase(input)
		assert.Equal(t, "host.name", result)
	})
}

func TestValidate(t *testing.T) {
	t.Run("validator is initialized", func(t *testing.T) {
		require.NotNil(t, Validate)
	})

	t.Run("can validate struct", func(t *testing.T) {
		cfg := testConfig{
			Host: "localhost",
			Port: 8080,
		}
		err := Validate.Struct(cfg)
		assert.NoError(t, err)
	})

	t.Run("validation fails on missing required field", func(t *testing.T) {
		cfg := testConfig{
			Host: "localhost",
		}
		err := Validate.Struct(cfg)
		assert.Error(t, err)
	})
}

// errorReader is a helper type that always returns an error when reading
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, strings.NewReader("").UnreadByte()
}
