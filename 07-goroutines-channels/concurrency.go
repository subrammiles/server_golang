// Package concurrency covers goroutines and channels: "don't communicate by sharing memory;
// share memory by communicating".
package concurrency

import (
	"context"
	"errors"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// ParallelSum splits the work across goroutines. A GOROUTINE is a very cheap thread managed by
// the Go runtime (a few KB of stack): starting thousands is normal.
func ParallelSum(numbers []int, parts int) int {
	results := make([]int, parts)
	chunk := (len(numbers) + parts - 1) / parts
	var wg sync.WaitGroup
	for p := range parts {
		wg.Go(func() { // Go 1.25: starts the goroutine and tracks it (Add/Done done for you)
			start, end := p*chunk, min((p+1)*chunk, len(numbers))
			for _, n := range numbers[min(start, end):end] {
				results[p] += n // each goroutine writes ONLY its own slot: no data race
			}
		})
	}
	wg.Wait() // block until every goroutine has finished
	total := 0
	for _, r := range results {
		total += r
	}
	return total
}

// ---- PIPELINE: stages connected by CHANNELS ----

// Generate sends numbers on a channel and CLOSES it when done (the sender closes, never the receiver).
func Generate(nums ...int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for _, n := range nums {
			out <- n // blocks until a receiver is ready (unbuffered channel)
		}
	}()
	return out
}

// Square is a stage: receive, transform, send.
func Square(in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for n := range in { // range over a channel ends when it is closed
			out <- n * n
		}
	}()
	return out
}

// Merge (FAN-IN) combines several channels into one.
func Merge(channels ...<-chan int) <-chan int {
	out := make(chan int)
	var wg sync.WaitGroup
	for _, ch := range channels {
		wg.Go(func() {
			for v := range ch {
				out <- v
			}
		})
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

// ---- WORKER POOL: a fixed number of goroutines process a queue of jobs ----

type Result struct {
	Job    int
	Output int
}

func WorkerPool(jobs []int, workers int, process func(int) int) []Result {
	jobCh := make(chan int)
	resultCh := make(chan Result, len(jobs)) // BUFFERED: sends don't block until it's full

	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			for job := range jobCh {
				resultCh <- Result{Job: job, Output: process(job)}
			}
		})
	}
	for _, j := range jobs {
		jobCh <- j
	}
	close(jobCh) // tell workers there's no more work
	wg.Wait()
	close(resultCh)

	results := make([]Result, 0, len(jobs))
	for r := range resultCh {
		results = append(results, r)
	}
	return results
}

// ---- SELECT: wait on several channel operations at once ----

var ErrTimeout = errors.New("timed out")

// FirstOf asks several "mirrors" and returns the fastest answer, or gives up after the timeout.
func FirstOf(timeout time.Duration, mirrors ...func() string) (string, error) {
	answers := make(chan string, len(mirrors)) // buffered so slow mirrors don't leak goroutines
	for _, m := range mirrors {
		go func() { answers <- m() }()
	}
	select {
	case a := <-answers:
		return a, nil
	case <-time.After(timeout):
		return "", ErrTimeout
	}
}

// ---- errgroup: goroutines that return errors, with a concurrency limit and shared cancellation ----

// FetchAll runs fetch for every id (at most `limit` at once). The FIRST error cancels the context
// for the others and is returned. This is the go-to pattern for parallel calls in real services.
func FetchAll(ctx context.Context, ids []int, limit int, fetch func(context.Context, int) (string, error)) ([]string, error) {
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(limit)
	results := make([]string, len(ids))
	for i, id := range ids {
		g.Go(func() error {
			v, err := fetch(ctx, id)
			if err != nil {
				return err
			}
			results[i] = v
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return results, nil
}
