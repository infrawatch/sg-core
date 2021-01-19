package main

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"

	"github.com/infrawatch/apputils/logging"
	"github.com/infrawatch/sg-core/pkg/config"
	"github.com/infrawatch/sg-core/pkg/data"
	"github.com/infrawatch/sg-core/pkg/transport"
)

const maxBufferSize = 4096

type configT struct {
	Path string `validate:"required"`
}

//Socket basic struct
type Socket struct {
	conf   configT
	logger *logging.Logger
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
		s.logger.Metadata(logging.Metadata{"plugin": "socket", "error": err})
		s.logger.Error("failed to listen on unix soc")
		return
	}

	s.logger.Metadata(logging.Metadata{"plugin": "socket"})
	s.logger.Info(fmt.Sprintf("socket listening on %s", laddr.Name))
	go func() {
		for {
			n, err := pc.Read(msgBuffer)
			//fmt.Printf("received message: %s\n", string(msgBuffer))

			if err != nil || n < 1 {
				done <- true
				return
			}
			w(msgBuffer[:n])
		}
	}()

	<-ctx.Done()
	pc.Close()
	os.Remove(s.conf.Path)
	s.logger.Metadata(logging.Metadata{"plugin": "socket"})
	s.logger.Info("exited")
}

//Listen ...
func (s *Socket) Listen(e data.Event) {
	fmt.Printf("Recieved event: %v\n", e)
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
		logger: l,
	}
}
