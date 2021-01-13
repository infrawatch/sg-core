package transport

import (
	"context"
	"strings"
	"sync"

	"github.com/infrawatch/sg-core-refactor/pkg/data"
)

// package transport defines the interfaces for interacting with transport
// plugins

//Mode indicates if transport is setup to receive or write
type Mode int

const (
	//WRITE ...
	WRITE = iota
	//READ ...
	READ
)

//String get string representation of mode
func (m Mode) String() string {
	return [...]string{"WRITE", "READ"}[m]
}

//FromString get mode from string
func (m Mode) FromString(s string) {
	m = map[string]Mode{
		"write": WRITE,
		"read":  READ,
	}[strings.ToLower(s)]
}

//WriteFn func type for writing from transport to handlers
type WriteFn func([]byte)

//Transport type listens on one interface and delivers data to core
//TODO: give transports a writer to send logs to
type Transport interface {
	Config([]byte) error
	Run(context.Context, *sync.WaitGroup, WriteFn, chan bool)
	Listen(data.Event)
}
