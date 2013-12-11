package main

import (
	"bytes"
	"text/template"
	"time"
)

type ResultStats struct {
	Succeeded int64
	Failed    int64
	Duration  time.Duration
	Total     *Stats
	ByStatus  map[int]*Stats
}

func NewResultStats() *ResultStats {
	return &ResultStats{
		Total:    NewStats(),
		ByStatus: make(map[int]*Stats),
	}
}

func (s *ResultStats) QPS() float64 {
	return float64(s.Succeeded) / float64(s.Duration.Seconds())
}

func (s *ResultStats) Insert(r *Result) {
	if r.Failed {
		s.Failed++
		return
	}
	s.Succeeded++
	s.Total.Insert(r)
	stats, ok := s.ByStatus[r.StatusCode]
	if !ok {
		stats = NewStats()
		s.ByStatus[r.StatusCode] = stats
	}
	stats.Insert(r)
}

func (s *ResultStats) PercentSuccessful() float64 {
	return float64(s.Succeeded) / float64(s.Succeeded+s.Failed) * 100
}

func (s *ResultStats) PercentFailed() float64 {
	return float64(s.Failed) / float64(s.Succeeded+s.Failed) * 100
}

var resultStatsFuncs = template.FuncMap{
	"divpct": func(a float64, b int64) float64 { return a / float64(b) * 100 },
}

var ResultStatsTmpl = template.Must(template.New("ResultStats").Funcs(resultStatsFuncs).Parse(
	`=== Summary ===
Test duration:            {{printf "%10.3f seconds" .Duration.Seconds}}
Successful requests:         {{printf "%7d" .Succeeded}} ({{printf "%.1f%%" .PercentSuccessful}})
Failed requests:             {{printf "%7d" .Failed}} ({{printf "%.1f%%" .PercentFailed}})
Successful request rate:  {{printf "%10.3f" .QPS}} requests / sec

=== Overall latencies ===
{{.Total}}

=== Breakdown by response status code ===
{{range $status, $_ := .ByStatus}}
--- Status {{$status}} ({{.Count}} requests | {{divpct .Count $.Succeeded | printf "%.3f"}}% of total) ---
{{.}}
{{end}}`))

func (s *ResultStats) String() string {
	buf := &bytes.Buffer{}
	if err := ResultStatsTmpl.Execute(buf, s); err != nil {
		panic(err)
	}
	return buf.String()
}
