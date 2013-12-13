package main

import (
	"math/rand"
	"net/http"
)

// A WeightedRequests is like a []*Request except that each request is associated with a weight, and it
// supports random weighted sampling.
type WeightedRequests struct {
	total        float64
	equalWeights bool
	requests     []WeightedRequest
}

type WeightedRequest struct {
	// When passed in, this is some weight, but inside a WeightedRequests this is a *running total*.
	Weight float64
	*http.Request
}

func NewWeightedRequests(requests []WeightedRequest) *WeightedRequests {
	if len(requests) == 0 {
		panic("unhandled empty requests")
	}
	var total float64
	newRequests := make([]WeightedRequest, len(requests))
	firstWeight := requests[0].Weight
	equalWeights := true
	for i, r := range requests {
		if r.Weight != firstWeight {
			equalWeights = false
		}
		total += r.Weight
		newRequests[i] = WeightedRequest{total, r.Request}
	}
	return &WeightedRequests{
		total:        total,
		equalWeights: equalWeights,
		requests:     newRequests,
	}
}

func (r *WeightedRequests) Random() *http.Request {
	// Special-case 1 for speed
	if len(r.requests) == 1 {
		return r.requests[0].Request
	}
	// Special-case all the weights being equal (probably the usual case) and generate a random index directly.
	if r.equalWeights {
		x := rand.Intn(len(r.requests))
		return r.requests[x].Request
	}

	// TODO: Probably faster not to binary search for, say, < 20 elements. Benchmark and see.
	x := rand.Float64() * r.total
	low := 0
	high := len(r.requests) - 1
	// Loop invariant: if i is the index to be chosen, x < request[i].weight and x >= request[i-1].weight
	// (or 0).
	// Termination: see comments below -- in either case, the bounds shrink each iteration.
	for low < high {
		mid := (low + high) / 2
		if x < r.requests[mid].Weight {
			high = mid // In case of an even number of elements, mid < high, so high always moves down here.
		} else {
			low = mid + 1 // Clearly mid >= low, so low always increases here.
		}
	}
	return r.requests[low].Request
}
