package smux

import (
	"net"
	"sync"
)

type Listener struct {
	net.Listener
}

func (l Listener) Accept() (Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return Conn{}, err
	}
	return Conn{
		Conn:    conn,
		streams: sync.Map{},
		ch:      make(chan Stream, 1),
		counter: NewCounter(START_STREAM_ID_OF_SERVER),
	}, nil
}
