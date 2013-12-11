package main

import (
	"bytes"
	"fmt"
	"math"
	"text/template"

	"github.com/bmizerany/perks/quantile"
)

const quantileEpsilon = 0.001

var (
	// These are defaults
	// TODO: I could make these not globals and pass them into NewStats but I'm too lazy for now.
	quants  = []float64{0.50, 0.90, 0.99}
	buckets []float64 // By default don't report buckets
)

type Stats struct {
	quantiles *quantile.Stream
	Count     float64
	// This has len(buckets) + 1 counts, unless buckets is empty in which case we don't record a single trivial
	// bucket.
	buckets []float64

	// The rest are in milliseconds
	Min          float64
	Max          float64
	total        float64
	sumOfSquares float64
}

func NewStats() *Stats {
	quantiles := quantile.NewTargeted(quants...)
	quantiles.SetEpsilon(quantileEpsilon)
	var b []float64
	if len(buckets) > 0 {
		b = make([]float64, len(buckets)+1)
	}
	return &Stats{
		buckets:   b,
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

type bucketWithPercents struct {
	Description string
	Count       float64
	Percent     float64
}

func (s *Stats) BucketsPercents() []*bucketWithPercents {
	// Length of s.buckets is always 0 or >= 2.
	if len(s.buckets) == 0 {
		return nil
	}
	result := make([]*bucketWithPercents, len(s.buckets))
	for i, bucket := range s.buckets {
		var description string
		switch i {
		case 0:
			description = fmt.Sprintf("less than %.1f ms", buckets[0])
		case len(s.buckets) - 1:
			description = fmt.Sprintf("more than %.1f ms", buckets[len(buckets)-1])
		default:
			description = fmt.Sprintf("%.1f - %.1f ms", buckets[i-1], buckets[i])
		}
		result[i] = &bucketWithPercents{
			Description: description,
			Count:       bucket,
			Percent:     (bucket / s.Count) * 100,
		}
	}
	return result
}

func (s *Stats) Insert(r *Result) {
	s.Count++
	v := r.LatencyMillis
	if v < s.Min {
		s.Min = v
	}
	if v > s.Max {
		s.Max = v
	}
	s.total += v
	s.sumOfSquares += v * v
	s.quantiles.Insert(v)

	// NOTE: could binary search if there's a huge list of buckets.
	if len(buckets) > 0 {
		var i int
		for i = 0; i < len(buckets); i++ {
			if r.LatencyMillis < buckets[i] {
				break
			}
		}
		s.buckets[i]++
	}
}

var StatsTmpl = template.Must(template.New("stats").Parse(
	`Mean           {{printf "%10.3f" .Mean}} ms
Std. Deviation {{printf "%10.3f" .StdDev}} ms
Min            {{printf "%10.3f" .Min}} ms
Max            {{printf "%10.3f" .Max}} ms
{{range .Quantiles}}Quantile {{index . 0 | printf "%0.3f"}} {{index . 1 | printf "%10.3f"}} ms
{{end}}
{{if $buckets := .BucketsPercents}}Latency bucket counts
{{range $bucket := $buckets}}{{printf "%-20s" .Description}} {{printf "%7.0f" .Count}} ({{printf "%.1f" .Percent}}%)
{{end}}{{end}}`))

func (s *Stats) String() string {
	buf := &bytes.Buffer{}
	if err := StatsTmpl.Execute(buf, s); err != nil {
		panic(err)
	}
	return buf.String()
}
