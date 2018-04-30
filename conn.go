package smux

import (
	"bytes"
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
listen:
	for {
		// Read header
		var buf bytes.Buffer
		read := 0
		for {
			header := make([]byte, NUM_BYTES_HEADER-read)
			n, err := c.Conn.Read(header)
			if err != nil || err == io.EOF {
				break listen
			}
			buf.Write(header[:n])
			read += n
			if read == NUM_BYTES_HEADER {
				break
			}
		}
		length, _, flag, streamId := parseHeader(buf.Bytes())

		if length > 0 {
			// Read payload
			var buf bytes.Buffer
			var read uint16
			for {
				payload := make([]byte, length-read)
				n, err := c.Conn.Read(payload)
				if err != nil || err == io.EOF {
					break listen
				}
				buf.Write(payload[:n])
				read += uint16(n)
				if read == length {
					break
				}
			}
			payload := buf.Bytes()

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
	id, err := c.counter.Get()
	if err != nil {
		return NewStream(id, stream, c), err
	}
	c.streams.Store(id, stream)
	return NewStream(id, stream, c), nil
}
