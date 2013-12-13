package main

import (
	"bytes"
	"strings"
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

func indent(s, prefix string) string {
	s = strings.TrimSpace(s)
	indentReplacer := strings.NewReplacer("\n", "\n"+prefix)
	prefixed := prefix + indentReplacer.Replace(s)
	return prefixed
}

var resultStatsFuncs = template.FuncMap{
	"div":    func(n, d float64) float64 { return n / d },
	"divpct": func(a float64, b int64) float64 { return a / float64(b) * 100 },
	"indent": indent,
}

var ResultStatsTmpl = template.Must(template.New("ResultStats").Funcs(resultStatsFuncs).Parse(
	`╔══╡ Summary
║ Test duration:           {{printf "%9.3f" .Duration.Seconds}} s
║ Successful requests:        {{printf "%6d" .Succeeded}} ({{printf "%.1f%%" .PercentSuccessful}})
║ Failed requests:            {{printf "%6d" .Failed}} ({{printf "%.1f%%" .PercentFailed}})
║ Successful request rate: {{printf "%9.3f" .QPS}} req/s
╚═

╔══╡ Overall latencies
{{indent .Total.String "║ "}}
╚═

╔══╡ Breakdown by response status code
║{{range $status, $_ := .ByStatus}}
║ ┌──┤ Status {{$status}} ({{.Count}} req | {{divpct .Count $.Succeeded | printf "%.1f"}}% of total | {{div .Count $.Duration.Seconds | printf "%.3f"}} req/s)
{{indent .String "║ │ "}}
║ └{{end}}
╚═`))

func (s *ResultStats) String() string {
	s.Total.Duration = s.Duration
	for _, stat := range s.ByStatus {
		stat.Duration = s.Duration
	}
	buf := &bytes.Buffer{}
	if err := ResultStatsTmpl.Execute(buf, s); err != nil {
		panic(err)
	}
	return buf.String()
}
