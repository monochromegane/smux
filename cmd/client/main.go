package main

import (
	"fmt"
	"io"
	"sync"

	"github.com/monochromegane/smux"
)

func main() {
	payload := []byte("1oge12oge23oge3")

	conn, err := smux.Dial("unix", "sockfile")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	fmt.Printf("%s\n", "listening...")
	go conn.Listen()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		fmt.Printf("%s\n", "stream accepting...")

		go func() {
			defer wg.Done()
			stream, err := conn.Stream()
			if err != nil {
				panic(err)
			}

			go stream.Poll()

			fmt.Printf("%s\n", "stream writing...")
			_, err = stream.Write(payload)
			if err != nil {
				panic(err)
			}
			stream.Close()

			for {
				out := make([]byte, 1024)
				n, err := stream.Read(out)
				if err == io.EOF {
					break
				}
				fmt.Printf("%d (%v) --> %v\n", n, err, out[:n])
				fmt.Printf("%s\n", out[:n])
			}
		}()
	}
	wg.Wait()
}
