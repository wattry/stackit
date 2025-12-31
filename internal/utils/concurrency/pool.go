// Package concurrency provides utilities for concurrent operations.
package concurrency

import (
	"runtime"
	"sync"
)

// Run runs the given worker function for each item in the slice in parallel.
// It uses runtime.GOMAXPROCS(0) as the default number of workers.
func Run[T any](items []T, worker func(item T)) {
	if len(items) == 0 {
		return
	}

	numWorkers := runtime.GOMAXPROCS(0)
	if numWorkers > len(items) {
		numWorkers = len(items)
	}

	jobs := make(chan T, len(items))
	for _, item := range items {
		jobs <- item
	}
	close(jobs)

	var wg sync.WaitGroup
	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func() {
			defer wg.Done()
			for item := range jobs {
				worker(item)
			}
		}()
	}
	wg.Wait()
}

// RunWithWorkers runs the given worker function for each item in the slice in parallel with a specified number of workers.
func RunWithWorkers[T any](items []T, numWorkers int, worker func(item T)) {
	if len(items) == 0 {
		return
	}

	if numWorkers <= 0 {
		numWorkers = runtime.GOMAXPROCS(0)
	}

	if numWorkers > len(items) {
		numWorkers = len(items)
	}

	jobs := make(chan T, len(items))
	for _, item := range items {
		jobs <- item
	}
	close(jobs)

	var wg sync.WaitGroup
	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func() {
			defer wg.Done()
			for item := range jobs {
				worker(item)
			}
		}()
	}
	wg.Wait()
}
