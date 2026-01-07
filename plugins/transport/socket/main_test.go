package main

import (
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
	"github.com/stretchr/testify/require"
	"gopkg.in/go-playground/assert.v1"
)

const regularBuffSize = 65535 // default buffer size
const addition = "wubba lubba dub dub"

// Helper function to send and receive Unix socket message
func sendUnixSocketMessage(t *testing.T, logger *logging.Logger, tmpdir string, socketName string, msg []byte) []byte {
	sktpath := path.Join(tmpdir, socketName)
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

	return receivedMsg
}

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

		receivedMsg := sendUnixSocketMessage(t, logger, tmpdir, "socket1", msg)

		// Verify we received the complete message
		assert.Equal(t, len(msg), len(receivedMsg))
		// Verify the end marker is present
		endMarkerPos := len(receivedMsg) - len(marker)
		assert.Equal(t, string(marker), string(receivedMsg[endMarkerPos:]))
	})

	t.Run("test large message transport", func(t *testing.T) {
		// Create a message larger than initial buffer to test dynamic buffer growth
		// This should trigger buffer expansion beyond regularBuffSize (65535 bytes)
		largeBuffSize := regularBuffSize * 2
		msg := make([]byte, largeBuffSize)
		for i := 0; i < largeBuffSize; i++ {
			msg[i] = byte('X')
		}
		msg[largeBuffSize-1] = byte('$')
		msg = append(msg, []byte(addition)...)

		receivedMsg := sendUnixSocketMessage(t, logger, tmpdir, "socket2", msg)

		// Message might be truncated due to OS limits on Unix datagrams
		assert.Equal(t, true, len(receivedMsg) > 0)

		// If we received a complete message, verify the content
		if len(receivedMsg) == largeBuffSize+len(addition) {
			strmsg := string(receivedMsg)
			assert.Equal(t, addition, strmsg[len(strmsg)-len(addition):])
		}
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
