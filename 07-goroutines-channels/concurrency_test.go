package concurrency

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync/atomic"
	"testing"
	"time"
)

func ExampleParallelSum() {
	nums := make([]int, 1000)
	for i := range nums {
		nums[i] = i + 1
	}
	fmt.Println(ParallelSum(nums, 4))
	// Output: 500500
}

func Example_pipeline() {
	total := 0
	for sq := range Square(Generate(1, 2, 3, 4)) {
		total += sq
	}
	fmt.Println(total)
	// Output: 30
}

func ExampleMerge() {
	var got []int
	for v := range Merge(Generate(1, 2), Generate(10, 20)) {
		got = append(got, v)
	}
	slices.Sort(got) // fan-in order is not deterministic
	fmt.Println(got)
	// Output: [1 2 10 20]
}

func Example_bufferedChannel() {
	ch := make(chan string, 2)
	ch <- "a" // doesn't block: there's room in the buffer
	ch <- "b"
	fmt.Println(len(ch), cap(ch), <-ch, <-ch)
	v, ok := <-ch // not reached while open & empty... so close first:
	_ = v
	_ = ok
	// Output: 2 2 a b
}

func TestWorkerPool(t *testing.T) {
	var running, maxRunning atomic.Int32
	results := WorkerPool([]int{1, 2, 3, 4, 5, 6}, 2, func(n int) int {
		now := running.Add(1)
		for {
			old := maxRunning.Load()
			if now <= old || maxRunning.CompareAndSwap(old, now) {
				break
			}
		}
		time.Sleep(5 * time.Millisecond)
		running.Add(-1)
		return n * 10
	})
	if len(results) != 6 {
		t.Fatalf("got %d results", len(results))
	}
	slices.SortFunc(results, func(a, b Result) int { return a.Job - b.Job })
	if results[5].Output != 60 {
		t.Errorf("job 6 -> %d", results[5].Output)
	}
	if maxRunning.Load() > 2 {
		t.Errorf("%d workers ran at once, want at most 2", maxRunning.Load())
	}
}

func TestFirstOf(t *testing.T) {
	slow := func() string { time.Sleep(200 * time.Millisecond); return "slow" }
	fast := func() string { time.Sleep(5 * time.Millisecond); return "fast" }

	if got, err := FirstOf(time.Second, slow, fast); err != nil || got != "fast" {
		t.Fatalf("got %q, %v", got, err)
	}
	if _, err := FirstOf(20*time.Millisecond, slow); !errors.Is(err, ErrTimeout) {
		t.Fatalf("want timeout, got %v", err)
	}
}

func TestFetchAll(t *testing.T) {
	t.Run("collects results in order, respecting the limit", func(t *testing.T) {
		var inFlight, peak atomic.Int32
		got, err := FetchAll(context.Background(), []int{1, 2, 3, 4, 5}, 2, func(ctx context.Context, id int) (string, error) {
			n := inFlight.Add(1)
			if n > peak.Load() {
				peak.Store(n)
			}
			time.Sleep(5 * time.Millisecond)
			inFlight.Add(-1)
			return fmt.Sprint("user-", id), nil
		})
		if err != nil || !slices.Equal(got, []string{"user-1", "user-2", "user-3", "user-4", "user-5"}) {
			t.Fatalf("got %v, %v", got, err)
		}
		if peak.Load() > 2 {
			t.Errorf("peak concurrency %d > 2", peak.Load())
		}
	})

	t.Run("the first error cancels the rest", func(t *testing.T) {
		var cancelled atomic.Int32
		_, err := FetchAll(context.Background(), []int{1, 2, 3}, 3, func(ctx context.Context, id int) (string, error) {
			if id == 1 {
				return "", errors.New("user 1 failed")
			}
			select {
			case <-ctx.Done():
				cancelled.Add(1)
				return "", ctx.Err()
			case <-time.After(time.Second):
				return "late", nil
			}
		})
		if err == nil || err.Error() != "user 1 failed" {
			t.Fatalf("got %v", err)
		}
		if cancelled.Load() != 2 {
			t.Errorf("%d goroutines saw the cancellation, want 2", cancelled.Load())
		}
	})
}
