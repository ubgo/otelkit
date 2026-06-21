// shutdown.go — Telemetry.Shutdown and RunOnSignal.
//
// Shutdown flushes/shuts down providers in order — logs, then metrics, then
// traces — so a prior phase's errors still reach the collectors before those
// providers close; all are attempted and errors aggregate via errors.Join.
// RunOnSignal blocks until SIGTERM/SIGINT (or ctx cancel), then runs Shutdown on
// a FRESH background context so a cancelled app context can't abort the flush.

package otelkit

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"
)

// Shutdown flushes and shuts down every provider in order — logs, then
// metrics, then traces — so a prior phase's errors still reach the collector
// before its provider closes. All providers are attempted regardless of
// individual failures; the errors are aggregated with errors.Join.
func (t *Telemetry) Shutdown(ctx context.Context) error {
	var errs []error
	for _, s := range t.shutdowns {
		if err := s(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// RunOnSignal blocks until ctx is cancelled or one of SIGINT/SIGTERM arrives,
// then runs Shutdown and returns its error. It is the one-call graceful-shutdown
// helper for long-running services.
//
// Shutdown runs on a fresh background context — not the (possibly already
// cancelled) ctx that ended the wait — so a cancelled app context cannot abort
// the flush. For a bounded shutdown deadline, call Shutdown(ctxWithTimeout)
// directly instead.
func (t *Telemetry) RunOnSignal(ctx context.Context) error {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(ch)

	select {
	case <-ctx.Done():
	case <-ch:
	}
	return t.Shutdown(context.Background())
}
