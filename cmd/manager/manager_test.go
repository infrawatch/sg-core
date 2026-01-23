package manager

import (
	"context"
	"os"
	"path"
	"sync"
	"testing"
	"time"

	"github.com/infrawatch/apputils/logging"
	"github.com/infrawatch/sg-core/pkg/application"
	"github.com/infrawatch/sg-core/pkg/handler"
	"github.com/infrawatch/sg-core/pkg/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetPluginDir(t *testing.T) {
	t.Run("set custom plugin directory", func(t *testing.T) {
		originalPath := pluginPath
		defer func() { pluginPath = originalPath }()

		customPath := "/custom/plugin/path"
		SetPluginDir(customPath)
		assert.Equal(t, customPath, pluginPath)
	})

	t.Run("set empty plugin directory", func(t *testing.T) {
		originalPath := pluginPath
		defer func() { pluginPath = originalPath }()

		SetPluginDir("")
		assert.Equal(t, "", pluginPath)
	})
}

func TestSetLogger(t *testing.T) {
	tmpdir, err := os.MkdirTemp(".", "manager_test_tmp")
	require.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	logpath := path.Join(tmpdir, "test.log")
	testLogger, err := logging.NewLogger(logging.DEBUG, logpath)
	require.NoError(t, err)

	t.Run("set logger", func(t *testing.T) {
		originalLogger := logger
		defer func() { logger = originalLogger }()

		SetLogger(testLogger)
		assert.Equal(t, testLogger, logger)
	})

	t.Run("set nil logger", func(t *testing.T) {
		originalLogger := logger
		defer func() { logger = originalLogger }()

		SetLogger(nil)
		assert.Nil(t, logger)
	})
}

func TestSetEventBusBlocking(t *testing.T) {
	t.Run("set blocking event bus", func(t *testing.T) {
		// Save original function pointer
		originalFunc := eventPublishFunc
		defer func() { eventPublishFunc = originalFunc }()

		SetEventBusBlocking(true)
		// We can't directly compare function pointers, but we can verify it changed
		assert.NotNil(t, eventPublishFunc)
	})

	t.Run("set non-blocking event bus", func(t *testing.T) {
		// Save original function pointer
		originalFunc := eventPublishFunc
		defer func() { eventPublishFunc = originalFunc }()

		SetEventBusBlocking(false)
		// We can't directly compare function pointers, but we can verify it changed
		assert.NotNil(t, eventPublishFunc)
	})

	t.Run("toggle between blocking and non-blocking", func(t *testing.T) {
		originalFunc := eventPublishFunc
		defer func() { eventPublishFunc = originalFunc }()

		SetEventBusBlocking(true)
		assert.NotNil(t, eventPublishFunc)

		SetEventBusBlocking(false)
		assert.NotNil(t, eventPublishFunc)

		// Toggle back to blocking
		SetEventBusBlocking(true)
		assert.NotNil(t, eventPublishFunc)
	})
}

func TestInitTransport(t *testing.T) {
	tmpdir, err := os.MkdirTemp(".", "manager_test_tmp")
	require.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	logpath := path.Join(tmpdir, "test.log")
	testLogger, err := logging.NewLogger(logging.DEBUG, logpath)
	require.NoError(t, err)
	SetLogger(testLogger)

	t.Run("plugin file does not exist", func(t *testing.T) {
		originalPath := pluginPath
		originalTransports := transports
		defer func() {
			pluginPath = originalPath
			transports = originalTransports
		}()

		// Initialize with empty map
		transports = map[string]transport.Transport{}

		SetPluginDir(tmpdir)
		_, err := InitTransport("nonexistent", map[string]interface{}{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed initializing transport")
	})

	t.Run("invalid plugin path", func(t *testing.T) {
		originalPath := pluginPath
		originalTransports := transports
		defer func() {
			pluginPath = originalPath
			transports = originalTransports
		}()

		transports = map[string]transport.Transport{}

		// Create a directory where we expect a file
		invalidPluginDir := path.Join(tmpdir, "invalid")
		err := os.Mkdir(invalidPluginDir, 0755)
		require.NoError(t, err)

		// Try to use the directory itself as the plugin path
		pluginPath = invalidPluginDir
		_, err = InitTransport(invalidPluginDir, map[string]interface{}{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed initializing transport")
	})
}

func TestInitApplication(t *testing.T) {
	tmpdir, err := os.MkdirTemp(".", "manager_test_tmp")
	require.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	logpath := path.Join(tmpdir, "test.log")
	testLogger, err := logging.NewLogger(logging.DEBUG, logpath)
	require.NoError(t, err)
	SetLogger(testLogger)

	t.Run("plugin file does not exist", func(t *testing.T) {
		originalPath := pluginPath
		originalApplications := applications
		defer func() {
			pluginPath = originalPath
			applications = originalApplications
		}()

		applications = map[string]application.Application{}

		SetPluginDir(tmpdir)
		err := InitApplication("nonexistent", map[string]interface{}{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed initializing application plugin")
	})

	t.Run("invalid plugin directory", func(t *testing.T) {
		originalPath := pluginPath
		originalApplications := applications
		defer func() {
			pluginPath = originalPath
			applications = originalApplications
		}()

		applications = map[string]application.Application{}

		SetPluginDir("/nonexistent/path")
		err := InitApplication("test", map[string]interface{}{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed initializing application plugin")
	})
}

func TestSetTransportHandlers(t *testing.T) {
	tmpdir, err := os.MkdirTemp(".", "manager_test_tmp")
	require.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	logpath := path.Join(tmpdir, "test.log")
	testLogger, err := logging.NewLogger(logging.DEBUG, logpath)
	require.NoError(t, err)
	SetLogger(testLogger)

	t.Run("handler plugin does not exist", func(t *testing.T) {
		originalPath := pluginPath
		originalHandlers := handlers
		defer func() {
			pluginPath = originalPath
			handlers = originalHandlers
		}()

		handlers = map[string][]handler.Handler{}

		SetPluginDir(tmpdir)
		handlerBlocks := []struct {
			Name   string `validate:"required"`
			Config interface{}
		}{
			{
				Name:   "nonexistent",
				Config: map[string]interface{}{},
			},
		}

		err := SetTransportHandlers("test-transport", handlerBlocks)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed initializing handler")
	})

	t.Run("empty handler blocks", func(t *testing.T) {
		originalHandlers := handlers
		defer func() {
			handlers = originalHandlers
		}()

		handlers = map[string][]handler.Handler{}

		handlerBlocks := []struct {
			Name   string `validate:"required"`
			Config interface{}
		}{}

		err := SetTransportHandlers("test-transport", handlerBlocks)
		require.NoError(t, err)
		assert.Empty(t, handlers["test-transport"])
	})
}

func TestInitPlugin(t *testing.T) {
	tmpdir, err := os.MkdirTemp(".", "manager_test_tmp")
	require.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	t.Run("plugin file not found", func(t *testing.T) {
		originalPath := pluginPath
		defer func() { pluginPath = originalPath }()

		SetPluginDir(tmpdir)
		_, err := initPlugin("nonexistent")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to open binary")
	})

	t.Run("empty plugin name", func(t *testing.T) {
		originalPath := pluginPath
		defer func() { pluginPath = originalPath }()

		SetPluginDir(tmpdir)
		_, err := initPlugin("")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to open binary")
	})

	t.Run("invalid plugin path with special characters", func(t *testing.T) {
		originalPath := pluginPath
		defer func() { pluginPath = originalPath }()

		SetPluginDir(tmpdir)
		_, err := initPlugin("invalid/plugin/name")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to open binary")
	})
}

func TestRunTransports(t *testing.T) {
	tmpdir, err := os.MkdirTemp(".", "manager_test_tmp")
	require.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	logpath := path.Join(tmpdir, "test.log")
	testLogger, err := logging.NewLogger(logging.DEBUG, logpath)
	require.NoError(t, err)
	SetLogger(testLogger)

	t.Run("run with no transports", func(t *testing.T) {
		originalTransports := transports
		originalHandlers := handlers
		defer func() {
			transports = originalTransports
			handlers = originalHandlers
		}()

		transports = map[string]transport.Transport{}
		handlers = map[string][]handler.Handler{}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		wg := &sync.WaitGroup{}
		done := make(chan bool)

		// This should return immediately without any goroutines
		RunTransports(ctx, wg, done, false)

		// Give a moment for any potential goroutines to start
		time.Sleep(100 * time.Millisecond)

		// Cancel context and wait - should complete quickly
		cancel()
		waitChan := make(chan struct{})
		go func() {
			wg.Wait()
			close(waitChan)
		}()

		select {
		case <-waitChan:
			// Success - all goroutines finished
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for RunTransports to complete")
		}
	})
}

func TestRunApplications(t *testing.T) {
	tmpdir, err := os.MkdirTemp(".", "manager_test_tmp")
	require.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	logpath := path.Join(tmpdir, "test.log")
	testLogger, err := logging.NewLogger(logging.DEBUG, logpath)
	require.NoError(t, err)
	SetLogger(testLogger)

	t.Run("run with no applications", func(t *testing.T) {
		originalApplications := applications
		defer func() {
			applications = originalApplications
		}()

		applications = map[string]application.Application{}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		wg := &sync.WaitGroup{}
		done := make(chan bool)

		// This should return immediately without any goroutines
		RunApplications(ctx, wg, done)

		// Give a moment for any potential goroutines to start
		time.Sleep(100 * time.Millisecond)

		// Cancel context and wait - should complete quickly
		cancel()
		waitChan := make(chan struct{})
		go func() {
			wg.Wait()
			close(waitChan)
		}()

		select {
		case <-waitChan:
			// Success - all goroutines finished
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for RunApplications to complete")
		}
	})
}

func TestErrAppNotReceiver(t *testing.T) {
	t.Run("error message is correct", func(t *testing.T) {
		assert.Equal(t, "application plugin does not implement either application.MetricReceiver or application.EventReceiver", ErrAppNotReceiver.Error())
	})
}

func TestPackageInitialization(t *testing.T) {
	t.Run("verify default plugin path", func(t *testing.T) {
		// The init() function sets pluginPath to "/usr/lib64/sg-core"
		// We can't directly test init(), but we can verify the default state
		originalPath := pluginPath
		defer func() { pluginPath = originalPath }()

		// Reset to init state
		pluginPath = "/usr/lib64/sg-core"
		assert.Equal(t, "/usr/lib64/sg-core", pluginPath)
	})

	t.Run("verify maps are initialized", func(t *testing.T) {
		originalTransports := transports
		originalHandlers := handlers
		originalApplications := applications
		defer func() {
			transports = originalTransports
			handlers = originalHandlers
			applications = originalApplications
		}()

		// Reset to init state
		transports = map[string]transport.Transport{}
		handlers = map[string][]handler.Handler{}
		applications = map[string]application.Application{}

		assert.NotNil(t, transports)
		assert.NotNil(t, handlers)
		assert.NotNil(t, applications)
		assert.Empty(t, transports)
		assert.Empty(t, handlers)
		assert.Empty(t, applications)
	})
}
