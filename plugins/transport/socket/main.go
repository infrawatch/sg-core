package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

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
	Path         string
	Type         string
	Url          string
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

func (s *Socket) initUnixSocket() *net.UnixConn {
	var laddr net.UnixAddr
	laddr.Name = s.conf.Path
	laddr.Net = "unixgram"

	os.Remove(s.conf.Path)
	pc, err := net.ListenUnixgram("unixgram", &laddr)
	if err != nil {
		s.logger.Errorf(err, "failed to bind unix socket %s", laddr.Name)
		return nil
	}
	// create socket file if it does not exist
	skt, err := pc.File()
	if err != nil {
		s.logger.Errorf(err, "failed to retrieve file handle for %s", laddr.Name)
		return nil
	}
	skt.Close()

	s.logger.Infof("socket listening on %s", laddr.Name)

	return pc
}

func (s *Socket) initUdpSocket() *net.UDPConn {
	addr, err := net.ResolveUDPAddr("udp", s.conf.Url)
	pc, err := net.ListenUDP("udp", addr)
	if err != nil {
		s.logger.Errorf(err, "failed to bind unix socket to url: %s", s.conf.Url)
		return nil
	}

	s.logger.Infof("socket listening on %s", s.conf.Url)

	return pc
}

// Run implements type Transport
func (s *Socket) Run(ctx context.Context, w transport.WriteFn, done chan bool) {
	var pc net.Conn
	if s.conf.Type == "unix" {
		pc = s.initUnixSocket()
	} else if s.conf.Type == "udp" {
		pc = s.initUdpSocket()
	} else {
		s.logger.Errorf(nil, "Unknown socket type")
		return
	}
	if pc == nil {
		s.logger.Errorf(nil, "Failed to initialize socket transport plugin")
	}
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

			if s.conf.DumpMessages.Enabled {
				_, err := s.dumpBuf.Write(msgBuffer[:n])
				if err != nil {
					s.logger.Errorf(err, "writing to dump buffer")
				}
				_, err = s.dumpBuf.WriteString("\n")
				if err != nil {
					s.logger.Errorf(err, "writing to dump buffer")
				}
				s.dumpBuf.Flush()
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
	if s.conf.Type == "unix" {
		os.Remove(s.conf.Path)
	}
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
		Type: "unix",
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

	s.conf.Type = strings.ToLower(s.conf.Type)
	if s.conf.Type != "unix" && s.conf.Type != "udp" {
		return fmt.Errorf("Unable to determine socket type from configuration file. Should be either \"unix\" or \"udp\", received: %s",
			s.conf.Type)
	}

	if s.conf.Type == "unix" && s.conf.Path == "" {
		return fmt.Errorf("The path configuration option is required when using unix socket type")
	}

	if s.conf.Type == "udp" && s.conf.Url == "" {
		return fmt.Errorf("The url configuration option is required when using udp socket type")
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
