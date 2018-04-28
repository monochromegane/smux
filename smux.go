package smux

import (
	"io"
	"net"
	"sync"
)

type Listener struct {
	net.Listener
}

func Listen(network, address string) (*Listener, error) {
	listener, err := net.Listen(network, address)
	if err != nil {
		return nil, err
	}
	return &Listener{listener}, nil
}

type Conn struct {
	net.Conn
	sync.Mutex
	streams sync.Map
	counter *Counter
	ch      chan Stream
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

type Stream struct {
	conn   net.Conn
	id     uint32
	writer *io.PipeWriter
	reader *io.PipeReader
	in     chan []byte
}

func NewStream(id uint32, in chan []byte, c *Conn) Stream {
	pr, pw := io.Pipe()
	return Stream{
		id:     id,
		in:     in,
		conn:   c,
		reader: pr,
		writer: pw,
	}
}

func (s Stream) Poll() {
	for payload := range s.in {
		s.writer.Write(payload)
	}
	s.writer.Close()
}

func (s Stream) Read(b []byte) (int, error) {
	return s.reader.Read(b)
}

func (s Stream) Write(b []byte) (int, error) {
	frames := NewFrame(s.id, b, false)
	sum := 0
	for i, _ := range frames {
		n, err := s.conn.Write(frames[i])
		if err != nil {
			return 0, err
		}
		sum += n
	}
	return sum, nil
}

func (s Stream) Close() error {
	_, err := s.conn.Write(NewEndStreamFrame(s.id))
	return err
}

func Dial(network, address string) (*Conn, error) {
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	return &Conn{
		Conn:    conn,
		streams: sync.Map{},
		ch:      make(chan Stream, 1),
		counter: NewCounter(1),
	}, nil
}
