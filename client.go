package smux

import (
	"bytes"
	"io"
	"sync"
)

type Client struct {
	sync.Mutex

	Network string
	Address string
	conn    *Conn
}

func (c *Client) Post(b []byte) ([]byte, error) {
	stream, err := c.getStream()
	if err != nil {
		return nil, err
	}

	_, err = stream.WriteOnce(b)
	if err != nil {
		return nil, err
	}

	go stream.Poll()

	var buf bytes.Buffer
	out := make([]byte, 1024)
	for {
		n, err := stream.Read(out)
		if err == io.EOF {
			break
		}
		buf.Write(out[:n])
	}
	return buf.Bytes(), nil
}

func (c *Client) getStream() (Stream, error) {
	c.Lock()
	defer c.Unlock()

	conn, err := c.getConn(false)
	if err != nil {
		return Stream{}, err
	}

	stream, err := conn.Stream()
	if err == ExceedError {
		conn, err := c.getConn(true)
		if err != nil {
			return Stream{}, err
		}
		return conn.Stream()
	} else {
		return stream, err
	}

}

func (c *Client) getConn(force bool) (*Conn, error) {
	if c.conn == nil || force {
		conn, err := Dial(c.Network, c.Address)
		if err != nil {
			return nil, err
		}
		go conn.Listen()
		c.conn = &conn
	}
	return c.conn, nil
}
