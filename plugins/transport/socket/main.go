package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/infrawatch/apputils/logging"
	"github.com/infrawatch/sg-core/pkg/config"
	"github.com/infrawatch/sg-core/pkg/data"
	"github.com/infrawatch/sg-core/pkg/transport"
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
	Path         string `validate:"required"`
	DumpMessages struct {
		Enabled bool
		Path    string
	} `yaml:"dumpMessages"` // only use for debug as this is very slow
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

	var laddr net.UnixAddr
	laddr.Name = s.conf.Path
	laddr.Net = "unixgram"

	os.Remove(s.conf.Path)
	pc, err := net.ListenUnixgram("unixgram", &laddr)
	if err != nil {
		s.logger.Errorf(err, "failed to bind unix socket %s", laddr.Name)
		return
	}
	// create socket file if it does not exist
	skt, err := pc.File()
	if err != nil {
		s.logger.Errorf(err, "failed to retrieve file handle for %s", laddr.Name)
		return
	}
	skt.Close()

	s.logger.Infof("socket %s bind successful", laddr.Name)
	go func() {
		for {
			var buff bytes.Buffer
			n, err := io.Copy(&buff, pc)
			if err != nil {
				s.logger.Errorf(err, "failed reading message")
			} else {
				s.logger.Debugf("read message of size %d bytes", n)
			}
			msg := buff.Bytes()

			if s.conf.DumpMessages.Enabled {
				_, err := s.dumpBuf.Write(msg)
				if err != nil {
					s.logger.Errorf(err, "writing to dump buffer")
				}
				_, err = s.dumpBuf.WriteString("\n")
				if err != nil {
					s.logger.Errorf(err, "writing to dump buffer")
				}
				s.dumpBuf.Flush()
			}

			w(msg)
			buff.Reset()
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
		DumpMessages: struct {
			Enabled bool
			Path    string
		}{
			Path: "/dev/stdout",
		},
	}

	err := config.ParseConfig(bytes.NewReader(c), &s.conf)
	if err != nil {
		return err
	}

	if s.conf.DumpMessages.Enabled {
		s.dumpFile, err = os.OpenFile(s.conf.DumpMessages.Path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			return err
		}

		s.dumpBuf = bufio.NewWriter(s.dumpFile)
	}

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
