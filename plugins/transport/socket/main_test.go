package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"io/ioutil"
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
	tmpdir, err := ioutil.TempDir(".", "socket_test_tmp")
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
	tmpdir, err := ioutil.TempDir(".", "socket_test_tmp")
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
	tmpdir, err := ioutil.TempDir(".", "socket_test_tmp")
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
