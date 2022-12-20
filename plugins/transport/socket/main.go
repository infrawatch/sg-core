package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/infrawatch/apputils/logging"
	"github.com/infrawatch/sg-core/pkg/config"
	"github.com/infrawatch/sg-core/pkg/data"
	"github.com/infrawatch/sg-core/pkg/transport"
)

const (
	maxBufferSize = 65535
	udp           = "udp"
	unix          = "unix"
	tcp           = "tcp"
	msgLengthSize = 8
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
	Path         string `validate:"required_without=Socketaddr"`
	Type         string
	Socketaddr   string `validate:"required_without=Path"`
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
	mutex    sync.Mutex
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

func (s *Socket) initUDPSocket() *net.UDPConn {
	addr, err := net.ResolveUDPAddr(udp, s.conf.Socketaddr)
	if err != nil {
		s.logger.Errorf(err, "failed to resolve udp address: %s", s.conf.Socketaddr)
		return nil
	}
	pc, err := net.ListenUDP(udp, addr)
	if err != nil {
		s.logger.Errorf(err, "failed to bind udp socket to addr: %s", s.conf.Socketaddr)
		return nil
	}

	s.logger.Infof("socket listening on %s", s.conf.Socketaddr)

	return pc
}

func (s *Socket) initTCPSocket() *net.TCPListener {
	addr, err := net.ResolveTCPAddr(tcp, s.conf.Socketaddr)
	if err != nil {
		s.logger.Errorf(err, "failed to resolve tcp address: %s", s.conf.Socketaddr)
		return nil
	}
	pc, err := net.ListenTCP(tcp, addr)
	if err != nil {
		s.logger.Errorf(err, "failed to bind tcp socket to addr: %s", s.conf.Socketaddr)
		return nil
	}

	s.logger.Infof("socket listening on %s", s.conf.Socketaddr)

	return pc
}

func (s *Socket) WriteTCPMsg(w transport.WriteFn, msgBuffer []byte, n int) (int64, error) {
	var pos int64 = 0
	var length int64
	reader := bytes.NewReader(msgBuffer[:n])
	for pos+msgLengthSize < int64(n) {
		_, err := reader.Seek(pos, io.SeekStart)
		if err != nil {
			return pos, err
		}
		err = binary.Read(reader, binary.LittleEndian, &length)
		if err != nil {
			return pos, err
		}

		if pos+msgLengthSize+length > int64(n) ||
			pos+msgLengthSize+length < 0 {
			break
		}
		s.mutex.Lock()
		w(msgBuffer[pos+msgLengthSize : pos+msgLengthSize+length])
		msgCount++
		s.mutex.Unlock()
		pos += msgLengthSize + length
	}
	return pos, nil
}

func (s *Socket) ReceiveData(maxBuffSize int64, done chan bool, pc net.Conn, w transport.WriteFn) {
	defer pc.Close()
	msgBuffer := make([]byte, maxBuffSize)
	var remainingMsg []byte
	for {
		n, err := pc.Read(msgBuffer)
		if err != nil || n < 1 {
			if err != nil {
				s.logger.Errorf(err, "reading from socket failed")
			}
			if s.conf.Type != tcp {
				done <- true
			}
			return
		}
		msgBuffer = append(remainingMsg, msgBuffer...)

		// whole buffer was used, so we are potentially handling larger message
		if n == len(msgBuffer) {
			s.logger.Warnf("full read buffer used")
		}

		n += len(remainingMsg)

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

		if s.conf.Type == tcp {
			parsed, err := s.WriteTCPMsg(w, msgBuffer, n)
			if err != nil {
				s.logger.Errorf(err, "error, while parsing messages")
				return
			}
			remainingMsg = make([]byte, int64(n)-parsed)
			copy(remainingMsg, msgBuffer[parsed:n])
		} else {
			w(msgBuffer[:n])
			msgCount++
		}
	}
}

// Run implements type Transport
func (s *Socket) Run(ctx context.Context, w transport.WriteFn, done chan bool) {
	var pc net.Conn
	switch s.conf.Type {
	case udp:
		pc = s.initUDPSocket()
		if pc == (*net.UDPConn)(nil) {
			s.logger.Errorf(nil, "Failed to initialize socket transport plugin with type: "+s.conf.Type)
			return
		}
		go s.ReceiveData(maxBufferSize, done, pc, w)

	case tcp:
		TCPSocket := s.initTCPSocket()
		if TCPSocket == nil {
			s.logger.Errorf(nil, "Failed to initialize socket transport plugin with type: "+s.conf.Type)
			return
		}
		go func() {
			for {
				pc, err := TCPSocket.AcceptTCP()
				if err != nil {
					select {
					case <-ctx.Done():
						break
					default:
						s.logger.Errorf(err, "failed to accept TCP connection")
						continue
					}
				}
				go s.ReceiveData(maxBufferSize, done, pc, w)
			}
		}()
	case unix:
		fallthrough
	default:
		pc = s.initUnixSocket()
		if pc == (*net.UnixConn)(nil) {
			s.logger.Errorf(nil, "Failed to initialize socket transport plugin with type: "+s.conf.Type)
			return
		}
		go s.ReceiveData(maxBufferSize, done, pc, w)
	}

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
	if s.conf.Type == unix {
		os.Remove(s.conf.Path)
	}
	s.dumpFile.Close()
	s.logger.Infof("exited")
}

// Listen ...
func (s *Socket) Listen(e data.Event) {
	fmt.Printf("received event: %v\n", e)
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
		Type: unix,
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
	if s.conf.Type != unix && s.conf.Type != udp && s.conf.Type != tcp {
		return fmt.Errorf("unable to determine socket type from configuration file. Should be one of \"unix\", \"udp\" or \"tcp\", received: %s",
			s.conf.Type)
	}

	if s.conf.Type == unix && s.conf.Path == "" {
		return fmt.Errorf("the path configuration option is required when using unix socket type")
	}

	if (s.conf.Type == udp || s.conf.Type == tcp) && s.conf.Socketaddr == "" {
		return fmt.Errorf("the socketaddr configuration option is required when using udp or tcp socket type")
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
