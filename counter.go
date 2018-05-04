package smux

import (
	"errors"
	"sync"
)

type Counter struct {
	sync.Mutex
	current uint32
}

func NewCounter(init uint32) *Counter {
	return &Counter{
		current: init,
	}
}

func (c *Counter) Get() (uint32, error) {
	c.Lock()
	defer c.Unlock()
	current := c.current
	if c.current+2 > MAX_STREAM_ID {
		return 0, ExceedError
	}
	c.current += 2
	return current, nil
}

var ExceedError = errors.New("Exceeded max stream id")
