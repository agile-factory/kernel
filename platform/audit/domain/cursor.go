package domain

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

// CursorToken encodes the keyset position in a paginated audit-log result.
//
// The two fields together form a stable, deterministic key that uniquely
// identifies a row's position in the canonical ordering:
//
//	ORDER BY occurred_at DESC, id ASC
//
// The token is always transmitted as an opaque Base64url-encoded JSON string.
// Clients must never parse or construct cursor values — they are implementation
// details that may change between minor releases.
type CursorToken struct {
	OccurredAt time.Time `json:"occurred_at"`
	ID         string    `json:"id"`
}

// EncodeCursor serialises a CursorToken to an opaque Base64url string
// suitable for inclusion in an HTTP response body.
func EncodeCursor(t CursorToken) (string, error) {
	b, err := json.Marshal(t)
	if err != nil {
		return "", fmt.Errorf("cursor encode: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// DecodeCursor parses the opaque Base64url string produced by EncodeCursor.
// Returns an error if the string is malformed or contains invalid JSON.
func DecodeCursor(s string) (*CursorToken, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("cursor decode (base64): %w", err)
	}
	var t CursorToken
	if err := json.Unmarshal(b, &t); err != nil {
		return nil, fmt.Errorf("cursor decode (json): %w", err)
	}
	if t.ID == "" {
		return nil, fmt.Errorf("cursor decode: missing id field")
	}
	return &t, nil
}

// CursorFromLog builds a CursorToken from the last AuditLog of a page.
// It is used by the use case layer to produce the next_cursor value.
func CursorFromLog(l *AuditLog) CursorToken {
	return CursorToken{
		OccurredAt: l.OccurredAt,
		ID:         l.ID,
	}
}
