package smux

import "sync"

type Counter struct {
	sync.Mutex
	current uint32
}

func NewCounter(init uint32) *Counter {
	return &Counter{
		current: init,
	}
}

func (c *Counter) Get() uint32 {
	c.Lock()
	defer c.Unlock()
	c.current += 2
	return c.current
}
