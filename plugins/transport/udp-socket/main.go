package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"io"
	"time"
	"strings"
	"sync"
	"encoding/binary"

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
func setBuffer(c *net.TCPConn) {
}

// Run implements type Transport
func (s *Socket) Run(ctx context.Context, w transport.WriteFn, done chan bool) {

	var m sync.Mutex
	addr, err := net.ResolveTCPAddr("tcp", s.conf.Uri)
	if err != nil {
		fmt.Println(err)
	}
	pcc, err := net.ListenTCP("tcp", addr)
	if err != nil {
		fmt.Println("smth")
	}
	go func(pcc *net.TCPListener) {
		for {
			pc, err := pcc.AcceptTCP()
			if err != nil {
				fmt.Println(err)
			}
			if err != nil {
				s.logger.Errorf(err, "failed to bind unix socket %s", s.conf.Uri)
				return
			}

			s.logger.Infof("socket listening on %s", s.conf.Uri)
			go func(maxBuffSize int64, pc *net.TCPConn) {
				remaining := []byte{}
				msgBuffer := make([]byte, maxBuffSize)
				for {
					if err != nil {
						fmt.Println(err)
					}
					n, err := pc.Read(msgBuffer)
					remained := len(remaining)
					msgBuffer = append(remaining, msgBuffer...)
					n += remained
					if err != nil {
						if err != nil {
							s.logger.Errorf(err, "reading from socket failed")
						}
						//done <- true
						pc.Close()
						return
					}

					// whole buffer was used, so we are potentially handling larger message
					if int64(n) == maxBuffSize {
						s.logger.Warnf("full read buffer used")
					}
					var pos int64
					pos = 0
					var length int64
					buffer := bytes.NewReader(msgBuffer[:n])
					for pos < int64(n) {
						buffer.Seek(pos, io.SeekStart)
						err := binary.Read(buffer, binary.LittleEndian, &length)
						if err != nil || pos + 8 + length > int64(n) || length < 0 {
							if length > 1000 || length < 0 {
							fmt.Println("Can't read message")
							fmt.Println(err)
							fmt.Printf("prev: %d\n", remained)
							fmt.Printf("n: %d\n", n)
							fmt.Printf("pos: %d\n", pos)
							fmt.Printf("len: %d\n", length)
							fmt.Printf("remaining = %d : %d\n", pos, n)
								fmt.Println(string(msgBuffer[:n]))
							}
							break
						}
						m.Lock()
						w(msgBuffer[pos + 8 : pos + 8 + length])
						msgCount++
						m.Unlock()
						pos += 8 + length
					}
					remaining = []byte{}
					remaining = append(remaining, msgBuffer[pos:n]...)
				}
			}(maxBufferSize, pc)
		}
	}(pcc)

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
	//pc.Close()
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
