package httpserver

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"
)

// Serve runs a PRODUCTION-SAFE http.Server until ctx is cancelled, then shuts down gracefully.
func Serve(ctx context.Context, ln net.Listener, handler http.Handler, log *slog.Logger) error {
	srv := &http.Server{
		Handler: handler,
		// TIMEOUTS: the zero values mean "wait forever", which lets slow or malicious clients hold
		// connections open until the server runs out of resources.
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		// Every request context derives from this, so handlers see shutdown too
		BaseContext: func(net.Listener) context.Context { return context.WithoutCancel(ctx) },
	}

	errCh := make(chan error, 1)
	go func() { errCh <- srv.Serve(ln) }()
	log.Info("listening", "addr", ln.Addr().String())

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	// GRACEFUL SHUTDOWN: stop accepting, wait for in-flight requests (up to 10 s), then return.
	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	if err := <-errCh; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	log.Info("stopped")
	return nil
}
