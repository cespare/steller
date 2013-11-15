package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sync"
	"text/template"
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
	QPS             int
	DurationSeconds int `json:"duration_seconds"`
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

type Report struct {
	Duration time.Duration
	Requests int64

	// Computed
	QPS float64
}

func (r *Report) PostProcess() {
	r.QPS = float64(r.Requests) / float64(r.Duration.Seconds())
}

var ReportTmpl = template.Must(template.New("report").Parse(
	`Test duration:            {{printf "%10.3fs" .Duration.Seconds}}
Successful requests:      {{printf "%10d" .Requests}}
Mean requests per second: {{printf "%10.3f" .QPS}}`))

func (r *Report) String() string {
	buf := &bytes.Buffer{}
	err := ReportTmpl.Execute(buf, r)
	if err != nil {
		panic(err)
	}
	return buf.String()
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

func runSingle(transport *http.Transport, request *http.Request, wg *sync.WaitGroup) {
	resp, err := transport.RoundTrip(request)
	if err != nil {
		panic(err) // TODO: handle
	}
	resp.Body.Close()
	wg.Done()
}

func runRequests(transport *http.Transport, requests []*http.Request, qps int, duration time.Duration) *Report {
	ticker := time.NewTicker(time.Second / time.Duration(qps))
	defer ticker.Stop()
	timer := time.NewTimer(duration)
	i := 0 // Current request
	report := &Report{}
	start := time.Now()
	done := make(chan bool)
	wg := &sync.WaitGroup{}
	for {
		select {
		case <-ticker.C:
			wg.Add(1)
			go func(r *http.Request) {
				req := *r
				req.Body = req.Body.(*Body).dup()
				runSingle(transport, &req, wg)
				done <- true
			}(requests[i])
			i = (i + 1) % len(requests)
		case <-done:
			report.Requests++
		case <-timer.C:
			report.Duration = time.Since(start)
			// Drain the requests
			wg.Wait()
			return report
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
		MaxIdleConnsPerHost:   10,               // TODO: tunable
		ResponseHeaderTimeout: 10 * time.Second, // TODO: tunable
	}

	report := runRequests(transport, requests, conf.QPS, time.Duration(conf.DurationSeconds)*time.Second)
	report.PostProcess()
	fmt.Println(report)
}
