package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"net"
	"os"
	"path"
	"sync"
	"testing"
	"time"

	"github.com/infrawatch/apputils/logging"
	"github.com/infrawatch/sg-core/pkg/data"
	"github.com/stretchr/testify/require"
	"gopkg.in/go-playground/assert.v1"
)

const regularBuffSize = 65535 // default buffer size
const addition = "wubba lubba dub dub"

func TestUnixSocketTransport(t *testing.T) {
	tmpdir, err := os.MkdirTemp(".", "socket_test_tmp")
	require.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	logpath := path.Join(tmpdir, "test.log")
	logger, err := logging.NewLogger(logging.DEBUG, logpath)
	require.NoError(t, err)

	t.Run("test normal message", func(t *testing.T) {
		// Create a normal-sized message (5KB)
		msg := make([]byte, 5000)
		for i := 0; i < len(msg); i++ {
			msg[i] = byte('A')
		}
		marker := []byte("--END--")
		copy(msg[len(msg)-len(marker):], marker)

		sktpath := path.Join(tmpdir, "socket1")
		skt, err := os.OpenFile(sktpath, os.O_RDWR|os.O_CREATE, os.ModeSocket|os.ModePerm)
		require.NoError(t, err)
		defer skt.Close()

		trans := Socket{
			conf: configT{
				Path: sktpath,
			},
			logger: &logWrapper{
				l: logger,
			},
		}

		ctx, cancel := context.WithCancel(context.Background())
		wg := sync.WaitGroup{}
		wg.Add(1)
		var receivedMsg []byte
		go trans.Run(ctx, func(mess []byte) {
			receivedMsg = mess
			wg.Done()
		}, make(chan bool))

		// Wait for socket file to be created
		for {
			stat, err := os.Stat(sktpath)
			require.NoError(t, err)
			if stat.Mode()&os.ModeType == os.ModeSocket {
				break
			}
			time.Sleep(250 * time.Millisecond)
		}

		wskt, err := net.DialUnix("unixgram", nil, &net.UnixAddr{Name: sktpath, Net: "unixgram"})
		require.NoError(t, err)
		_, err = wskt.Write(msg)
		require.NoError(t, err)

		wg.Wait()
		cancel()
		time.Sleep(100 * time.Millisecond)
		wskt.Close()

		// Verify we received the complete message
		assert.Equal(t, len(msg), len(receivedMsg))
		// Verify the end marker is present
		endMarkerPos := len(receivedMsg) - len(marker)
		assert.Equal(t, string(marker), string(receivedMsg[endMarkerPos:]))
	})

	t.Run("test large message transport", func(t *testing.T) {
		// Create a message larger than initial buffer to test dynamic buffer growth
		largeBuffSize := regularBuffSize * 2 // 131070 bytes
		msg := make([]byte, largeBuffSize)
		for i := 0; i < largeBuffSize; i++ {
			msg[i] = byte('X')
		}
		msg[largeBuffSize-1] = byte('$')
		msg = append(msg, []byte(addition)...) // Total: 131089 bytes

		// Setup socket using same pattern as sendUnixSocketMessage
		sktpath := path.Join(tmpdir, "socket2")
		skt, err := os.OpenFile(sktpath, os.O_RDWR|os.O_CREATE, os.ModeSocket|os.ModePerm)
		require.NoError(t, err)
		defer skt.Close()

		trans := Socket{
			conf: configT{
				Path: sktpath,
			},
			logger: &logWrapper{
				l: logger,
			},
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var receivedMsgs [][]byte
		var mutex sync.Mutex
		wg := sync.WaitGroup{}
		wg.Add(3) // Expecting 3 messages

		go trans.Run(ctx, func(mess []byte) {
			mutex.Lock()
			receivedMsgs = append(receivedMsgs, mess)
			mutex.Unlock()
			wg.Done()
		}, make(chan bool))

		// Wait for socket file to be created
		for {
			stat, err := os.Stat(sktpath)
			require.NoError(t, err)
			if stat.Mode()&os.ModeType == os.ModeSocket {
				break
			}
			time.Sleep(250 * time.Millisecond)
		}

		wskt, err := net.DialUnix("unixgram", nil, &net.UnixAddr{Name: sktpath, Net: "unixgram"})
		require.NoError(t, err)
		defer wskt.Close()

		// Send the same message 3 times
		_, err = wskt.Write(msg)
		require.NoError(t, err)
		time.Sleep(100 * time.Millisecond)

		_, err = wskt.Write(msg)
		require.NoError(t, err)
		time.Sleep(100 * time.Millisecond)

		_, err = wskt.Write(msg)
		require.NoError(t, err)

		wg.Wait()

		// Verify we received 3 messages
		require.Equal(t, 3, len(receivedMsgs))

		// First message: the message is truncated to the maximum 64KB (65535 bytes)
		require.Equal(t, len(receivedMsgs[0]), regularBuffSize)

		// Second message: check for 128KB (131070 bytes) with '$' at position 131069
		require.Equal(t, len(receivedMsgs[1]), largeBuffSize)
		assert.Equal(t, byte('$'), receivedMsgs[1][131069])

		// Third message: check for > 128KB (131070 bytes) with "wubba lubba dub dub" at the end
		require.GreaterOrEqual(t, len(receivedMsgs[2]), largeBuffSize+len(addition))
		endStr := string(receivedMsgs[2][len(receivedMsgs[2])-len(addition):])
		assert.Equal(t, addition, endStr)
	})
}

// Helper function to send and receive UDP socket message
func sendUDPSocketMessage(t *testing.T, logger *logging.Logger, addr string, msg []byte) ([]byte, error) {
	trans := Socket{
		conf: configT{
			Socketaddr: addr,
			Type:       "udp",
		},
		logger: &logWrapper{
			l: logger,
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	wg.Add(1)
	var receivedMsg []byte
	messageReceived := false
	go trans.Run(ctx, func(mess []byte) {
		receivedMsg = mess
		messageReceived = true
		wg.Done()
	}, make(chan bool))

	// Wait for socket to be ready
	time.Sleep(100 * time.Millisecond)

	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	require.NoError(t, err)
	wskt, err := net.DialUDP("udp", nil, udpAddr)
	require.NoError(t, err)
	_, writeErr := wskt.Write(msg)

	if writeErr == nil && messageReceived {
		wg.Wait()
	}
	cancel()
	time.Sleep(100 * time.Millisecond)
	wskt.Close()

	return receivedMsg, writeErr
}

func TestUdpSocketTransport(t *testing.T) {
	tmpdir, err := os.MkdirTemp(".", "socket_test_tmp")
	require.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	logpath := path.Join(tmpdir, "test.log")
	logger, err := logging.NewLogger(logging.DEBUG, logpath)
	require.NoError(t, err)

	t.Run("test normal message", func(t *testing.T) {
		// Create a normal message (5KB)
		msg := make([]byte, 5000)
		for i := 0; i < len(msg); i++ {
			msg[i] = byte('U')
		}
		marker := []byte("--UDP-END--")
		copy(msg[len(msg)-len(marker):], marker)

		receivedMsg, err := sendUDPSocketMessage(t, logger, "127.0.0.1:8650", msg)
		require.NoError(t, err)

		// Verify we received the complete message
		assert.Equal(t, len(msg), len(receivedMsg))
		// Verify the end marker is present
		endMarkerPos := len(receivedMsg) - len(marker)
		assert.Equal(t, string(marker), string(receivedMsg[endMarkerPos:]))
	})

	t.Run("test large message transport", func(t *testing.T) {
		// Create message that exceeds UDP datagram limits
		// UDP max payload is ~65507 bytes, we're trying to send 65535 + 19 = 65554 bytes
		largeBuffSize := regularBuffSize - len(addition)
		msg := make([]byte, largeBuffSize)
		for i := 0; i < largeBuffSize; i++ {
			msg[i] = byte('X')
		}
		msg[largeBuffSize-1] = byte('$')
		msg = append(msg, []byte(addition)...)

		_, err := sendUDPSocketMessage(t, logger, "127.0.0.1:8652", msg)

		// Verify that sending a message that's too large for UDP fails
		require.Error(t, err)
	})
}

// Helper function to connect to TCP with retries
func connectTCPWithRetry(t *testing.T, addr string) net.Conn {
	wskt, err := net.Dial("tcp", addr)
	if err != nil {
		for retries := 0; err != nil && retries < 3; retries++ {
			time.Sleep(500 * time.Millisecond)
			wskt, err = net.Dial("tcp", addr)
		}
	}
	require.NoError(t, err)
	return wskt
}

// Helper function to create a TCP message with length header
func createTCPMessage(t *testing.T, content []byte) []byte {
	msgLength := new(bytes.Buffer)
	err := binary.Write(msgLength, binary.LittleEndian, uint64(len(content)))
	require.NoError(t, err)
	return append(msgLength.Bytes(), content...)
}

// Helper function to send and verify TCP socket message with marker
func sendTCPSocketMessage(t *testing.T, logger *logging.Logger, addr string, msgSize int, fillByte byte, marker []byte) {
	trans := Socket{
		conf: configT{
			Socketaddr: addr,
			Type:       "tcp",
		},
		logger: &logWrapper{
			l: logger,
		},
	}

	msgContent := make([]byte, msgSize)
	for i := 0; i < msgSize; i++ {
		msgContent[i] = fillByte
	}
	copy(msgContent[len(msgContent)-len(marker):], marker)

	fullMsg := createTCPMessage(t, msgContent)

	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	wg.Add(1)
	go trans.Run(ctx, func(mess []byte) {
		assert.Equal(t, msgSize, len(mess))
		endMarkerPos := len(mess) - len(marker)
		assert.Equal(t, string(marker), string(mess[endMarkerPos:]))
		wg.Done()
	}, make(chan bool))

	time.Sleep(100 * time.Millisecond)

	wskt := connectTCPWithRetry(t, addr)
	_, err := wskt.Write(fullMsg)
	require.NoError(t, err)

	wg.Wait()
	cancel()
	time.Sleep(100 * time.Millisecond)
	wskt.Close()
}

func TestTcpSocketTransport(t *testing.T) {
	tmpdir, err := os.MkdirTemp(".", "socket_test_tmp")
	require.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	logpath := path.Join(tmpdir, "test.log")
	logger, err := logging.NewLogger(logging.DEBUG, logpath)
	require.NoError(t, err)

	t.Run("test normal message", func(t *testing.T) {
		// Create a normal message (5KB)
		sendTCPSocketMessage(t, logger, "127.0.0.1:8660", 5000, 'T', []byte("--TCP-END--"))
	})

	t.Run("test message exceeding initial buffer", func(t *testing.T) {
		// Create a message larger than initial buffer (100KB)
		sendTCPSocketMessage(t, logger, "127.0.0.1:8661", 100000, 'B', []byte("--LARGE-TCP--"))
	})

	t.Run("test multiple large messages", func(t *testing.T) {
		trans := Socket{
			conf: configT{
				Socketaddr: "127.0.0.1:8663",
				Type:       "tcp",
			},
			logger: &logWrapper{
				l: logger,
			},
		}

		numMessages := 3
		messageSizes := []int{80000, 120000, 90000}
		var combinedMsg bytes.Buffer

		// Create multiple large messages
		for i := 0; i < numMessages; i++ {
			msgContent := make([]byte, messageSizes[i])
			fillByte := byte('0' + i)
			for j := 0; j < messageSizes[i]; j++ {
				msgContent[j] = fillByte
			}
			combinedMsg.Write(createTCPMessage(t, msgContent))
		}

		// Setup message verification
		ctx, cancel := context.WithCancel(context.Background())
		receivedCount := 0
		var mutex sync.Mutex
		wg := sync.WaitGroup{}
		wg.Add(numMessages)

		go trans.Run(ctx, func(mess []byte) {
			mutex.Lock()
			defer mutex.Unlock()

			// Verify message size matches one of our expected sizes
			found := false
			for i, expectedSize := range messageSizes {
				if len(mess) == expectedSize {
					expectedByte := byte('0' + i)
					allMatch := true
					for _, b := range mess {
						if b != expectedByte {
							allMatch = false
							break
						}
					}
					if allMatch {
						found = true
						receivedCount++
						wg.Done()
						break
					}
				}
			}
			assert.Equal(t, true, found)
		}, make(chan bool))

		// Wait for socket to be ready
		time.Sleep(100 * time.Millisecond)

		// Connect and send all messages
		wskt := connectTCPWithRetry(t, "127.0.0.1:8663")
		_, err = wskt.Write(combinedMsg.Bytes())
		require.NoError(t, err)

		wg.Wait()

		mutex.Lock()
		assert.Equal(t, numMessages, receivedCount)
		mutex.Unlock()

		cancel()
		time.Sleep(100 * time.Millisecond)
		wskt.Close()
	})

	t.Run("test large message transport multiple connections", func(t *testing.T) {
		trans := Socket{
			conf: configT{
				Socketaddr: "127.0.0.1:8665",
				Type:       "tcp",
			},
			logger: &logWrapper{
				l: logger,
			},
		}

		msgContent := make([]byte, regularBuffSize)
		for i := 0; i < regularBuffSize; i++ {
			msgContent[i] = byte('X')
		}
		msgContent[regularBuffSize-1] = byte('$')
		msgContent = append(msgContent, []byte(addition)...)
		msg := createTCPMessage(t, msgContent)

		// verify transport
		ctx, cancel := context.WithCancel(context.Background())
		wg := sync.WaitGroup{}
		wg.Add(2)
		go trans.Run(ctx, func(mess []byte) {
			strmsg := string(mess)
			assert.Equal(t, regularBuffSize+len(addition), len(strmsg))   // we received whole message
			assert.Equal(t, addition, strmsg[len(strmsg)-len(addition):]) // and the out-of-band part is correct
			wg.Done()
		}, make(chan bool))

		// Wait for socket to be ready
		time.Sleep(100 * time.Millisecond)

		// write to socket
		wskt1 := connectTCPWithRetry(t, "127.0.0.1:8665")

		// We shouldn't need to retry the second connection, if this fails, then something is wrong
		wskt2, err := net.Dial("tcp", "127.0.0.1:8665")
		require.NoError(t, err)

		_, err = wskt1.Write(msg)
		require.NoError(t, err)
		_, err = wskt2.Write(msg)
		require.NoError(t, err)

		wg.Wait()
		cancel()
		time.Sleep(100 * time.Millisecond)
		wskt1.Close()
		wskt2.Close()
	})
}

func TestNew(t *testing.T) {
	tmpdir, err := os.MkdirTemp(".", "socket_test_tmp")
	require.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	logpath := path.Join(tmpdir, "test.log")
	logger, err := logging.NewLogger(logging.DEBUG, logpath)
	require.NoError(t, err)

	trans := New(logger)
	require.NotNil(t, trans)

	socket, ok := trans.(*Socket)
	require.True(t, ok)
	require.NotNil(t, socket.logger)
	require.NotNil(t, socket.logger.l)
}

func TestListen(t *testing.T) {
	tmpdir, err := os.MkdirTemp(".", "socket_test_tmp")
	require.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	logpath := path.Join(tmpdir, "test.log")
	logger, err := logging.NewLogger(logging.DEBUG, logpath)
	require.NoError(t, err)

	trans := New(logger)
	socket := trans.(*Socket)

	// Listen should not panic and should print the event
	testEvent := data.Event{
		Index:     "test-index",
		Time:      123.456,
		Type:      data.EVENT,
		Publisher: "test-publisher",
		Severity:  data.INFO,
	}
	socket.Listen(testEvent)
}

func TestConfig(t *testing.T) {
	tmpdir, err := os.MkdirTemp(".", "socket_test_tmp")
	require.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	logpath := path.Join(tmpdir, "test.log")
	logger, err := logging.NewLogger(logging.DEBUG, logpath)
	require.NoError(t, err)

	t.Run("valid unix socket config", func(t *testing.T) {
		trans := New(logger)
		socket := trans.(*Socket)

		config := `
type: unix
path: /tmp/test.sock
`
		err := socket.Config([]byte(config))
		require.NoError(t, err)
		assert.Equal(t, "unix", socket.conf.Type)
		assert.Equal(t, "/tmp/test.sock", socket.conf.Path)
	})

	t.Run("valid udp socket config", func(t *testing.T) {
		trans := New(logger)
		socket := trans.(*Socket)

		config := `
type: udp
socketaddr: 127.0.0.1:9999
`
		err := socket.Config([]byte(config))
		require.NoError(t, err)
		assert.Equal(t, "udp", socket.conf.Type)
		assert.Equal(t, "127.0.0.1:9999", socket.conf.Socketaddr)
	})

	t.Run("valid tcp socket config", func(t *testing.T) {
		trans := New(logger)
		socket := trans.(*Socket)

		config := `
type: tcp
socketaddr: 127.0.0.1:8888
`
		err := socket.Config([]byte(config))
		require.NoError(t, err)
		assert.Equal(t, "tcp", socket.conf.Type)
		assert.Equal(t, "127.0.0.1:8888", socket.conf.Socketaddr)
	})

	t.Run("config with dump messages enabled", func(t *testing.T) {
		trans := New(logger)
		socket := trans.(*Socket)

		dumpPath := path.Join(tmpdir, "dump.txt")
		config := `
type: unix
path: /tmp/test.sock
dumpMessages:
  enabled: true
  path: ` + dumpPath
		err := socket.Config([]byte(config))
		require.NoError(t, err)
		assert.Equal(t, true, socket.conf.DumpMessages.Enabled)
		assert.Equal(t, dumpPath, socket.conf.DumpMessages.Path)
		require.NotNil(t, socket.dumpFile)
		require.NotNil(t, socket.dumpBuf)
		socket.dumpFile.Close()
	})

	t.Run("invalid socket type", func(t *testing.T) {
		trans := New(logger)
		socket := trans.(*Socket)

		config := `
type: invalid
path: /tmp/test.sock
`
		err := socket.Config([]byte(config))
		require.Error(t, err)
		require.Contains(t, err.Error(), "unable to determine socket type")
	})

	t.Run("unix socket without path", func(t *testing.T) {
		trans := New(logger)
		socket := trans.(*Socket)

		config := `
type: unix
`
		err := socket.Config([]byte(config))
		require.Error(t, err)
		require.Contains(t, err.Error(), "path")
	})

	t.Run("udp socket without socketaddr", func(t *testing.T) {
		trans := New(logger)
		socket := trans.(*Socket)

		config := `
type: udp
`
		err := socket.Config([]byte(config))
		require.Error(t, err)
		require.Contains(t, err.Error(), "socketaddr")
	})

	t.Run("tcp socket without socketaddr", func(t *testing.T) {
		trans := New(logger)
		socket := trans.(*Socket)

		config := `
type: tcp
`
		err := socket.Config([]byte(config))
		require.Error(t, err)
		require.Contains(t, err.Error(), "socketaddr")
	})

	t.Run("invalid yaml config", func(t *testing.T) {
		trans := New(logger)
		socket := trans.(*Socket)

		config := `
this is not: valid: yaml
`
		err := socket.Config([]byte(config))
		require.Error(t, err)
	})

	t.Run("case insensitive socket type", func(t *testing.T) {
		trans := New(logger)
		socket := trans.(*Socket)

		config := `
type: TCP
socketaddr: 127.0.0.1:8888
`
		err := socket.Config([]byte(config))
		require.NoError(t, err)
		assert.Equal(t, "tcp", socket.conf.Type)
	})

	t.Run("default values", func(t *testing.T) {
		trans := New(logger)
		socket := trans.(*Socket)

		config := `
path: /tmp/test.sock
`
		err := socket.Config([]byte(config))
		require.NoError(t, err)
		assert.Equal(t, "unix", socket.conf.Type)
		assert.Equal(t, false, socket.conf.DumpMessages.Enabled)
		assert.Equal(t, "/dev/stdout", socket.conf.DumpMessages.Path)
	})
}

func TestInitializationErrors(t *testing.T) {
	tmpdir, err := os.MkdirTemp(".", "socket_test_tmp")
	require.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	logpath := path.Join(tmpdir, "test.log")
	logger, err := logging.NewLogger(logging.DEBUG, logpath)
	require.NoError(t, err)

	t.Run("unix socket initialization with path in non-existent directory", func(t *testing.T) {
		// Create a file where we want to create a directory, causing mkdir to fail
		blockingFile := path.Join(tmpdir, "blocking_file")
		err := os.WriteFile(blockingFile, []byte("test"), 0644)
		require.NoError(t, err)

		// Try to create a socket in a "subdirectory" of this file (which is impossible)
		invalidPath := path.Join(blockingFile, "subdir", "socket.sock")

		trans := Socket{
			conf: configT{
				Path: invalidPath,
				Type: unix,
			},
			logger: &logWrapper{
				l: logger,
			},
		}

		result := trans.initUnixSocket()
		require.Nil(t, result)
	})

	t.Run("udp socket initialization with invalid address", func(t *testing.T) {
		trans := Socket{
			conf: configT{
				Socketaddr: "not-a-valid-address:::::99999",
				Type:       udp,
			},
			logger: &logWrapper{
				l: logger,
			},
		}

		result := trans.initUDPSocket()
		require.Nil(t, result)
	})

	t.Run("udp socket initialization with address already in use", func(t *testing.T) {
		// First, bind to a port
		addr, err := net.ResolveUDPAddr(udp, "127.0.0.1:18680")
		require.NoError(t, err)
		firstConn, err := net.ListenUDP(udp, addr)
		require.NoError(t, err)
		defer firstConn.Close()

		// Now try to bind to the same port
		trans := Socket{
			conf: configT{
				Socketaddr: "127.0.0.1:18680",
				Type:       udp,
			},
			logger: &logWrapper{
				l: logger,
			},
		}

		result := trans.initUDPSocket()
		require.Nil(t, result)
	})

	t.Run("tcp socket initialization with invalid address", func(t *testing.T) {
		trans := Socket{
			conf: configT{
				Socketaddr: "not-a-valid-address:::::99999",
				Type:       tcp,
			},
			logger: &logWrapper{
				l: logger,
			},
		}

		result := trans.initTCPSocket()
		require.Nil(t, result)
	})

	t.Run("tcp socket initialization with address already in use", func(t *testing.T) {
		// First, bind to a port
		addr, err := net.ResolveTCPAddr(tcp, "127.0.0.1:18681")
		require.NoError(t, err)
		firstListener, err := net.ListenTCP(tcp, addr)
		require.NoError(t, err)
		defer firstListener.Close()

		// Now try to bind to the same port
		trans := Socket{
			conf: configT{
				Socketaddr: "127.0.0.1:18681",
				Type:       tcp,
			},
			logger: &logWrapper{
				l: logger,
			},
		}

		result := trans.initTCPSocket()
		require.Nil(t, result)
	})
}

func TestDumpMessagesFeature(t *testing.T) {
	tmpdir, err := os.MkdirTemp(".", "socket_test_tmp")
	require.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	logpath := path.Join(tmpdir, "test.log")
	logger, err := logging.NewLogger(logging.DEBUG, logpath)
	require.NoError(t, err)

	t.Run("unix socket with dump messages enabled", func(t *testing.T) {
		dumpPath := path.Join(tmpdir, "dump_unix.txt")
		sktpath := path.Join(tmpdir, "socket_dump")
		skt, err := os.OpenFile(sktpath, os.O_RDWR|os.O_CREATE, os.ModeSocket|os.ModePerm)
		require.NoError(t, err)
		defer skt.Close()

		trans := Socket{
			conf: configT{
				Path: sktpath,
				Type: unix,
				DumpMessages: struct {
					Enabled bool
					Path    string
				}{
					Enabled: true,
					Path:    dumpPath,
				},
			},
			logger: &logWrapper{
				l: logger,
			},
		}

		// Initialize dump file and buffer
		trans.dumpFile, err = os.OpenFile(dumpPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		require.NoError(t, err)
		defer trans.dumpFile.Close()
		trans.dumpBuf = bufio.NewWriter(trans.dumpFile)

		ctx, cancel := context.WithCancel(context.Background())
		wg := sync.WaitGroup{}
		wg.Add(1)
		var receivedMsg []byte
		go trans.Run(ctx, func(mess []byte) {
			receivedMsg = mess
			wg.Done()
		}, make(chan bool))

		// Wait for socket file to be created
		for {
			stat, err := os.Stat(sktpath)
			require.NoError(t, err)
			if stat.Mode()&os.ModeType == os.ModeSocket {
				break
			}
			time.Sleep(250 * time.Millisecond)
		}

		// Send a message
		msg := []byte("test message with dump")
		wskt, err := net.DialUnix("unixgram", nil, &net.UnixAddr{Name: sktpath, Net: "unixgram"})
		require.NoError(t, err)
		_, err = wskt.Write(msg)
		require.NoError(t, err)

		wg.Wait()
		cancel()
		time.Sleep(100 * time.Millisecond)
		wskt.Close()

		// Verify message was received
		assert.Equal(t, string(msg), string(receivedMsg))

		// Verify message was dumped to file
		dumpContent, err := os.ReadFile(dumpPath)
		require.NoError(t, err)
		require.Contains(t, string(dumpContent), "test message with dump")
	})

	t.Run("tcp socket with dump messages enabled", func(t *testing.T) {
		dumpPath := path.Join(tmpdir, "dump_tcp.txt")
		trans := Socket{
			conf: configT{
				Socketaddr: "127.0.0.1:18690",
				Type:       tcp,
				DumpMessages: struct {
					Enabled bool
					Path    string
				}{
					Enabled: true,
					Path:    dumpPath,
				},
			},
			logger: &logWrapper{
				l: logger,
			},
		}

		// Initialize dump file and buffer
		trans.dumpFile, err = os.OpenFile(dumpPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		require.NoError(t, err)
		defer trans.dumpFile.Close()
		trans.dumpBuf = bufio.NewWriter(trans.dumpFile)

		msgContent := []byte("tcp dump test message")
		fullMsg := createTCPMessage(t, msgContent)

		ctx, cancel := context.WithCancel(context.Background())
		wg := sync.WaitGroup{}
		wg.Add(1)
		go trans.Run(ctx, func(mess []byte) {
			assert.Equal(t, string(msgContent), string(mess))
			wg.Done()
		}, make(chan bool))

		time.Sleep(100 * time.Millisecond)

		wskt := connectTCPWithRetry(t, "127.0.0.1:18690")
		_, err = wskt.Write(fullMsg)
		require.NoError(t, err)

		wg.Wait()
		cancel()
		time.Sleep(100 * time.Millisecond)
		wskt.Close()

		// Verify message was dumped to file
		dumpContent, err := os.ReadFile(dumpPath)
		require.NoError(t, err)
		require.Contains(t, string(dumpContent), "tcp dump test message")
	})
}

func TestWriteTCPMsgErrors(t *testing.T) {
	tmpdir, err := os.MkdirTemp(".", "socket_test_tmp")
	require.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	logpath := path.Join(tmpdir, "test.log")
	logger, err := logging.NewLogger(logging.DEBUG, logpath)
	require.NoError(t, err)

	t.Run("overflow protection - negative length", func(t *testing.T) {
		trans := Socket{
			conf: configT{
				Socketaddr: "127.0.0.1:8670",
				Type:       "tcp",
			},
			logger: &logWrapper{
				l: logger,
			},
		}

		// Create a buffer with a message that would cause overflow
		msgBuffer := make([]byte, 100)
		// Write a very large length value that will overflow when added to position
		binary.LittleEndian.PutUint64(msgBuffer[0:8], uint64(0x7FFFFFFFFFFFFFFF))

		messageCount := 0
		pos, err := trans.WriteTCPMsg(func(data []byte) {
			messageCount++
		}, msgBuffer, len(msgBuffer))

		require.NoError(t, err)
		// Should stop without processing any messages due to overflow protection
		assert.Equal(t, 0, messageCount)
		assert.Equal(t, int64(0), pos)
	})

	t.Run("incomplete message - not enough data", func(t *testing.T) {
		trans := Socket{
			conf: configT{
				Socketaddr: "127.0.0.1:8671",
				Type:       "tcp",
			},
			logger: &logWrapper{
				l: logger,
			},
		}

		// Create a buffer with message length header indicating more data than available
		msgBuffer := make([]byte, 20)
		// Indicate 100 bytes of data, but we only have 12 bytes after the length header
		binary.LittleEndian.PutUint64(msgBuffer[0:8], uint64(100))
		copy(msgBuffer[8:], []byte("test"))

		messageCount := 0
		pos, err := trans.WriteTCPMsg(func(data []byte) {
			messageCount++
		}, msgBuffer, len(msgBuffer))

		require.NoError(t, err)
		// Should not process the incomplete message
		assert.Equal(t, 0, messageCount)
		assert.Equal(t, int64(0), pos)
	})

	t.Run("multiple messages with partial last message", func(t *testing.T) {
		trans := Socket{
			conf: configT{
				Socketaddr: "127.0.0.1:8672",
				Type:       "tcp",
			},
			logger: &logWrapper{
				l: logger,
			},
		}

		var msgBuffer bytes.Buffer

		// First complete message
		msg1 := []byte("Complete message 1")
		binary.Write(&msgBuffer, binary.LittleEndian, uint64(len(msg1)))
		msgBuffer.Write(msg1)

		// Second complete message
		msg2 := []byte("Complete message 2")
		binary.Write(&msgBuffer, binary.LittleEndian, uint64(len(msg2)))
		msgBuffer.Write(msg2)

		// Third incomplete message (header indicates more data than available)
		binary.Write(&msgBuffer, binary.LittleEndian, uint64(1000))
		msgBuffer.Write([]byte("Incomplete"))

		receivedMessages := []string{}
		pos, err := trans.WriteTCPMsg(func(data []byte) {
			receivedMessages = append(receivedMessages, string(data))
		}, msgBuffer.Bytes(), msgBuffer.Len())

		require.NoError(t, err)
		// Should process only the two complete messages
		assert.Equal(t, 2, len(receivedMessages))
		assert.Equal(t, "Complete message 1", receivedMessages[0])
		assert.Equal(t, "Complete message 2", receivedMessages[1])
		// Position should be at the start of the incomplete message
		expectedPos := int64(8 + len(msg1) + 8 + len(msg2))
		assert.Equal(t, expectedPos, pos)
	})
}
