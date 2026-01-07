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
	defaultBufferSize = 65535
	minBufferSize     = 1024
	maxBufferSize     = 10485760  // 10MB - max configurable buffer size
	maxMessageSize    = 104857600 // 100MB - max single message size for TCP
	udp               = "udp"
	unix              = "unix"
	tcp               = "tcp"
	msgLengthSize     = 8
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
	BufferSize   int64  `yaml:"bufferSize"` // read buffer size in bytes (default: 65535, min: 1024, max: 10485760)
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
	var pos int64
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

		// Combine remaining data from previous iteration with newly read data
		var data []byte
		if len(remainingMsg) > 0 {
			data = make([]byte, len(remainingMsg)+n)
			copy(data, remainingMsg)
			copy(data[len(remainingMsg):], msgBuffer[:n])
		} else {
			data = msgBuffer[:n]
		}
		totalSize := len(data)

		// whole buffer was used, so we are potentially handling larger message
		if n == int(maxBuffSize) {
			if s.conf.Type == tcp {
				s.logger.Debugf("full read buffer used (%d bytes read), will read more if needed", n)
			} else {
				// For UDP/Unix sockets, this means the message was truncated by the OS
				s.logger.Errorf(nil, "message truncated: buffer size (%d bytes) exceeded for %s socket - increase bufferSize configuration or message will be incomplete", maxBuffSize, s.conf.Type)
			}
		}

		// For TCP, check if we need to read more data for a large message
		if s.conf.Type == tcp && len(data) >= msgLengthSize {
			// Peek at the message length header
			reader := bytes.NewReader(data[:msgLengthSize])
			var msgLength int64
			err := binary.Read(reader, binary.LittleEndian, &msgLength)
			if err == nil && msgLength > 0 {
				// Validate message size
				if msgLength > maxMessageSize {
					s.logger.Errorf(nil, "rejecting TCP message: size (%d bytes) exceeds maximum allowed (%d bytes)", msgLength, maxMessageSize)
					return
				}
				requiredSize := msgLengthSize + int(msgLength)
				// If the message is larger than our current data, read more
				if requiredSize > len(data) {
					s.logger.Debugf("large TCP message detected, size: %d bytes, reading more data", msgLength)
					// Read additional data until we have the complete message
					additionalNeeded := requiredSize - len(data)
					for additionalNeeded > 0 {
						readSize := maxBuffSize
						if int64(additionalNeeded) < maxBuffSize {
							readSize = int64(additionalNeeded)
						}
						tempBuf := make([]byte, readSize)
						n, err := pc.Read(tempBuf)
						if err != nil || n < 1 {
							s.logger.Errorf(err, "failed to read continuation of large TCP message (expected %d more bytes)", additionalNeeded)
							return
						}
						data = append(data, tempBuf[:n]...)
						additionalNeeded = requiredSize - len(data)
					}
					totalSize = len(data)
					s.logger.Debugf("completed reading large TCP message, total size: %d bytes", totalSize)
				}
			}
		}

		if s.conf.DumpMessages.Enabled {
			_, err := s.dumpBuf.Write(data)
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
			parsed, err := s.WriteTCPMsg(w, data, totalSize)
			if err != nil {
				s.logger.Errorf(err, "error, while parsing messages")
				return
			}
			remainingMsg = make([]byte, int64(totalSize)-parsed)
			copy(remainingMsg, data[parsed:totalSize])
		} else {
			w(data)
			msgCount++
			remainingMsg = nil
		}
	}
}

// Run implements type Transport
func (s *Socket) Run(ctx context.Context, w transport.WriteFn, done chan bool) {
	// Use default buffer size if not configured (e.g., in tests)
	bufferSize := s.conf.BufferSize
	if bufferSize == 0 {
		bufferSize = defaultBufferSize
	}

	var pc net.Conn
	switch s.conf.Type {
	case udp:
		pc = s.initUDPSocket()
		if pc == (*net.UDPConn)(nil) {
			s.logger.Errorf(nil, "Failed to initialize socket transport plugin with type: "+s.conf.Type)
			return
		}
		go s.ReceiveData(bufferSize, done, pc, w)

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
				go s.ReceiveData(bufferSize, done, pc, w)
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
		go s.ReceiveData(bufferSize, done, pc, w)
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
		Type:       unix,
		BufferSize: defaultBufferSize,
	}

	err := config.ParseConfig(bytes.NewReader(c), &s.conf)
	if err != nil {
		return err
	}

	// Validate buffer size
	if s.conf.BufferSize < minBufferSize {
		return fmt.Errorf("bufferSize must be at least %d bytes, got %d", minBufferSize, s.conf.BufferSize)
	}
	if s.conf.BufferSize > maxBufferSize {
		return fmt.Errorf("bufferSize must be at most %d bytes, got %d", maxBufferSize, s.conf.BufferSize)
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
