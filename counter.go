package smux

import (
	"fmt"
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
	if c.current+2 > MAX_STREAM_ID {
		return 0, nil
	}
	c.current += 2
	return c.current, nil
}

type ExceedError struct{}

func (e ExceedError) Error() string {
	return fmt.Sprintf("Exceeded max stream id: %d", MAX_STREAM_ID)
}
