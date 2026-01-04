package main

import (
	"fmt"
	"os"
	"runtime/trace"
	"sync"
	"time"
)

// Sample program demonstrating various blocking patterns for trace analysis

func main() {
	f, err := os.Create("trace.out")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	trace.Start(f)
	defer trace.Stop()

	fmt.Println("Generating trace with blocking patterns...")

	var wg sync.WaitGroup

	// Pattern 1: Channel blocking
	channelBlockingDemo(&wg)

	// Pattern 2: Mutex contention
	mutexContentionDemo(&wg)

	// Pattern 3: Syscall blocking
	syscallBlockingDemo(&wg)

	wg.Wait()

	fmt.Println("Trace generation complete. Run: goschedviz trace.out")
}

// channelBlockingDemo creates goroutines that block on channel operations
func channelBlockingDemo(wg *sync.WaitGroup) {
	ch := make(chan int) // Unbuffered channel for maximum blocking

	// Many receivers waiting on empty channel
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				<-ch // Block waiting for data
				time.Sleep(1 * time.Millisecond)
			}
		}(i)
	}

	// Slow sender creates bottleneck
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 500; i++ {
			time.Sleep(2 * time.Millisecond) // Intentional delay
			ch <- i
		}
		close(ch)
	}()
}

// mutexContentionDemo creates high mutex contention
func mutexContentionDemo(wg *sync.WaitGroup) {
	var mu sync.Mutex
	counter := 0

	// Many goroutines competing for same lock
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				mu.Lock()
				counter++
				time.Sleep(500 * time.Microsecond) // Hold lock longer
				mu.Unlock()
			}
		}()
	}
}

// syscallBlockingDemo performs syscalls that cause blocking
func syscallBlockingDemo(wg *sync.WaitGroup) {
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				// Sleep syscall causes goroutine to block
				time.Sleep(10 * time.Millisecond)
			}
		}()
	}
}
