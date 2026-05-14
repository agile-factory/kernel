package txctx

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
)

// fakeTx is the minimal pgx.Tx surface needed to test context propagation
// without a real database connection. Only the methods exercised by these
// tests have meaningful behaviour; the rest panic if called.
type fakeTx struct{ pgx.Tx }

func TestWithTxFromContext_RoundTrip(t *testing.T) {
	tx := &fakeTx{}
	ctx := WithTx(context.Background(), tx)

	got, ok := FromContext(ctx)
	if !ok {
		t.Fatalf("FromContext: expected ok=true after WithTx")
	}
	if got.(*fakeTx) != tx {
		t.Fatalf("FromContext: expected the exact tx instance back")
	}
}

func TestFromContext_Missing(t *testing.T) {
	if _, ok := FromContext(context.Background()); ok {
		t.Fatalf("FromContext: expected ok=false on a plain context")
	}
}

func TestMustFromContext_NoTx(t *testing.T) {
	_, err := MustFromContext(context.Background())
	if !errors.Is(err, ErrNoTx) {
		t.Fatalf("MustFromContext: expected ErrNoTx, got %v", err)
	}
}
