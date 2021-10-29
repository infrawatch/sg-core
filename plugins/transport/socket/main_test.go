package main

import (
	"context"
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

func TestSocketTransport(t *testing.T) {
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
		msg := make([]byte, initBufferSize)
		addition := "wubba lubba dub dub"
		for i := 0; i < initBufferSize; i++ {
			msg[i] = byte('X')
		}
		msg[initBufferSize-1] = byte('$')
		msg = append(msg, []byte(addition)...)

		// verify transport
		ctx, cancel := context.WithCancel(context.Background())
		wg := sync.WaitGroup{}
		go trans.Run(ctx, func(mess []byte) {
			wg.Add(1)
			strmsg := string(mess)
			assert.Equal(t, initBufferSize+len(addition), len(strmsg))    // we received whole message
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
