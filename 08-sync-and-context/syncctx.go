// Package syncctx covers shared memory (mutexes, atomics, Once) and context (cancellation, deadlines).
package syncctx

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// Account protects its balance with a MUTEX: only one goroutine at a time may run the locked section.
type Account struct {
	mu      sync.Mutex
	balance int
}

func (a *Account) Deposit(amount int) {
	a.mu.Lock()
	defer a.mu.Unlock() // defer guarantees the unlock even if the code below panics
	a.balance += amount
}

var ErrInsufficient = errors.New("insufficient funds")

// Withdraw: the check AND the update happen under one lock. Checking, unlocking, then updating
// would let two goroutines both pass the check (a race condition).
func (a *Account) Withdraw(amount int) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.balance < amount {
		return ErrInsufficient
	}
	a.balance -= amount
	return nil
}

func (a *Account) Balance() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.balance
}

// Stats uses ATOMICS: lock-free operations for simple counters.
type Stats struct {
	Requests atomic.Int64
}

// Config is READ often and WRITTEN rarely: an RWMutex lets many readers in at once.
type Config struct {
	mu     sync.RWMutex
	values map[string]string
}

func NewConfig() *Config { return &Config{values: map[string]string{}} }

func (c *Config) Get(key string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.values[key]
}

func (c *Config) Set(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values[key] = value
}

// sync.Once runs an expensive initialisation exactly once, even with many concurrent callers.
var (
	loadOnce  sync.Once
	loadCount atomic.Int32
	settings  map[string]string
)

func Settings() map[string]string {
	loadOnce.Do(func() {
		loadCount.Add(1)
		time.Sleep(10 * time.Millisecond) // e.g. read a file or call a service
		settings = map[string]string{"region": "ap-south-1"}
	})
	return settings
}

// ---- CONTEXT: carries cancellation, deadlines and request-scoped values across API boundaries ----

type ctxKey string

const RequestIDKey ctxKey = "requestID" // a private key type avoids collisions with other packages

// SlowQuery pretends to be a database call that honours cancellation. EVERY blocking call
// in a server should take a ctx as its FIRST parameter and give up when ctx is done.
func SlowQuery(ctx context.Context, d time.Duration) (string, error) {
	select {
	case <-time.After(d):
		id, _ := ctx.Value(RequestIDKey).(string)
		return "rows for request " + id, nil
	case <-ctx.Done():
		return "", ctx.Err() // context.Canceled or context.DeadlineExceeded
	}
}

// HandleRequest gives the whole operation a deadline; every call below shares it.
func HandleRequest(parent context.Context, budget, queryTime time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(parent, budget)
	defer cancel() // ALWAYS call cancel to release the timer, even on success
	return SlowQuery(ctx, queryTime)
}

// Ticker is a background worker that stops cleanly when its context is cancelled (no goroutine leak).
func Ticker(ctx context.Context, every time.Duration, tick func()) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		t := time.NewTicker(every)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				tick()
			case <-ctx.Done():
				return
			}
		}
	}()
	return done
}
