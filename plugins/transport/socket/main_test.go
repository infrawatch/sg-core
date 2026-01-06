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

const regularBuffSize = 16384
const addition = "wubba lubba dub dub"

func TestUnixSocketTransport(t *testing.T) {
	tmpdir, err := os.MkdirTemp(".", "socket_test_tmp")
	require.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	logpath := path.Join(tmpdir, "test.log")
	logger, err := logging.NewLogger(logging.DEBUG, logpath)
	require.NoError(t, err)

	sktpath := path.Join(tmpdir, "socket")
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

	t.Run("test small buffer with message within buffer size", func(t *testing.T) {
		sktpath2 := path.Join(tmpdir, "socket2")
		skt2, err := os.OpenFile(sktpath2, os.O_RDWR|os.O_CREATE, os.ModeSocket|os.ModePerm)
		require.NoError(t, err)
		defer skt2.Close()

		// Configure a small buffer size
		smallBufferTrans := Socket{
			conf: configT{
				Path:       sktpath2,
				BufferSize: 1500,
			},
			logger: &logWrapper{
				l: logger,
			},
		}

		// Create a message smaller than the buffer size (1200 bytes)
		msg := make([]byte, 1200)
		for i := 0; i < 1200; i++ {
			msg[i] = byte('U')
		}
		marker := []byte("--UNIX-END--")
		copy(msg[len(msg)-len(marker):], marker)

		// Setup message verification
		ctx, cancel := context.WithCancel(context.Background())
		wg := sync.WaitGroup{}
		go smallBufferTrans.Run(ctx, func(mess []byte) {
			wg.Add(1)
			// Verify we received the complete message
			assert.Equal(t, 1200, len(mess))
			// Verify the end marker is present
			endMarkerPos := len(mess) - len(marker)
			assert.Equal(t, string(marker), string(mess[endMarkerPos:]))
			wg.Done()
		}, make(chan bool))

		// Wait for socket file to be created
		for {
			stat, err := os.Stat(sktpath2)
			require.NoError(t, err)
			if stat.Mode()&os.ModeType == os.ModeSocket {
				break
			}
			time.Sleep(250 * time.Millisecond)
		}

		// Send the message
		wskt, err := net.DialUnix("unixgram", nil, &net.UnixAddr{Name: sktpath2, Net: "unixgram"})
		require.NoError(t, err)
		_, err = wskt.Write(msg)
		require.NoError(t, err)

		// Wait for message processing
		time.Sleep(100 * time.Millisecond)

		cancel()
		wg.Wait()
		wskt.Close()
	})

	t.Run("test small buffer with multiple messages", func(t *testing.T) {
		sktpath3 := path.Join(tmpdir, "socket3")
		skt3, err := os.OpenFile(sktpath3, os.O_RDWR|os.O_CREATE, os.ModeSocket|os.ModePerm)
		require.NoError(t, err)
		defer skt3.Close()

		// Configure a small buffer size
		smallBufferTrans := Socket{
			conf: configT{
				Path:       sktpath3,
				BufferSize: 1500,
			},
			logger: &logWrapper{
				l: logger,
			},
		}

		// Create multiple messages that fit within buffer
		numMessages := 3
		expectedSizes := []int{800, 1200, 900}
		messages := make([][]byte, numMessages)

		for i := 0; i < numMessages; i++ {
			messages[i] = make([]byte, expectedSizes[i])
			// Fill with a pattern unique to each message
			fillByte := byte('0' + i)
			for j := 0; j < expectedSizes[i]; j++ {
				messages[i][j] = fillByte
			}
		}

		// Setup message verification
		ctx, cancel := context.WithCancel(context.Background())
		receivedCount := 0
		var mutex sync.Mutex
		wg := sync.WaitGroup{}
		wg.Add(numMessages)

		go smallBufferTrans.Run(ctx, func(mess []byte) {
			mutex.Lock()
			defer mutex.Unlock()

			// Verify message size matches one of our expected sizes
			found := false
			for i, expectedSize := range expectedSizes {
				if len(mess) == expectedSize {
					// Verify the content is correct (all bytes should be the same)
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

		// Wait for socket file to be created
		for {
			stat, err := os.Stat(sktpath3)
			require.NoError(t, err)
			if stat.Mode()&os.ModeType == os.ModeSocket {
				break
			}
			time.Sleep(250 * time.Millisecond)
		}

		// Send each message separately (unixgram is datagram-based)
		wskt, err := net.DialUnix("unixgram", nil, &net.UnixAddr{Name: sktpath3, Net: "unixgram"})
		require.NoError(t, err)

		for i := 0; i < numMessages; i++ {
			_, err = wskt.Write(messages[i])
			require.NoError(t, err)
			// Small delay between messages
			time.Sleep(10 * time.Millisecond)
		}

		// Wait for all messages to be processed
		wg.Wait()

		mutex.Lock()
		assert.Equal(t, numMessages, receivedCount)
		mutex.Unlock()

		cancel()
		wskt.Close()
	})

	t.Run("test small buffer with message exceeding buffer size", func(t *testing.T) {
		sktpath4 := path.Join(tmpdir, "socket4")
		skt4, err := os.OpenFile(sktpath4, os.O_RDWR|os.O_CREATE, os.ModeSocket|os.ModePerm)
		require.NoError(t, err)
		defer skt4.Close()

		// Configure a small buffer size
		smallBufferTrans := Socket{
			conf: configT{
				Path:       sktpath4,
				BufferSize: 1500,
			},
			logger: &logWrapper{
				l: logger,
			},
		}

		// Create a message larger than the buffer size (2000 bytes)
		// For Unix datagram sockets, messages larger than buffer will be truncated by the OS
		msg := make([]byte, 2000)
		for i := 0; i < 2000; i++ {
			msg[i] = byte('X')
		}
		marker := []byte("--SHOULD-BE-TRUNCATED--")
		copy(msg[len(msg)-len(marker):], marker)

		// Setup message verification
		ctx, cancel := context.WithCancel(context.Background())
		wg := sync.WaitGroup{}
		go smallBufferTrans.Run(ctx, func(mess []byte) {
			wg.Add(1)
			// For Unix datagram, the message will be truncated to buffer size
			// The exact size depends on OS behavior, but it should be <= buffer size
			assert.Equal(t, true, len(mess) <= 1500)
			// The end marker should NOT be present since message was truncated
			endMarker := string(marker)
			messageStr := string(mess)
			if len(messageStr) >= len(endMarker) {
				assert.Equal(t, false, messageStr[len(messageStr)-len(endMarker):] == endMarker)
			}
			wg.Done()
		}, make(chan bool))

		// Wait for socket file to be created
		for {
			stat, err := os.Stat(sktpath4)
			require.NoError(t, err)
			if stat.Mode()&os.ModeType == os.ModeSocket {
				break
			}
			time.Sleep(250 * time.Millisecond)
		}

		// Send the large message
		wskt, err := net.DialUnix("unixgram", nil, &net.UnixAddr{Name: sktpath4, Net: "unixgram"})
		require.NoError(t, err)
		_, err = wskt.Write(msg)
		require.NoError(t, err)

		// Wait for message processing
		time.Sleep(100 * time.Millisecond)

		cancel()
		wg.Wait()
		wskt.Close()
	})

	t.Run("test large message transport", func(t *testing.T) {
		msg := make([]byte, regularBuffSize)
		for i := 0; i < regularBuffSize; i++ {
			msg[i] = byte('X')
		}
		msg[regularBuffSize-1] = byte('$')
		msg = append(msg, []byte(addition)...)

		// verify transport
		ctx, cancel := context.WithCancel(context.Background())
		wg := sync.WaitGroup{}
		go trans.Run(ctx, func(mess []byte) {
			wg.Add(1)
			strmsg := string(mess)
			assert.Equal(t, regularBuffSize+len(addition), len(strmsg))   // we received whole message
			assert.Equal(t, addition, strmsg[len(strmsg)-len(addition):]) // and the out-of-band part is correct
			wg.Done()
		}, make(chan bool))

		// wait for socket file to be created
		for {
			stat, err := os.Stat(sktpath)
			require.NoError(t, err)
			if stat.Mode()&os.ModeType == os.ModeSocket {
				break
			}
			time.Sleep(250 * time.Millisecond)
		}

		// write to socket
		wskt, err := net.DialUnix("unixgram", nil, &net.UnixAddr{Name: sktpath, Net: "unixgram"})
		require.NoError(t, err)
		_, err = wskt.Write(msg)
		require.NoError(t, err)

		cancel()
		wg.Wait()
		wskt.Close()
	})
}

func TestUdpSocketTransport(t *testing.T) {
	tmpdir, err := os.MkdirTemp(".", "socket_test_tmp")
	require.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	logpath := path.Join(tmpdir, "test.log")
	logger, err := logging.NewLogger(logging.DEBUG, logpath)
	require.NoError(t, err)

	trans := Socket{
		conf: configT{
			Socketaddr: "127.0.0.1:8642",
			Type:       "udp",
		},
		logger: &logWrapper{
			l: logger,
		},
	}

	t.Run("test small buffer with message within buffer size", func(t *testing.T) {
		// Configure a small buffer size
		smallBufferTrans := Socket{
			conf: configT{
				Socketaddr: "127.0.0.1:8645",
				Type:       "udp",
				BufferSize: 1500,
			},
			logger: &logWrapper{
				l: logger,
			},
		}

		// Create a message smaller than the buffer size (1200 bytes)
		msg := make([]byte, 1200)
		for i := 0; i < 1200; i++ {
			msg[i] = byte('U')
		}
		marker := []byte("--UDP-END--")
		copy(msg[len(msg)-len(marker):], marker)

		// Setup message verification
		ctx, cancel := context.WithCancel(context.Background())
		wg := sync.WaitGroup{}
		go smallBufferTrans.Run(ctx, func(mess []byte) {
			wg.Add(1)
			// Verify we received the complete message
			assert.Equal(t, 1200, len(mess))
			// Verify the end marker is present
			endMarkerPos := len(mess) - len(marker)
			assert.Equal(t, string(marker), string(mess[endMarkerPos:]))
			wg.Done()
		}, make(chan bool))

		// Wait for socket to be ready
		time.Sleep(100 * time.Millisecond)

		// Send the message
		addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:8645")
		require.NoError(t, err)
		wskt, err := net.DialUDP("udp", nil, addr)
		require.NoError(t, err)
		_, err = wskt.Write(msg)
		require.NoError(t, err)

		// Wait for message processing
		time.Sleep(100 * time.Millisecond)

		cancel()
		wg.Wait()
		wskt.Close()
	})

	t.Run("test small buffer with multiple messages", func(t *testing.T) {
		// Configure a small buffer size
		smallBufferTrans := Socket{
			conf: configT{
				Socketaddr: "127.0.0.1:8646",
				Type:       "udp",
				BufferSize: 1500,
			},
			logger: &logWrapper{
				l: logger,
			},
		}

		// Create multiple messages that fit within buffer
		numMessages := 3
		expectedSizes := []int{800, 1200, 900}
		messages := make([][]byte, numMessages)

		for i := 0; i < numMessages; i++ {
			messages[i] = make([]byte, expectedSizes[i])
			// Fill with a pattern unique to each message
			fillByte := byte('0' + i)
			for j := 0; j < expectedSizes[i]; j++ {
				messages[i][j] = fillByte
			}
		}

		// Setup message verification
		ctx, cancel := context.WithCancel(context.Background())
		receivedCount := 0
		var mutex sync.Mutex
		wg := sync.WaitGroup{}
		wg.Add(numMessages)

		go smallBufferTrans.Run(ctx, func(mess []byte) {
			mutex.Lock()
			defer mutex.Unlock()

			// Verify message size matches one of our expected sizes
			found := false
			for i, expectedSize := range expectedSizes {
				if len(mess) == expectedSize {
					// Verify the content is correct (all bytes should be the same)
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

		// Send each message separately (UDP is datagram-based)
		addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:8646")
		require.NoError(t, err)
		wskt, err := net.DialUDP("udp", nil, addr)
		require.NoError(t, err)

		for i := 0; i < numMessages; i++ {
			_, err = wskt.Write(messages[i])
			require.NoError(t, err)
			// Small delay between messages
			time.Sleep(10 * time.Millisecond)
		}

		// Wait for all messages to be processed
		wg.Wait()

		mutex.Lock()
		assert.Equal(t, numMessages, receivedCount)
		mutex.Unlock()

		cancel()
		wskt.Close()
	})

	t.Run("test small buffer with message exceeding buffer size", func(t *testing.T) {
		// Configure a small buffer size
		smallBufferTrans := Socket{
			conf: configT{
				Socketaddr: "127.0.0.1:8647",
				Type:       "udp",
				BufferSize: 1500,
			},
			logger: &logWrapper{
				l: logger,
			},
		}

		// Create a message larger than the buffer size (2000 bytes)
		// For UDP, messages larger than buffer will be truncated by the OS
		msg := make([]byte, 2000)
		for i := 0; i < 2000; i++ {
			msg[i] = byte('X')
		}
		marker := []byte("--SHOULD-BE-TRUNCATED--")
		copy(msg[len(msg)-len(marker):], marker)

		// Setup message verification
		ctx, cancel := context.WithCancel(context.Background())
		wg := sync.WaitGroup{}
		go smallBufferTrans.Run(ctx, func(mess []byte) {
			wg.Add(1)
			// For UDP, the message will be truncated to buffer size
			// The exact size depends on OS behavior, but it should be <= buffer size
			assert.Equal(t, true, len(mess) <= 1500)
			// The end marker should NOT be present since message was truncated
			endMarker := string(marker)
			messageStr := string(mess)
			assert.Equal(t, false, messageStr[len(messageStr)-len(endMarker):] == endMarker)
			wg.Done()
		}, make(chan bool))

		// Wait for socket to be ready
		time.Sleep(100 * time.Millisecond)

		// Send the large message
		addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:8647")
		require.NoError(t, err)
		wskt, err := net.DialUDP("udp", nil, addr)
		require.NoError(t, err)
		_, err = wskt.Write(msg)
		require.NoError(t, err)

		// Wait for message processing
		time.Sleep(100 * time.Millisecond)

		cancel()
		wg.Wait()
		wskt.Close()
	})

	t.Run("test large message transport", func(t *testing.T) {
		msg := make([]byte, regularBuffSize)
		for i := 0; i < regularBuffSize; i++ {
			msg[i] = byte('X')
		}
		msg[regularBuffSize-1] = byte('$')
		msg = append(msg, []byte(addition)...)

		// verify transport
		ctx, cancel := context.WithCancel(context.Background())
		wg := sync.WaitGroup{}
		go trans.Run(ctx, func(mess []byte) {
			wg.Add(1)
			strmsg := string(mess)
			assert.Equal(t, regularBuffSize+len(addition), len(strmsg))   // we received whole message
			assert.Equal(t, addition, strmsg[len(strmsg)-len(addition):]) // and the out-of-band part is correct
			wg.Done()
		}, make(chan bool))

		// write to socket
		addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:8642")
		require.NoError(t, err)
		wskt, err := net.DialUDP("udp", nil, addr)
		require.NoError(t, err)
		_, err = wskt.Write(msg)
		require.NoError(t, err)

		cancel()
		wg.Wait()
		wskt.Close()
	})
}

func TestTcpSocketTransport(t *testing.T) {
	tmpdir, err := os.MkdirTemp(".", "socket_test_tmp")
	require.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	logpath := path.Join(tmpdir, "test.log")
	logger, err := logging.NewLogger(logging.DEBUG, logpath)
	require.NoError(t, err)

	trans := Socket{
		conf: configT{
			Socketaddr: "127.0.0.1:8642",
			Type:       "tcp",
		},
		logger: &logWrapper{
			l: logger,
		},
	}

	t.Run("test small buffer with large message", func(t *testing.T) {
		// Configure a small buffer size to test the large message handling
		smallBufferTrans := Socket{
			conf: configT{
				Socketaddr: "127.0.0.1:8643",
				Type:       "tcp",
				BufferSize: 1500, // Small buffer to force multi-read for large messages
			},
			logger: &logWrapper{
				l: logger,
			},
		}

		// Create a message larger than the buffer size (3000 bytes of content)
		msgContent := make([]byte, 3000)
		for i := 0; i < 3000; i++ {
			msgContent[i] = byte('A' + (i % 26)) // Fill with A-Z pattern
		}
		// Add a marker at the end to verify we received the complete message
		marker := []byte("--END-MARKER--")
		msgContent = append(msgContent[:len(msgContent)-len(marker)], marker...)

		// Prepend the message length header for TCP protocol
		msgLength := new(bytes.Buffer)
		err := binary.Write(msgLength, binary.LittleEndian, uint64(len(msgContent)))
		require.NoError(t, err)
		fullMsg := append(msgLength.Bytes(), msgContent...)

		// Setup message verification
		ctx, cancel := context.WithCancel(context.Background())
		wg := sync.WaitGroup{}
		go smallBufferTrans.Run(ctx, func(mess []byte) {
			wg.Add(1)
			// Verify we received the complete message
			assert.Equal(t, 3000, len(mess))
			// Verify the end marker is present
			endMarkerPos := len(mess) - len(marker)
			assert.Equal(t, string(marker), string(mess[endMarkerPos:]))
			wg.Done()
		}, make(chan bool))

		// Connect and send the large message
		var wskt net.Conn
		wskt, err = net.Dial("tcp", "127.0.0.1:8643")
		if err != nil {
			// The socket might not be listening yet, wait and retry
			for retries := 0; err != nil && retries < 3; retries++ {
				time.Sleep(2 * time.Second)
				wskt, err = net.Dial("tcp", "127.0.0.1:8643")
			}
		}
		require.NoError(t, err)

		_, err = wskt.Write(fullMsg)
		require.NoError(t, err)

		// Wait for message to be processed
		time.Sleep(500 * time.Millisecond)

		cancel()
		wg.Wait()
		wskt.Close()
	})

	t.Run("test small buffer with multiple messages", func(t *testing.T) {
		// Configure a small buffer size
		smallBufferTrans := Socket{
			conf: configT{
				Socketaddr: "127.0.0.1:8644",
				Type:       "tcp",
				BufferSize: 1500,
			},
			logger: &logWrapper{
				l: logger,
			},
		}

		// Create multiple messages of varying sizes
		numMessages := 5
		messages := make([][]byte, numMessages)
		expectedSizes := []int{800, 1200, 2500, 900, 3500}

		var combinedMsg bytes.Buffer
		for i := 0; i < numMessages; i++ {
			msgContent := make([]byte, expectedSizes[i])
			// Fill with a pattern unique to each message
			fillByte := byte('0' + i)
			for j := 0; j < expectedSizes[i]; j++ {
				msgContent[j] = fillByte
			}
			messages[i] = msgContent

			// Write length header
			msgLength := new(bytes.Buffer)
			err := binary.Write(msgLength, binary.LittleEndian, uint64(len(msgContent)))
			require.NoError(t, err)
			combinedMsg.Write(msgLength.Bytes())
			combinedMsg.Write(msgContent)
		}

		// Setup message verification
		ctx, cancel := context.WithCancel(context.Background())
		receivedCount := 0
		var mutex sync.Mutex
		wg := sync.WaitGroup{}
		wg.Add(numMessages)

		go smallBufferTrans.Run(ctx, func(mess []byte) {
			mutex.Lock()
			defer mutex.Unlock()

			// Verify message size matches one of our expected sizes
			found := false
			for i, expectedSize := range expectedSizes {
				if len(mess) == expectedSize {
					// Verify the content is correct (all bytes should be the same)
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

		// Connect and send all messages in one write
		var wskt net.Conn
		wskt, err = net.Dial("tcp", "127.0.0.1:8644")
		if err != nil {
			for retries := 0; err != nil && retries < 3; retries++ {
				time.Sleep(2 * time.Second)
				wskt, err = net.Dial("tcp", "127.0.0.1:8644")
			}
		}
		require.NoError(t, err)

		_, err = wskt.Write(combinedMsg.Bytes())
		require.NoError(t, err)

		// Wait for all messages to be processed
		wg.Wait()

		mutex.Lock()
		assert.Equal(t, numMessages, receivedCount)
		mutex.Unlock()

		cancel()
		wskt.Close()
	})

	t.Run("test large message transport single connection", func(t *testing.T) {
		msg := make([]byte, regularBuffSize)
		for i := 0; i < regularBuffSize; i++ {
			msg[i] = byte('X')
		}
		msg[regularBuffSize-1] = byte('$')
		msg = append(msg, []byte(addition)...)
		msgLength := new(bytes.Buffer)
		err := binary.Write(msgLength, binary.LittleEndian, uint64(len(msg)))
		require.NoError(t, err)
		msg = append(msgLength.Bytes(), msg...)

		// verify transport
		ctx, cancel := context.WithCancel(context.Background())
		wg := sync.WaitGroup{}
		go trans.Run(ctx, func(mess []byte) {
			wg.Add(1)
			strmsg := string(mess)
			assert.Equal(t, regularBuffSize+len(addition), len(strmsg))   // we received whole message
			assert.Equal(t, addition, strmsg[len(strmsg)-len(addition):]) // and the out-of-band part is correct
			wg.Done()
		}, make(chan bool))

		// write to socket
		wskt, err := net.Dial("tcp", "127.0.0.1:8642")
		if err != nil {
			// The socket might not be listening yet, wait a little bit and try to connect again
			for retries := 0; err != nil && retries < 3; retries++ {
				time.Sleep(2 * time.Second)
				wskt, err = net.Dial("tcp", "127.0.0.1:8642")
			}
		}
		require.NoError(t, err)
		_, err = wskt.Write(msg)
		require.NoError(t, err)

		cancel()
		wg.Wait()
		wskt.Close()
	})

	t.Run("test large message transport multiple connections", func(t *testing.T) {
		msg := make([]byte, regularBuffSize)
		for i := 0; i < regularBuffSize; i++ {
			msg[i] = byte('X')
		}
		msg[regularBuffSize-1] = byte('$')
		msg = append(msg, []byte(addition)...)
		msgLength := new(bytes.Buffer)
		err := binary.Write(msgLength, binary.LittleEndian, uint64(len(msg)))
		require.NoError(t, err)
		msg = append(msgLength.Bytes(), msg...)

		// verify transport
		ctx, cancel := context.WithCancel(context.Background())
		wg := sync.WaitGroup{}
		go trans.Run(ctx, func(mess []byte) {
			wg.Add(1)
			strmsg := string(mess)
			assert.Equal(t, regularBuffSize+len(addition), len(strmsg))   // we received whole message
			assert.Equal(t, addition, strmsg[len(strmsg)-len(addition):]) // and the out-of-band part is correct
			wg.Done()
		}, make(chan bool))

		// write to socket
		wskt1, err := net.Dial("tcp", "127.0.0.1:8642")
		if err != nil {
			// The socket might not be listening yet, wait a little bit and try to connect again
			for retries := 0; err != nil && retries < 3; retries++ {
				time.Sleep(2 * time.Second)
				wskt1, err = net.Dial("tcp", "127.0.0.1:8642")
			}
		}
		require.NoError(t, err)

		// We shouldn't need to retry the second connection, if this fails, then something is wrong
		wskt2, err := net.Dial("tcp", "127.0.0.1:8642")
		require.NoError(t, err)

		_, err = wskt1.Write(msg)
		require.NoError(t, err)
		_, err = wskt2.Write(msg)
		require.NoError(t, err)

		cancel()
		wg.Wait()
		wskt1.Close()
		wskt2.Close()
	})
}
