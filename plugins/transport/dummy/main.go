package main

import (
	"context"
	"encoding/json"
	"math/rand"
	"sync"
	"time"

	"github.com/infrawatch/apputils/logging"
	"github.com/infrawatch/sg-core-refactor/pkg/data"
	"github.com/infrawatch/sg-core-refactor/pkg/transport"
)

const maxBufferSize = 4096

var msgBuffer []byte

var sent bool

func init() {
	msgBuffer = make([]byte, maxBufferSize)
}

type metric struct {
	Values         []float64 `json:"values"`
	Dstypes        []string  `json:"dstypes"`
	Dsnames        []string  `json:"dsnames"`
	Time           int64     `json:"time"`
	Interval       int       `json:"interval"`
	Host           string    `json:"host"`
	Plugin         string    `json:"plugin"`
	PluginInstance string    `json:"plugin_instance"`
	Type           string    `json:"type"`
	TypeInstance   string    `json:"type_instance"`
}

func generateMetric() []metric {
	if !sent {
		//sent = true
		return []metric{{
			Values: []float64{
				rand.Float64() * 1000,
				rand.Float64() * 10000,
			},
			Dstypes: []string{
				"derive",
				"counter",
			},
			Dsnames: []string{
				"rx",
				"tx",
			},
			Host:           "controller-0.redhat.local",
			Time:           time.Now().Unix(),
			Interval:       5,
			Plugin:         "virt",
			PluginInstance: "asdf",
			Type:           "if_packets",
			TypeInstance:   "tap73125d-60",
		},
			{
				Values: []float64{
					rand.Float64() * 1000,
					rand.Float64() * 10000,
				},
				Dstypes: []string{
					"derive",
					"counter",
				},
				Dsnames: []string{
					"in",
					"out",
				},
				Host:           "controller-0.redhat.local",
				Time:           time.Now().Unix(),
				Interval:       5,
				Plugin:         "virt",
				PluginInstance: "asdf",
				Type:           "if_packets",
				TypeInstance:   "tap73125d-60",
			},
		}
	}
	return []metric{}
}

//Dummy basic struct
type Dummy struct {
}

//Run implements type Transport
func (s *Dummy) Run(ctx context.Context, wg *sync.WaitGroup, w transport.WriteFn, done chan bool) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			goto done
		case <-time.After(time.Second * 1):
			time.Sleep(time.Second * 1)
			r, err := json.Marshal(generateMetric())
			if err != nil {
				panic(err)
			}
			w(r)
		}
	}

done:
}

//Listen ...
func (s *Dummy) Listen(e data.Event) {

}

//Config load configurations
func (s *Dummy) Config(c []byte) error {
	return nil
}

//New create new socket transport
func New(l *logging.Logger) transport.Transport {
	return &Dummy{}
}
