package main

import (
	"fmt"
	"net/http"
)

const (
	addr = "localhost:6666"
	//delay         = 10 * time.Millisecond
	//maxConcurrent = 1000
)

var (
// Using a buffered channel as a kind of "multi-mutex" to limit the number of concurrent requests to
// maxConcurrent.
//limitChan = make(chan struct{}, maxConcurrent)
)

func handler(w http.ResponseWriter, r *http.Request) {
	//limitChan <- struct{}{}
	//time.Sleep(delay)
	//<-limitChan
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handler)
	//qps := float64(maxConcurrent) / float64(delay.Seconds())
	//fmt.Printf("delay = %s, max concurrent = %d, max qps = %0.3f\n", delay, maxConcurrent, qps)
	fmt.Println("Now listening on", addr)
	fmt.Println(http.ListenAndServe(addr, mux))
}
