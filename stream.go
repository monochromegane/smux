package smux

import (
	"io"
	"net"
)

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
	return s.write(b, false)
}

func (s Stream) WriteOnce(b []byte) (int, error) {
	return s.write(b, true)
}

func (s Stream) write(b []byte, seal bool) (int, error) {
	frames := packing(s.id, b, seal)
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
	_, err := s.conn.Write(sealing(s.id))
	return err
}
