package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
)

// A Request is a user-configured request (which can be used to make an http.Request).
type Request struct {
	Method   string
	URL      string
	Body     string
	BodyFile string `json:"body_file"`
	Headers  map[string]string
}

type Conf struct {
	Requests        []*Request
	RequestsFile    string `json:"requests_file"`
	TargetQPS       int    `json:"target_qps"`
	DurationSeconds int    `json:"duration_seconds"`
	MaxConcurrent   int    `json:"max_concurrent"`
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
	if conf.RequestsFile != "" {
		reqFile, err := os.Open(conf.RequestsFile)
		if err != nil {
			return nil, fmt.Errorf("Cannot open requests file %s: %s", conf.RequestsFile, err)
		}
		reqDecoder := json.NewDecoder(reqFile)
		requests := []*Request{}
		if err := reqDecoder.Decode(requests); err != nil {
			return nil, fmt.Errorf("Cannot parse json requests file %s: %s", conf.RequestsFile, err)
		}
		conf.Requests = append(conf.Requests, requests...)
	}
	if len(conf.Requests) == 0 {
		return nil, fmt.Errorf("No requests supplied")
	}
	for _, r := range conf.Requests {
		if r.BodyFile != "" {
			if r.Body != "" {
				return nil, fmt.Errorf("Bad request: both body and body_file specified.")
			}
			contents, err := ioutil.ReadFile(r.BodyFile)
			if err != nil {
				return nil, fmt.Errorf("Bad request: error reading body_file %s: %s", r.BodyFile, err)
			}
			r.Body = string(contents)
		}
	}
	if conf.MaxConcurrent == 0 {
		conf.MaxConcurrent = 100
	}
	return conf, nil
}