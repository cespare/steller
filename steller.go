package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

var (
	confFile = flag.String("conf", "conf.json", "JSON configuration file")
)

func fatal(args ...interface{}) {
	fmt.Println(args...)
	os.Exit(1)
}

// A Request is a user-configured request (which can be used to make an http.Request).
type Request struct {
	Method  string
	URL     string
	Body    string
	Headers map[string]string
}

type Conf struct {
	Requests        []*Request
	TargetQPS       int `json:"target_qps"`
	DurationSeconds int `json:"duration_seconds"`
	MaxConcurrent   int `json:"max_concurrent"`
}

// A Body is a ReadCloser with a static []byte message inside.
type Body struct {
	buf []byte
	off int
}

func newBody(b []byte) *Body { return &Body{b, 0} }

func (b *Body) dup() *Body { return &Body{b.buf, 0} }

func (b *Body) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	n := copy(p, b.buf[b.off:])
	if n == 0 {
		return 0, io.EOF
	}
	b.off += n
	return n, nil
}

func (b *Body) Close() error {
	b.off = len(b.buf)
	return nil
}

func constructRequests(userRequests []*Request) ([]*http.Request, error) {
	requests := []*http.Request{}
	for _, userRequest := range userRequests {
		u, err := url.Parse(userRequest.URL)
		if err != nil {
			return nil, fmt.Errorf("Bad url '%s': %s", userRequest.URL, err)
		}
		// The returned request needs to be copied and have its Body swapped out for a duplicate whenever the
		// request is going to be used.
		req := &http.Request{
			Method:        userRequest.Method,
			URL:           u,
			Body:          newBody([]byte(userRequest.Body)),
			Header:        make(http.Header), // http.Transport doesn't like a nil header
			ContentLength: int64(len(userRequest.Body)),
		}
		requests = append(requests, req)
	}
	return requests, nil
}

// Result is the result of doing a single request.
type Result struct {
	LatencyMillis float64
}

func runSingle(transport *http.Transport, request *http.Request) Result {
	start := time.Now()
	resp, err := transport.RoundTrip(request)
	elapsed := time.Since(start)
	if err != nil {
		panic(err) // TODO: handle
	}
	resp.Body.Close()
	return Result{float64(elapsed.Seconds() * 1000)}
}

type TestParams struct {
	Transport     *http.Transport
	Requests      []*http.Request
	TargetQPS     int
	Duration      time.Duration
	MaxConcurrent int
}

func runRequests(params *TestParams) *Stats {
	stats := NewStats()
	wg := &sync.WaitGroup{}
	ticker := NewPTicker(float64(params.TargetQPS))
	defer ticker.Stop()
	timer := time.NewTimer(params.Duration)
	i := 0 // Current request

	results := make(chan Result)
	requestCh := make(chan *http.Request)
	for j := 0; j < params.MaxConcurrent; j++ {
		go func() {
			for r := range requestCh {
				req := *r
				req.Body = req.Body.(*Body).dup()
				results <- runSingle(params.Transport, &req)
				wg.Done()
			}
		}()
	}

	fmt.Println("Starting test...")
	start := time.Now()
	for {
		select {
		case <-ticker.C:
			// Send, if a goroutine is ready.
			wg.Add(1)
			select {
			case requestCh <- params.Requests[i]:
				i = (i + 1) % len(params.Requests)
			default:
				wg.Done()
			}
		case result := <-results:
			stats.Insert(result.LatencyMillis)
		case <-timer.C:
			fmt.Println("Test finished. Cleaning up...")
			stats.Duration = time.Since(start)
			done := make(chan bool)
			go func() {
				wg.Wait()
				done <- true
			}()
			for {
				select {
				case <-results: // Drain the requests
				case <-done:
					return stats
				}
			}
		}
	}
}

func parseConfig() (*Conf, error) {
	flag.Parse()
	conf := &Conf{}
	f, err := os.Open(*confFile)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(f)
	if err := decoder.Decode(conf); err != nil {
		return nil, err
	}

	// Sanity checking
	if len(conf.Requests) == 0 {
		return nil, fmt.Errorf("No requests supplied")
	}
	if conf.MaxConcurrent == 0 {
		conf.MaxConcurrent = 100
	}
	return conf, nil
}

func main() {
	conf, err := parseConfig()
	if err != nil {
		fatal(fmt.Sprintf("Config error: %s", err))
	}

	requests, err := constructRequests(conf.Requests)
	if err != nil {
		fatal(fmt.Sprintf("Config error: %s", err))
	}

	transport := &http.Transport{
		// Don't automatically enable gzip compression
		DisableCompression:    true,
		MaxIdleConnsPerHost:   conf.MaxConcurrent, // TODO: separately tunable?
		ResponseHeaderTimeout: 10 * time.Second,   // TODO: tunable
	}

	params := &TestParams{
		Transport:     transport,
		Requests:      requests,
		TargetQPS:     conf.TargetQPS,
		Duration:      time.Duration(conf.DurationSeconds) * time.Second,
		MaxConcurrent: conf.MaxConcurrent,
	}
	stats := runRequests(params)
	fmt.Println(stats)
}
