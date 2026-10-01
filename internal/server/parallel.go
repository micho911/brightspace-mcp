package server

import "sync"

// maxParallel bounds concurrent requests to Brightspace from one tool call.
const maxParallel = 4

// forEachLimit calls fn(0..n-1) with at most maxParallel calls at once and
// waits for all of them. fn writes its result into a slot of its own.
func forEachLimit(n int, fn func(i int)) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxParallel)
	for i := range n {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			fn(i)
		}()
	}
	wg.Wait()
}
