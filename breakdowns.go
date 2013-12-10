package main

import (
	"fmt"
	"strconv"
	"time"
)

type BreakdownKeyVal struct {
	Name string
	*Stats
}

func (b BreakdownKeyVal) String() string {
	return fmt.Sprintf("%s:\n%s\n", b.Name, b.Stats)
}

type BreakdownKeyVals []BreakdownKeyVal

func (b BreakdownKeyVals) Len() int           { return len(b) }
func (b BreakdownKeyVals) Less(i, j int) bool { return b[i].Name < b[j].Name }
func (b BreakdownKeyVals) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }

type Breakdown interface {
	Description() string
	Insert(r *Result)
	SetDuration(time.Duration)
	AllStats() BreakdownKeyVals
}

type Total struct {
	*Stats
}

func (t Total) Description() string         { return "Overall results" }
func (t Total) AllStats() BreakdownKeyVals  { return BreakdownKeyVals{{"total", t.Stats}} }
func (t Total) SetDuration(d time.Duration) { t.Duration = d }

type StatusCodeBreakdown map[int]*Stats

func (b StatusCodeBreakdown) Description() string { return "Breakdown by response status code" }

func (b StatusCodeBreakdown) AllStats() BreakdownKeyVals {
	var kvs BreakdownKeyVals
	for status, stats := range b {
		name := strconv.Itoa(status)
		kvs = append(kvs, BreakdownKeyVal{name, stats})
	}
	return kvs
}

func (b StatusCodeBreakdown) Insert(r *Result) {
	stats, ok := b[r.StatusCode]
	if !ok {
		stats = NewStats()
		b[r.StatusCode] = stats
	}
	stats.Insert(r)
}

func (b StatusCodeBreakdown) SetDuration(d time.Duration) {
	for _, stats := range b {
		stats.Duration = d
	}
}

func NewBreakdowns() []Breakdown {
	return []Breakdown{&Total{NewStats()}, make(StatusCodeBreakdown)}
}
