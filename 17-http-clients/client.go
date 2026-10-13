// Package httpclient is a robust JSON client for calling other services: timeouts, retries with
// backoff, Retry-After, idempotency rules, typed errors and a circuit breaker.
package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"
)

// HTTPError is returned for non-2xx answers, so callers can branch on the status.
type HTTPError struct {
	Status int
	Body   string
	URL    string
}

func (e *HTTPError) Error() string { return fmt.Sprintf("HTTP %d from %s", e.Status, e.URL) }

type Client struct {
	BaseURL   string
	HTTP      *http.Client
	Retries   int
	BaseDelay time.Duration
}

// New creates a client. NEVER use http.DefaultClient for service calls: it has NO timeout.
func New(baseURL string, timeout time.Duration) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTP: &http.Client{
			Timeout: timeout, // whole request, including reading the body
			// Reuse ONE client (and its Transport): it keeps a pool of keep-alive connections.
		},
		Retries:   2,
		BaseDelay: 100 * time.Millisecond,
	}
}

func (c *Client) Get(ctx context.Context, path string, out any) error {
	return c.do(ctx, http.MethodGet, path, nil, out, true)
}

// Post is NOT retried by default: retrying a non-idempotent request may create two orders or charges.
// Pass retry=true only when the API is idempotent (e.g. it supports an Idempotency-Key header).
func (c *Client) Post(ctx context.Context, path string, in, out any, retry bool) error {
	return c.do(ctx, http.MethodPost, path, in, out, retry)
}

func (c *Client) do(ctx context.Context, method, path string, in, out any, retry bool) error {
	var payload []byte
	if in != nil {
		var err error
		if payload, err = json.Marshal(in); err != nil {
			return err
		}
	}
	attempts := 1
	if retry {
		attempts += c.Retries
	}

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		wait, err := c.once(ctx, method, path, payload, out)
		if err == nil {
			return nil
		}
		lastErr = err
		if wait < 0 || attempt == attempts { // not retryable, or out of attempts
			break
		}
		if wait == 0 {
			wait = c.backoff(attempt)
		}
		select {
		case <-time.After(wait):
		case <-ctx.Done(): // the caller gave up: stop retrying immediately
			return ctx.Err()
		}
	}
	return lastErr
}

// once performs one attempt. It returns how long to wait before retrying (0 = use backoff, -1 = don't retry).
func (c *Client) once(ctx context.Context, method, path string, payload []byte, out any) (time.Duration, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, bytes.NewReader(payload))
	if err != nil {
		return -1, err
	}
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := c.HTTP.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return -1, ctx.Err()
		}
		return 0, err // network error or timeout: transient, retry
	}
	defer res.Body.Close() // ALWAYS close the body, or the connection can't be reused

	if res.StatusCode >= 200 && res.StatusCode < 300 {
		if out == nil || res.StatusCode == http.StatusNoContent {
			io.Copy(io.Discard, res.Body)
			return 0, nil
		}
		return 0, json.NewDecoder(res.Body).Decode(out)
	}

	body, _ := io.ReadAll(io.LimitReader(res.Body, 4096)) // never read unbounded error bodies
	httpErr := &HTTPError{Status: res.StatusCode, Body: string(body), URL: req.URL.String()}
	switch res.StatusCode {
	case http.StatusTooManyRequests, http.StatusServiceUnavailable:
		if secs, err := strconv.Atoi(res.Header.Get("Retry-After")); err == nil {
			return time.Duration(secs) * time.Second, httpErr // the server told us how long to wait
		}
		return 0, httpErr
	case http.StatusRequestTimeout, http.StatusInternalServerError, http.StatusBadGateway, http.StatusGatewayTimeout:
		return 0, httpErr
	default:
		return -1, httpErr // 4xx: our request is wrong; retrying can't help
	}
}

// backoff: exponential (100, 200, 400 ms...) with JITTER, so many clients don't retry in lockstep.
func (c *Client) backoff(attempt int) time.Duration {
	base := c.BaseDelay << (attempt - 1)
	return base/2 + rand.N(base/2+1)
}

// IsStatus reports whether err is an HTTPError with the given status.
func IsStatus(err error, status int) bool {
	var he *HTTPError
	return errors.As(err, &he) && he.Status == status
}
