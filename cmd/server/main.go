package main

import (
	"fmt"
	"io"
	"os"

	"github.com/monochromegane/smux"
)

func main() {
	fmt.Printf("%s\n", "Start")
	os.Remove("sockfile")
	l, err := smux.Listen("unix", "sockfile")
	if err != nil {
		panic(err)
	}
	defer l.Close()

	for {
		fmt.Printf("%s\n", "connection accepting...")
		conn, err := l.Accept()
		if err != nil {
			panic(err)
		}
		defer conn.Close()

		fmt.Printf("%s\n", "listening...")
		go conn.Listen()

		go func() {
			for {
				fmt.Printf("%s\n", "stream accepting...")
				stream, err := conn.Accept()
				if err != nil {
					panic(err)
				}

				go func() {
					defer stream.Close()
					go stream.Poll()
					fmt.Printf("%s\n", "stream accept!")
					for i := 0; i < 30; i++ {
						buf := make([]byte, 3)
						fmt.Printf("%d: %s\n", i, "stream reading...!")
						n, err := stream.Read(buf)
						fmt.Printf("%d (%v) --> %v\n", n, err, buf[:n])
						if err == io.EOF {
							break
						}
					}
					n, err := stream.Write([]byte("1iyo12iyo23iyo3"))
					fmt.Printf("stream response... %d (%v)\n", n, err)
				}()
			}
		}()
	}
}
