package events

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRunHandler_PanicBecomesError(t *testing.T) {
	d := &OutboxDispatcher{handlerTimeout: time.Second}
	err := d.runHandler(context.Background(), func(context.Context, Event) error { panic("boom") }, Event{})
	if err == nil {
		t.Fatal("expected an error from a panicking handler")
	}
}

func TestRunHandler_Deadline(t *testing.T) {
	d := &OutboxDispatcher{handlerTimeout: 10 * time.Millisecond}
	err := d.runHandler(context.Background(), func(ctx context.Context, _ Event) error {
		<-ctx.Done()
		return ctx.Err()
	}, Event{})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("want DeadlineExceeded, got %v", err)
	}
}

func TestBackoff_Capped(t *testing.T) {
	if got := backoff(0); got != time.Second {
		t.Fatalf("backoff(0) = %s", got)
	}
	if got := backoff(50); got != 5*time.Minute {
		t.Fatalf("backoff(50) = %s", got)
	}
}
