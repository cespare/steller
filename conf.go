package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
)

// A Request is a user-configured request (which can be used to make an http.Request).
type Request struct {
	Method   string
	URL      string
	Body     string
	BodyFile string `json:"body_file"`
	Headers  map[string]string
	Weight   *float64 // Pointer to distinguish whether the user provided the number or not
}

type ReportingStats struct {
	Quantiles []float64
	Buckets   []float64
}

type TargetQPS struct {
	unlimited bool
	qps       int
}

type Conf struct {
	Requests        []*Request
	RequestsFile    string          `json:"requests_file"`
	TargetQPS       TargetQPS       `json:"target_qps"`
	DurationSeconds int             `json:"duration_seconds"`
	MaxConcurrent   int             `json:"max_concurrent"`
	ReportingStats  *ReportingStats `json:"reporting_stats"`
}

func (q *TargetQPS) UnmarshalJSON(b []byte) error {
	if bytes.Equal(b, []byte(`"unlimited"`)) {
		q.unlimited = true
		return nil
	}
	q.unlimited = false
	if err := json.Unmarshal(b, &q.qps); err != nil {
		return err
	}
	return nil
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
		if err := reqDecoder.Decode(&requests); err != nil {
			return nil, fmt.Errorf("Cannot parse json requests file %s: %s", conf.RequestsFile, err)
		}
		conf.Requests = append(conf.Requests, requests...)
	}
	for _, request := range conf.Requests {
		if request.Weight == nil {
			request.Weight = new(float64)
			*request.Weight = 1
		}
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

	if conf.ReportingStats != nil {
		sort.Float64s(conf.ReportingStats.Quantiles)
		sort.Float64s(conf.ReportingStats.Buckets)
		for _, q := range conf.ReportingStats.Quantiles {
			if q < 0 || q >= 1 {
				return nil, fmt.Errorf("Bad quantile %f", q)
			}
		}
		if len(conf.ReportingStats.Buckets) > 0 {
			last := conf.ReportingStats.Buckets[0]
			for i, b := range conf.ReportingStats.Buckets {
				if b <= 0 {
					return nil, fmt.Errorf("Bucket is <= 0: %f", b)
				}
				if i > 0 && b == last {
					return nil, fmt.Errorf("Duplicate buckets: %f", b)
				}
				last = b
			}
		}
	}
	return conf, nil
}
