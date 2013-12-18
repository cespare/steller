package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/cespare/gomaxprocs"
)

var (
	confFile = flag.String("conf", "conf.json", "JSON configuration file")
)

func init() { gomaxprocs.SetToNumCPU() }

func fatal(args ...interface{}) {
	fmt.Println(args...)
	os.Exit(1)
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

func constructRequests(userRequests []*Request) (*WeightedRequests, error) {
	requests := []WeightedRequest{}
	for _, userRequest := range userRequests {
		u, err := url.Parse(userRequest.URL)
		if err != nil {
			return nil, fmt.Errorf("Bad url '%s': %s", userRequest.URL, err)
		}
		// The returned request needs to be copied and have its Body swapped out for a duplicate whenever the
		// request is going to be used.
		header := make(http.Header)
		for k, v := range userRequest.Headers {
			header.Set(k, v)
		}
		req := WeightedRequest{
			Weight: *userRequest.Weight,
			Request: &http.Request{
				Method:        userRequest.Method,
				URL:           u,
				Body:          newBody([]byte(userRequest.Body)),
				Header:        header,
				ContentLength: int64(len(userRequest.Body)),
			},
		}
		requests = append(requests, req)
	}
	return NewWeightedRequests(requests), nil
}

// Result is the result of doing a single request.
type Result struct {
	Failed        bool // Could not round-trip the request at all
	StatusCode    int
	LatencyMillis float64
}

func runSingle(transport *http.Transport, request *http.Request) *Result {
	start := time.Now()
	resp, err := transport.RoundTrip(request)
	elapsed := time.Since(start)
	if err != nil {
		return &Result{Failed: true}
	}
	io.Copy(ioutil.Discard, resp.Body) // This is necessary to keep the TCP connection alive.
	resp.Body.Close()
	return &Result{
		StatusCode:    resp.StatusCode,
		LatencyMillis: float64(elapsed.Seconds() * 1000),
	}
}

type TestParams struct {
	Transport     *http.Transport
	Requests      *WeightedRequests
	TargetQPS     TargetQPS
	Duration      time.Duration
	MaxConcurrent int
}

func runRequests(params *TestParams) *ResultStats {
	resultStats := NewResultStats()
	wg := &sync.WaitGroup{}
	var ticks chan struct{}
	if params.TargetQPS.unlimited {
		ticks = make(chan struct{})
		done := make(chan struct{})
		go func() {
			for {
				select {
				case ticks <- struct{}{}:
				case <-done:
					return
				}
			}
		}()
		defer func() { done <- struct{}{} }()
	} else {
		ticker := NewPTicker(float64(params.TargetQPS.qps))
		defer ticker.Stop()
		ticks = ticker.C
	}
	timer := time.NewTimer(params.Duration)

	results := make(chan *Result)
	requestCh := make(chan *http.Request)
	cancel := make(chan bool)
	for j := 0; j < params.MaxConcurrent; j++ {
		go func() {
			// Each of the goroutines spawns a partner goroutine that actually makes the request. This way the
			// primary goroutine of the pair can cancel outstanding requests.
			requests := make(chan *http.Request)
			defer func() { close(requests) }()
			go func() {
				for r := range requests {
					results <- runSingle(params.Transport, r)
					wg.Done()
				}
			}()
			var currentRequest *http.Request
			for r := range requestCh {
				req := *r
				req.Body = req.Body.(*Body).dup()
				select {
				case requests <- &req:
					currentRequest = &req
				case <-cancel:
					if currentRequest != nil {
						// Cancel the outstanding request
						params.Transport.CancelRequest(currentRequest)
						// Not going to send the current request
						wg.Done()
						return
					}
				}
			}
		}()
	}

	fmt.Println("Starting test...")
	start := time.Now()
	for {
		select {
		case <-ticks:
			// Send, if a goroutine is ready.
			wg.Add(1)
			select {
			case requestCh <- params.Requests.Random():
			default:
				wg.Done()
			}
		case result := <-results:
			resultStats.Insert(result)
		case <-timer.C:
			fmt.Println("Test finished. Cleaning up...")
			resultStats.Duration = time.Since(start)
			done := make(chan bool)
			go func() {
				wg.Wait()
				done <- true
			}()
			close(requestCh)
			for {
				select {
				case <-results: // Drain responses
				case cancel <- true: // Cancel outstanding requests
				case <-done:
					return resultStats
				}
			}
		}
	}
}

func main() {
	conf, err := parseConfig()
	if err != nil {
		fatal(fmt.Sprintf("Config error: %s", err))
	}
	if conf.ReportingStats != nil {
		if conf.ReportingStats.Buckets != nil {
			buckets = conf.ReportingStats.Buckets
		}
		if conf.ReportingStats.Quantiles != nil {
			quants = conf.ReportingStats.Quantiles
		}
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
	results := runRequests(params)

	fmt.Println()
	// Sanity check first
	if results.Succeeded == 0 {
		if results.Failed == 0 {
			fmt.Println("ERROR: no requests made.")
		} else {
			fmt.Printf("ERROR: all requests (%d) failed. Is the target server accepting requests?\n",
				results.Failed)
		}
		return
	}
	fmt.Println(results)
}
