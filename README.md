# smux [![Build Status](https://travis-ci.org/monochromegane/smux.svg?branch=master)](https://travis-ci.org/monochromegane/smux)

smux is a socket multiplexer.
smux multiplexes one connection with a virtual channel called a stream.
It behaves like a very simple HTTP/2 binary framing layer, but it reduces protocol overhead.

smux sends and receives multiple requests and responses in parallel using a single connection. Therefore, our application will be fast and simple.

# Usage

smux provides simple server and client.

```go
// smux server
server := smux.Server{
	Network: "tcp", // or "unix"
	Address: "localhost:3000", // or "sockfile"
        Handler: smux.HandlerFunc(func(w io.Writer, r io.Reader) {
                io.Copy(ioutil.Discard, r)
		fmt.Fprint(w, "Hello, smux client!")
        }),
}

server.ListenAndServe()
```

```go
// smux client
client := smux.Client{
	Network: "tcp", // or "unix"
	Address: "localhost:3000", // or "sockfile"
}

body, _ := client.Post([]byte("Hello, smux server!"))
fmt.Printf("%s\n", body) // "Hello, smux client!"
```

And smux provides raw level interface (stream.Write and Read). You can learn from client and server code.

## Performance

Benchmark for HTTP, smux, and similar implementation.

![benchmark](https://user-images.githubusercontent.com/1845486/39610904-7695e7da-4f8e-11e8-989c-5a2cfac3a4f9.png)

Benchmark script is [here](https://github.com/monochromegane/smux/blob/master/cmd/bench).
It runs on MacBook Pro (15-inch, 2017), CPU 2.8 GHz Intel Core i7, memory 16 GB. Go version is go1.10.2 darwin/amd64.

Currently, xtaci/smux (ssmux) is fast. I am currently speeding up my smux !

## License

[MIT](https://github.com/monochromegane/smux/blob/master/LICENSE)

## Author

[monochromegane](https://github.com/monochromegane)

