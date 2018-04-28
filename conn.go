package smux

import (
	"io"
	"net"
	"sync"
)

type Conn struct {
	net.Conn
	sync.Mutex
	streams sync.Map
	counter *Counter
	ch      chan Stream
}

func (c *Conn) Listen() {
	for {
		// Read header
		buf := make([]byte, 8)
		_, err := c.Conn.Read(buf)
		if err == io.EOF {
			break
		}
		header := NewFrameHeader(buf)

		if header.length > 0 {
			// Read payload
			payload := make([]byte, header.length)
			_, err = c.Conn.Read(payload)
			if err == io.EOF {
				break
			}

			// Write payload to stream
			if _, ok := c.streams.Load(header.streamId); !ok {
				stream := make(chan []byte, 10)
				c.streams.Store(header.streamId, stream)
				c.ch <- NewStream(header.streamId, stream, c)
			}
			v, _ := c.streams.Load(header.streamId)
			stream := v.(chan []byte)
			select {
			case stream <- payload:
			default:
				// TODO: recover
			}
		}
		if header.flag == 1 {
			if v, ok := c.streams.Load(header.streamId); ok {
				stream := v.(chan []byte)
				close(stream)
				c.streams.Delete(header.streamId)
			}
		}
	}
}

func (c *Conn) Accept() (Stream, error) {
	return <-c.ch, nil
}

func (c *Conn) Stream() (Stream, error) {
	stream := make(chan []byte, 10)
	id := c.counter.Get()
	c.streams.Store(id, stream)
	return NewStream(id, stream, c), nil
}
