package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"time"
	"strings"

	"github.com/infrawatch/apputils/logging"
	"github.com/infrawatch/sg-core/pkg/config"
	"github.com/infrawatch/sg-core/pkg/data"
	"github.com/infrawatch/sg-core/pkg/transport"
)

const (
	maxBufferSize = 65535
)

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
	Uri         string `validate:"required"`
	Host        string
	Port        string
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

// Socket basic struct
type Socket struct {
	conf     configT
	logger   *logWrapper
	dumpBuf  *bufio.Writer
	dumpFile *os.File
}

// Run implements type Transport
func (s *Socket) Run(ctx context.Context, w transport.WriteFn, done chan bool) {

	addr, err := net.ResolveUDPAddr("udp", s.conf.Uri)
	pc, err := net.ListenUDP("udp", addr)
	if err != nil {
		s.logger.Errorf(err, "failed to bind unix socket %s", s.conf.Uri)
		return
	}

	s.logger.Infof("socket listening on %s", s.conf.Uri)
	go func(maxBuffSize int64) {
		msgBuffer := make([]byte, maxBuffSize)
		for {
			n, err := pc.Read(msgBuffer)
			if err != nil || n < 1 {
				if err != nil {
					s.logger.Errorf(err, "reading from socket failed")
				}
				done <- true
				return
			}

			// whole buffer was used, so we are potentially handling larger message
			if n == len(msgBuffer) {
				s.logger.Warnf("full read buffer used")
			}

			w(msgBuffer[:n])
			msgCount++
		}
	}(maxBufferSize)

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
	s.dumpFile.Close()
	s.logger.Infof("exited")
}

// Listen ...
func (s *Socket) Listen(e data.Event) {
	fmt.Printf("Received event: %v\n", e)
}

// Config load configurations
func (s *Socket) Config(c []byte) error {
	s.conf = configT{
		Uri: "/dev/stdout",
		Host: "localhost",
		Port: "8642",
	}

	err := config.ParseConfig(bytes.NewReader(c), &s.conf)
	if err != nil {
		return err
	}
	str := strings.Split(s.conf.Uri, ":")
	s.conf.Host = str[0]
	s.conf.Port = str[1]

	return nil
}

// New create new socket transport
func New(l *logging.Logger) transport.Transport {
	return &Socket{
		logger: &logWrapper{
			l: l,
		},
	}
}
