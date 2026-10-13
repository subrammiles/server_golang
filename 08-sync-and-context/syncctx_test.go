package syncctx

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

// Run with -race: the race detector would flag any unsynchronised access.
func TestAccountIsSafeForConcurrentUse(t *testing.T) {
	var acc Account
	var wg sync.WaitGroup
	for range 1000 {
		wg.Go(func() { acc.Deposit(1) })
	}
	wg.Wait()
	if acc.Balance() != 1000 {
		t.Fatalf("balance = %d, want 1000", acc.Balance())
	}
}

func TestOnlyOneWithdrawalOfTheLastCoinSucceeds(t *testing.T) {
	var acc Account
	acc.Deposit(1)
	var ok atomic.Int32
	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() {
			if acc.Withdraw(1) == nil {
				ok.Add(1)
			}
		})
	}
	wg.Wait()
	if ok.Load() != 1 || acc.Balance() != 0 {
		t.Fatalf("%d withdrawals succeeded, balance %d", ok.Load(), acc.Balance())
	}
}

func TestAtomicCounterAndRWMutex(t *testing.T) {
	var stats Stats
	cfg := NewConfig()
	var wg sync.WaitGroup
	for i := range 100 {
		wg.Go(func() {
			stats.Requests.Add(1)
			if i%10 == 0 {
				cfg.Set("mode", "busy")
			}
			_ = cfg.Get("mode")
		})
	}
	wg.Wait()
	if stats.Requests.Load() != 100 || cfg.Get("mode") != "busy" {
		t.Fatal("unexpected state")
	}
}

func TestOnce(t *testing.T) {
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() { _ = Settings() })
	}
	wg.Wait()
	if loadCount.Load() != 1 || Settings()["region"] != "ap-south-1" {
		t.Fatalf("loaded %d times", loadCount.Load())
	}
}

// testing/synctest (Go 1.25): time is FAKE inside the bubble. These tests "wait" seconds
// but finish instantly and deterministically: no flaky sleeps.
func TestContextDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx := context.WithValue(context.Background(), RequestIDKey, "req-42")

		got, err := HandleRequest(ctx, 2*time.Second, time.Second) // query fits in the budget
		if err != nil || got != "rows for request req-42" {
			t.Fatalf("got %q, %v", got, err)
		}

		_, err = HandleRequest(ctx, time.Second, 5*time.Second) // query too slow
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("want DeadlineExceeded, got %v", err)
		}
	})
}

func TestCancellationStopsTheWorker(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		ticks := 0
		done := Ticker(ctx, time.Minute, func() { ticks++ })

		time.Sleep(5*time.Minute + time.Second) // instant in the bubble
		cancel()
		<-done // the goroutine exited: no leak

		if ticks != 5 {
			t.Fatalf("ticks = %d, want 5", ticks)
		}
	})
}

func TestCancelledContextAbortsImmediately(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	if _, err := SlowQuery(ctx, time.Hour); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
	if time.Since(start) > time.Second {
		t.Fatal("should not have waited")
	}
}
