package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	amqp "github.com/Azure/go-amqp"

	"github.com/infrawatch/apputils/logging"
	"github.com/infrawatch/sg-core/pkg/config"
	"github.com/infrawatch/sg-core/pkg/data"
	"github.com/infrawatch/sg-core/pkg/transport"
)

var (
	appname  = "amqp1"
	msgCount int64
	lastVal  int64
)

func rate() int64 {
	rate := msgCount - lastVal
	lastVal = msgCount
	return rate
}

type configT struct {
	URI          string `validate:"required"`
	Channel      string `validate:"required"`
	LinkCredit   uint32 `yaml:"linkCredit"`
	DumpMessages struct {
		Enabled bool
		Path    string
	} `yaml:"dumpMessages"` // only use for debug as this is very slow
}

// Amqp1 basic struct
type Amqp1 struct {
	conn     *amqp.Client
	sess     *amqp.Session
	receiver *amqp.Receiver
	conf     configT
	logger   *logging.Logger
	dumpBuf  *bufio.Writer
	dumpFile *os.File
}

func sendMessage(msg interface{}, w transport.WriteFn, logger *logging.Logger) {
	if tmsg, ok := msg.(string); ok {
		w([]byte(tmsg))
		msgCount++
	} else {
		logger.Metadata(logging.Metadata{"plugin": appname, "type": fmt.Sprintf("%T", msg)})
		logger.Error("unknown type of received message")
	}
}

// Run implements type Transport
func (at *Amqp1) Run(ctx context.Context, w transport.WriteFn, done chan bool) {
	var err error
	// connect
	at.conn, err = amqp.Dial(at.conf.URI)
	if err != nil {
		at.logger.Metadata(logging.Metadata{"plugin": appname, "error": err})
		at.logger.Error("failed to connect")
		return
	}
	defer at.conn.Close()

	// open session
	at.sess, err = at.conn.NewSession()
	if err != nil {
		at.logger.Metadata(logging.Metadata{"plugin": appname, "error": err})
		at.logger.Error("failed to create session")
		return
	}

	// create receiver
	at.receiver, err = at.sess.NewReceiver(
		amqp.LinkSourceAddress(at.conf.Channel),
		amqp.LinkCredit(at.conf.LinkCredit),
	)
	if err != nil {
		at.logger.Metadata(logging.Metadata{"plugin": appname, "error": err})
		at.logger.Error("failed to create receiver")
		return
	}
	defer func(rcv *amqp.Receiver) {
		ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
		rcv.Close(ctx)
		cancel()
	}(at.receiver)

	at.logger.Metadata(logging.Metadata{
		"plugin":     appname,
		"connection": fmt.Sprintf("%s/%s", at.conf.URI, at.receiver.Address()),
	})
	at.logger.Info("listening")

	for {
		at.logger.Debug(fmt.Sprintf("receiving %d msg/s", rate()))
		err := at.receiver.HandleMessage(ctx, func(msg *amqp.Message) error {
			// accept message
			msg.Accept(context.Background())
			// dump message
			if at.conf.DumpMessages.Enabled {
				_, errr := at.dumpBuf.Write(msg.GetData())
				if errr != nil {
					return errr
				}
				_, errr = at.dumpBuf.WriteString("\n")
				if errr != nil {
					return errr
				}
				at.dumpBuf.Flush()
			}
			// send message
			switch val := msg.Value.(type) {
			case []interface{}:
				for _, itm := range val {
					sendMessage(itm, w, at.logger)
				}
			case interface{}:
				sendMessage(val, w, at.logger)
			default:
				at.logger.Metadata(logging.Metadata{"plugin": appname, "type": val})
				at.logger.Error("unknown message format")
			}
			return nil
		})

		if err != nil && !strings.Contains(err.Error(), "context canceled") {
			at.logger.Metadata(logging.Metadata{"plugin": appname, "error": err})
			at.logger.Error("failed to handle message")
			break
		}
	}

	at.dumpFile.Close()
	at.logger.Metadata(logging.Metadata{"plugin": appname})
	at.logger.Info("exited")
}

// Listen ...
func (at *Amqp1) Listen(e data.Event) {
	at.logger.Metadata(logging.Metadata{"plugin": appname, "event": e})
	at.logger.Debug("received event")
}

// Config load configurations
func (at *Amqp1) Config(c []byte) error {
	at.conf = configT{
		DumpMessages: struct {
			Enabled bool
			Path    string
		}{
			false,
			"",
		},
		URI:        "amqp://127.0.0.1:5672",
		Channel:    "rsyslog/logs",
		LinkCredit: 1024,
	}

	err := config.ParseConfig(bytes.NewReader(c), &at.conf)
	if err != nil {
		return err
	}

	if at.conf.DumpMessages.Enabled {
		at.dumpFile, err = os.OpenFile(at.conf.DumpMessages.Path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			return err
		}

		at.dumpBuf = bufio.NewWriter(at.dumpFile)
	}

	return nil
}

// New create new amqp1 transport
func New(l *logging.Logger) transport.Transport {
	return &Amqp1{
		logger: l,
	}
}
