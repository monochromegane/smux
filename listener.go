package smux

import (
	"net"
	"sync"
)

type Listener struct {
	net.Listener
}

func (l Listener) Accept() (*Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return &Conn{
		Conn:    conn,
		streams: sync.Map{},
		ch:      make(chan Stream, 1),
		counter: NewCounter(2),
	}, nil
}
