package smux

import (
	"sync"
	"testing"
)

func TestCounterGet(t *testing.T) {
	counter := NewCounter(START_STREAM_ID_OF_CLIENT)

	loop := 50
	ids := make(chan uint32, loop)
	var wg sync.WaitGroup
	wg.Add(loop)
	for i := 0; i < loop; i++ {
		go func() {
			defer wg.Done()
			stream_id, _ := counter.Get()
			ids <- stream_id
		}()
	}
	wg.Wait()
	close(ids)

	results := make(map[uint32]struct{})
	for id := range ids {
		results[id] = struct{}{}
	}

	for i := START_STREAM_ID_OF_CLIENT; i < loop*2; i += 2 {
		if _, ok := results[uint32(i)]; !ok {
			t.Errorf("Counter.Get should return stream id, but not return %d", i)
		}
	}
}
