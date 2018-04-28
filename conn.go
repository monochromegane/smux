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
		buf := make([]byte, NUM_BYTES_HEADER)
		_, err := c.Conn.Read(buf)
		if err == io.EOF {
			break
		}
		length, _, flag, streamId := parseHeader(buf)

		if length > 0 {
			// Read payload
			payload := make([]byte, length)
			_, err = c.Conn.Read(payload)
			if err == io.EOF {
				break
			}

			// Write payload to stream
			if _, ok := c.streams.Load(streamId); !ok {
				stream := make(chan []byte, 10)
				c.streams.Store(streamId, stream)
				c.ch <- NewStream(streamId, stream, c)
			}
			v, _ := c.streams.Load(streamId)
			stream := v.(chan []byte)
			select {
			case stream <- payload:
			default:
				// TODO: recover
			}
		}
		if flag == FLAG_DATA_END_STREAM {
			if v, ok := c.streams.Load(streamId); ok {
				stream := v.(chan []byte)
				close(stream)
				c.streams.Delete(streamId)
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
