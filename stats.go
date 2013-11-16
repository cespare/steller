package main

import (
	"bytes"
	"math"
	"text/template"
	"time"

	"github.com/bmizerany/perks/quantile"
)

var (
	quants          = []float64{0.50, 0.90, 0.95, 0.99}
	quantileEpsilon = 0.001
)

type Stats struct {
	quantiles *quantile.Stream
	Duration  time.Duration
	Count     float64

	// The rest are in milliseconds
	Min          float64
	Max          float64
	total        float64
	sumOfSquares float64
}

func NewStats() *Stats {
	quantiles := quantile.NewTargeted(quants...)
	quantiles.SetEpsilon(quantileEpsilon)
	return &Stats{
		Min:       math.MaxFloat64,
		quantiles: quantiles,
	}
}

// Mean returns the mean value in milliseconds
func (s *Stats) Mean() float64 { return s.total / s.Count }

// http://en.wikipedia.org/wiki/Standard_deviation#Rapid_calculation_methods
func (s *Stats) StdDev() float64 {
	return math.Sqrt((s.Count*s.sumOfSquares)-(s.total*s.total)) / s.Count
}

func (s *Stats) Quantiles() [][2]float64 {
	result := [][2]float64{}
	for _, quant := range quants {
		result = append(result, [2]float64{quant, s.quantiles.Query(quant)})
	}
	return result
}

func (s *Stats) QPS() float64 {
	return s.Count / float64(s.Duration.Seconds())
}

func (s *Stats) Insert(v float64) {
	s.Count++
	if v < s.Min {
		s.Min = v
	}
	if v > s.Max {
		s.Max = v
	}
	s.total += v
	s.sumOfSquares += v * v
	s.quantiles.Insert(v)
}

var StatsTmpl = template.Must(template.New("stats").Parse(
	`=== Summary ===
Test duration:            {{printf "%10.3f seconds" .Duration.Seconds}}
Successful requests:         {{printf "%7.0f" .Count}}
Mean requests per second: {{printf "%10.3f" .QPS}}

=== Request latencies (ms) ===
Mean:           {{printf "%10.3f" .Mean}}
Std. Deviation: {{printf "%10.3f" .StdDev}}
Min:            {{printf "%10.3f" .Min}}
Max:            {{printf "%10.3f" .Max}}
{{range .Quantiles}}Quantile {{index . 0 | printf "%0.3f"}}: {{index . 1 | printf "%10.3f"}}
{{end}}`))

func (s *Stats) String() string {
	buf := &bytes.Buffer{}
	err := StatsTmpl.Execute(buf, s)
	if err != nil {
		panic(err)
	}
	return buf.String()
}
