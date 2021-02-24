package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/infrawatch/apputils/logging"
	"github.com/infrawatch/sg-core/pkg/config"
	"github.com/infrawatch/sg-core/pkg/data"
	"github.com/infrawatch/sg-core/pkg/transport"
)

const (
	appname = "dummy-alertmanager"
)

type configT struct {
	Port   int
	Output string
}

//DummyAM listens on given port and prints all HTTP requests
type DummyAM struct {
	conf   configT
	logger *logging.Logger
}

//Run implements type Transport
func (dam *DummyAM) Run(ctx context.Context, w transport.WriteFn, done chan bool) {
	// print all received requests
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {

		dam.logger.Debug("received HTTP request")
		out, err := os.OpenFile(dam.conf.Output, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			dam.logger.Metadata(logging.Metadata{"plugin": "dummy-alertmanager", "error": err})
			dam.logger.Error("failed to open output file")
		} else {
			defer out.Close()
		}
		msg, err := ioutil.ReadAll(req.Body)
		if err != nil {
			dam.logger.Metadata(logging.Metadata{"plugin": "dummy-alertmanager", "error": err})
			dam.logger.Error("failed to read request")
		} else {
			out.WriteString(fmt.Sprintf("%s\n", msg))
		}

	})

	srv := &http.Server{Addr: fmt.Sprintf(":%d", dam.conf.Port)}
	go func(server *http.Server, ctx context.Context) {
		<-ctx.Done()
		if err := srv.Shutdown(ctx); err != nil {
			dam.logger.Metadata(logging.Metadata{"plugin": appname, "error": err})
			dam.logger.Error("failed to shut down HTTP server")
		} else {
			dam.logger.Info("shutting down HTTP server")
		}
	}(srv, ctx)

	err := srv.ListenAndServe()
	dam.logger.Metadata(logging.Metadata{"plugin": appname, "error": err})
	dam.logger.Info("exited")
}

//Listen ...
func (dam *DummyAM) Listen(e data.Event) {
}

//Config load configurations
func (dam *DummyAM) Config(c []byte) error {
	err := config.ParseConfig(bytes.NewReader(c), &dam.conf)
	if err != nil {
		return err
	}
	return nil
}

//New create new socket transport
func New(l *logging.Logger) transport.Transport {
	return &DummyAM{
		conf: configT{
			Port:   16661,
			Output: "/dev/stdout",
		},
		logger: l,
	}
}
