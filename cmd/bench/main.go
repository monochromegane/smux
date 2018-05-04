package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/hashicorp/yamux"
	"github.com/monochromegane/smux"
	ssmux "github.com/xtaci/smux"
	"golang.org/x/net/http2"
)

var (
	port          int
	host          string
	mode          string
	proto         string
	cert          string
	key           string
	delay         int
	numJobs       int
	numConcurrent int
)

func init() {
	flag.IntVar(&port, "port", 3000, "number of port")
	flag.StringVar(&host, "host", "localhost", "hostname")
	flag.StringVar(&mode, "mode", "server", "server|client")
	flag.StringVar(&proto, "proto", "smux", "http|http2|smux|yamux|ssmux")
	flag.StringVar(&cert, "cert", "server.crt", "cert file")
	flag.StringVar(&key, "key", "server.key", "key file")
	flag.IntVar(&delay, "delay", 10, "Handler running time (Millisecond)")
	flag.IntVar(&numJobs, "jobs", 10000, "number of jobs")
	flag.IntVar(&numConcurrent, "concurrent", 100, "number of concurrent")
	flag.Parse()
}

func main() {
	if mode == "server" {
		server := newServer()
		server.Run()
	} else {
		client := newClient()

		errCnt := 0
		var wg sync.WaitGroup
		ch := make(chan struct{})
		for i := 0; i < numConcurrent; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for _ = range ch {
					err := client.Post()
					if err != nil {
						errCnt++
					}
				}
			}()
		}

		start := time.Now()
		for i := 0; i < numJobs; i++ {
			ch <- struct{}{}
		}
		close(ch)
		wg.Wait()

		elapsed := time.Since(start)
		fmt.Printf("%s,%d,%d,%0.2f,%0.2f,%d,%0.3f,%d\n", proto, numJobs, numConcurrent, float64(elapsed)/float64(time.Second), float64(numJobs)/(float64(elapsed)/float64(time.Second)), errCnt, float64(errCnt)/float64(numJobs), delay)
	}

}

type Server interface {
	Run()
}

type Client interface {
	Post() error
}

func newServer() Server {
	switch proto {
	case "http": // HTTP/1.1
		return newHttpServer()
	case "http2":
		return newHttp2Server()
	case "smux":
		return newSmuxServer()
	case "yamux":
		return newYamuxServer()
	case "ssmux":
		return newSsmuxServer()
	default:
		return newHttpServer()
	}
}

func newClient() Client {
	switch proto {
	case "http": // HTTP/1.1
		http.DefaultTransport.(*http.Transport).MaxIdleConns = numConcurrent
		http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = numConcurrent
		return newHttpClient()
	case "http2":
		http.DefaultTransport = &http2.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
		return newHttp2Client()
	case "smux":
		return newSmuxClient()
	case "yamux":
		return newYamuxClient()
	case "ssmux":
		return newSsmuxClient()
	default:
		return newHttpClient()
	}
}

// HTTP
func newHttpServer() Server {
	m := http.NewServeMux()
	m.Handle("/", http.HandlerFunc(httpHandler(responseData())))
	s := http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: m,
	}
	return HttpServer{s}
}

type HttpServer struct {
	server http.Server
}

func (s HttpServer) Run() {
	s.server.ListenAndServe()
}

func newHttpClient() Client {
	return HttpClient{
		requestData: requestData(),
		url:         fmt.Sprintf("http://%s:%d", host, port),
	}
}

type HttpClient struct {
	requestData []byte
	url         string
}

func (c HttpClient) Post() error {
	resp, err := http.Post(c.url, "application/json", bytes.NewBuffer(c.requestData))
	if err != nil {
		return err
	}
	io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()

	return nil
}

// HTTP/2
func newHttp2Server() Server {
	s := newHttpServer()
	return Http2Server{
		server:   s.(HttpServer).server,
		certFile: cert,
		keyFile:  key,
	}
}

type Http2Server struct {
	server   http.Server
	certFile string
	keyFile  string
}

func (s Http2Server) Run() {
	s.server.ListenAndServeTLS(s.certFile, s.keyFile)
}

func newHttp2Client() Client {
	return HttpClient{
		requestData: requestData(),
		url:         fmt.Sprintf("https://%s:%d", host, port),
	}
}

// SMUX
func newSmuxServer() Server {
	s := smux.Server{
		Network: "tcp",
		Address: fmt.Sprintf("%s:%d", host, port),
		Handler: smux.HandlerFunc(smuxHandler(responseData())),
	}
	return SmuxServer{s}
}

type SmuxServer struct {
	server smux.Server
}

func (s SmuxServer) Run() {
	s.server.ListenAndServe()
}

type SmuxClient struct {
	client      smux.Client
	requestData []byte
}

func newSmuxClient() Client {
	return &SmuxClient{
		requestData: requestData(),
		client:      smux.Client{Network: "tcp", Address: fmt.Sprintf("%s:%d", host, port)},
	}
}

func (c *SmuxClient) Post() error {
	body, err := c.client.Post(c.requestData)
	if err != nil {
		return err
	}
	io.Copy(ioutil.Discard, bytes.NewReader(body))
	return nil
}

// Yamux
type YamuxServer struct {
	responseData []byte
	listener     net.Listener
	handler      smux.Handler
}

func (s YamuxServer) Run() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			break
		}
		go func() {
			session, err := yamux.Server(conn, nil)
			if err != nil {
				panic(err)
			}
			var wg sync.WaitGroup
			for {
				stream, err := session.Accept()
				if err != nil {
					break
				}
				wg.Add(1)
				go func() {
					defer wg.Done()
					defer stream.Close()

					var buf bytes.Buffer
					out := make([]byte, 512)
					read := 0
					for {
						n, err := stream.Read(out)
						if err == io.EOF {
							break
						}
						buf.Write(out[:n])
						read += n
						if read == 2727 { // Workaround...
							break
						}
					}

					var b bytes.Buffer
					w := bufio.NewWriter(&b)
					s.handler.Serve(w, bytes.NewReader(buf.Bytes()))
					w.Flush()
					stream.Write(b.Bytes())
				}()
			}
			wg.Wait()
		}()
	}

}

func newYamuxServer() Server {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		panic(err)
	}
	return YamuxServer{
		listener:     listener,
		responseData: responseData(),
		handler:      smux.HandlerFunc(smuxHandler(responseData())),
	}
}

type YamuxClient struct {
	session     *yamux.Session
	requestData []byte
}

func (c YamuxClient) Post() error {
	stream, err := c.session.Open()
	if err != nil {
		return err
	}
	stream.Write(c.requestData)
	// stream.Close()

	var buf bytes.Buffer
	out := make([]byte, 1024)
	for {
		n, err := stream.Read(out)
		if err == io.EOF {
			break
		}
		buf.Write(out[:n])
	}
	io.Copy(ioutil.Discard, bytes.NewReader(buf.Bytes()))
	return nil
}

func newYamuxClient() Client {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		panic(err)
	}

	session, err := yamux.Client(conn, nil)
	if err != nil {
		panic(err)
	}
	return YamuxClient{
		session:     session,
		requestData: requestData(),
	}
}

// xtaci/smux
type SsmuxServer struct {
	responseData []byte
	listener     net.Listener
	handler      smux.Handler
}

func (s SsmuxServer) Run() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			break
		}
		go func() {
			session, err := ssmux.Server(conn, nil)
			if err != nil {
				panic(err)
			}
			var wg sync.WaitGroup
			for {
				stream, err := session.AcceptStream()
				if err != nil {
					break
				}
				wg.Add(1)
				go func() {
					defer wg.Done()
					defer stream.Close()

					var buf bytes.Buffer
					out := make([]byte, 512)
					read := 0
					for {
						n, err := stream.Read(out)
						if err == io.EOF {
							break
						}
						buf.Write(out[:n])
						read += n
						if read == 2727 { // Workaround...
							break
						}
					}

					var b bytes.Buffer
					w := bufio.NewWriter(&b)
					s.handler.Serve(w, bytes.NewReader(buf.Bytes()))
					w.Flush()
					stream.Write(b.Bytes())
				}()
			}
			wg.Wait()
		}()
	}

}

func newSsmuxServer() Server {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		panic(err)
	}
	return SsmuxServer{
		listener:     listener,
		responseData: responseData(),
		handler:      smux.HandlerFunc(smuxHandler(responseData())),
	}
}

type SsmuxClient struct {
	session     *ssmux.Session
	requestData []byte
}

func (c SsmuxClient) Post() error {
	stream, err := c.session.OpenStream()
	if err != nil {
		return err
	}
	stream.Write(c.requestData)

	var buf bytes.Buffer
	out := make([]byte, 1024)
	for {
		n, err := stream.Read(out)
		if err == io.EOF {
			break
		}

		buf.Write(out[:n])
	}
	io.Copy(ioutil.Discard, bytes.NewReader(buf.Bytes()))
	return nil
}

func newSsmuxClient() Client {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		panic(err)
	}

	session, err := ssmux.Client(conn, nil)
	if err != nil {
		panic(err)
	}
	return SsmuxClient{
		session:     session,
		requestData: requestData(),
	}
}

type Request struct {
	Query []float32 `json:"query"`
}

type Response struct {
	Ids []int `josn:"ids"`
}

func smuxHandler(data []byte) func(io.Writer, io.Reader) {
	return func(w io.Writer, r io.Reader) {
		body, _ := ioutil.ReadAll(r)
		var req Request
		json.Unmarshal(body, &req)
		time.Sleep(time.Duration(delay) * time.Millisecond)
		fmt.Fprint(w, string(data))
	}
}

func httpHandler(data []byte) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		body, _ := ioutil.ReadAll(r.Body)
		var req Request
		json.Unmarshal(body, &req)
		time.Sleep(time.Duration(delay) * time.Millisecond)
		fmt.Fprint(w, string(data))
	}
}

func requestData() []byte {
	q := make([]float32, 256)
	for i, _ := range q {
		q[i] = rand.Float32()
	}
	req, _ := json.Marshal(Request{Query: q})
	return req
}

func responseData() []byte {
	ids := make([]int, 10)
	for i, _ := range ids {
		ids[i] = rand.Int()
	}
	res, _ := json.Marshal(Response{Ids: ids})
	return res
}
