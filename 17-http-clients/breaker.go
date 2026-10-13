package httpclient

import (
	"errors"
	"sync"
	"time"
)

// ErrOpen is returned without calling the dependency while the circuit is open.
var ErrOpen = errors.New("circuit open: failing fast")

// Breaker is a CIRCUIT BREAKER: after `threshold` consecutive failures it OPENS and rejects calls
// for `cooldown` (fail fast, give the dependency time to recover), then lets ONE trial call through
// (HALF-OPEN). Success closes it again; failure re-opens it.
type Breaker struct {
	mu        sync.Mutex
	threshold int
	cooldown  time.Duration
	now       func() time.Time
	failures  int
	openedAt  time.Time
	state     string
}

func NewBreaker(threshold int, cooldown time.Duration) *Breaker {
	return &Breaker{threshold: threshold, cooldown: cooldown, now: time.Now, state: "closed"}
}

func (b *Breaker) Call(fn func() error) error {
	b.mu.Lock()
	if b.state == "open" {
		if b.now().Sub(b.openedAt) < b.cooldown {
			b.mu.Unlock()
			return ErrOpen
		}
		b.state = "half-open"
	}
	b.mu.Unlock()

	err := fn()

	b.mu.Lock()
	defer b.mu.Unlock()
	if err == nil {
		b.state, b.failures = "closed", 0
		return nil
	}
	b.failures++
	if b.state == "half-open" || b.failures >= b.threshold {
		b.state, b.openedAt = "open", b.now()
	}
	return err
}

func (b *Breaker) State() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.state
}
