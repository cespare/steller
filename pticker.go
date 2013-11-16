package main

import (
	"math/rand"
	"time"
)

func init() { rand.Seed(time.Now().UnixNano()) }

// A PTicker is similar to a time.Ticker, except that instead of a constant delay between ticks, it uses an
// exponentially distributed delay (modeling a Poisson process). Additionally, it sends an empty struct on C
// (rather than a time.Time) because I don't use this value for my purposes.
type PTicker struct {
	C    chan struct{}
	rate float64
	done chan struct{}
}

// NewPTicker makes a PTicker with a given (per-second) rate paramter.
func NewPTicker(rate float64) *PTicker {
	t := &PTicker{
		C:    make(chan struct{}),
		rate: rate,
		done: make(chan struct{}),
	}
	go func() {
		for {
			time.Sleep(time.Duration(rand.ExpFloat64() / t.rate * float64(time.Second)))
			select {
			case <-t.done:
				return
			case t.C <- struct{}{}:
			default:
				// Don't do anything if nobody's listening
			}
		}
	}()
	return t
}

func (t *PTicker) Stop() {
	if t.done != nil {
		t.done <- struct{}{}
		t.done = nil
	}
}
