package main

import (
	"fmt"
	"math/rand"
	"net/http"
	_ "net/http/pprof" // Essential for goschedviz to connect!
	"sync"
	"time"
)

// This program simulates a "busy" web server with some performance issues.
// run: go run examples/live_server.go

func main() {
	var mu sync.Mutex
	var sharedData int

	// 1. Simulate background workers fighting for a lock (Mutex Contention)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for {
				mu.Lock()
				// Simulate some work inside the lock (bad practice, but good for visualization!)
				time.Sleep(10 * time.Millisecond)
				sharedData++
				mu.Unlock()

				// Simulate internal processing
				time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)
			}
		}(i)
	}

	// 2. Simulate Channel communication
	dataChan := make(chan int)
	go func() {
		for {
			// Producer
			dataChan <- rand.Intn(100)
			time.Sleep(50 * time.Millisecond)
		}
	}()
	go func() {
		for {
			// Consumer
			<-dataChan
		}
	}()

	fmt.Println("🚀 Live Server is running on http://localhost:6060")
	fmt.Println("👉 Now open 'goschedviz' in another terminal and choose option [1]")
	fmt.Println("(Press Ctrl+C to stop this server)")

	// Start the Pprof server
	if err := http.ListenAndServe("localhost:6060", nil); err != nil {
		fmt.Printf("Error starting server: %v\n", err)
	}
}
