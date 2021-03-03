package main

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/infrawatch/apputils/logging"
	"github.com/infrawatch/sg-core/pkg/config"
	"github.com/infrawatch/sg-core/pkg/data"
	"github.com/infrawatch/sg-core/pkg/transport"
)

const maxBufferSize = 16384

var (
	msgCount int64
	lastVal  int64
)

func rate() int64 {
	rate := msgCount - lastVal
	lastVal = msgCount
	return rate
}

type configT struct {
	Path string `validate:"required"`
}
type logWrapper struct {
	l *logging.Logger
}

func (lw *logWrapper) Errorf(err error, format string, a ...interface{}) {
	lw.l.Metadata(logging.Metadata{"plugin": "socket", "error": err})
	lw.l.Error(fmt.Sprintf(format, a...))
}

func (lw *logWrapper) Infof(format string, a ...interface{}) {
	lw.l.Metadata(logging.Metadata{"plugin": "socket"})
	lw.l.Info(fmt.Sprintf(format, a...))
}

func (lw *logWrapper) Debugf(format string, a ...interface{}) {
	lw.l.Metadata(logging.Metadata{"plugin": "socket"})
	lw.l.Debug(fmt.Sprintf(format, a...))
}

func (lw *logWrapper) Warnf(format string, a ...interface{}) {
	lw.l.Metadata(logging.Metadata{"plugin": "socket"})
	lw.l.Warn(fmt.Sprintf(format, a...))
}

//Socket basic struct
type Socket struct {
	conf   configT
	logger *logWrapper
}

//Run implements type Transport
func (s *Socket) Run(ctx context.Context, w transport.WriteFn, done chan bool) {

	msgBuffer := make([]byte, maxBufferSize)
	var laddr net.UnixAddr

	laddr.Name = s.conf.Path
	laddr.Net = "unixgram"

	os.Remove(s.conf.Path)

	pc, err := net.ListenUnixgram("unixgram", &laddr)
	if err != nil {
		s.logger.Errorf(err, "failed to listen on unix socket %s", laddr.Name)
		return
	}

	s.logger.Infof("socket listening on %s", laddr.Name)
	go func() {
		for {
			n, err := pc.Read(msgBuffer)
			//fmt.Printf("received message: %s\n", string(msgBuffer))

			if err != nil || n < 1 {
				if err != nil {
					s.logger.Errorf(err, "reading from socket failed")
				}
				done <- true
				return
			}
			w(msgBuffer[:n])
			msgCount++
		}
	}()

	for {
		select {
		case <-ctx.Done():
			goto Done
		default:
			time.Sleep(time.Second)
			s.logger.Debugf("receiving %d msg/s", rate())
		}
	}
Done:
	pc.Close()
	os.Remove(s.conf.Path)
	s.logger.Infof("exited")
}

//Listen ...
func (s *Socket) Listen(e data.Event) {
	fmt.Printf("Received event: %v\n", e)
}

//Config load configurations
func (s *Socket) Config(c []byte) error {
	s.conf = configT{}
	err := config.ParseConfig(bytes.NewReader(c), &s.conf)
	if err != nil {
		return err
	}
	return nil
}

//New create new socket transport
func New(l *logging.Logger) transport.Transport {
	return &Socket{
		logger: &logWrapper{
			l: l,
		},
	}
}
