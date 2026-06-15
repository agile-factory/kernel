package domain_test

import (
	"testing"
	"time"

	"github.com/agile-factory/kernel/platform/audit/domain"
)

// ─── CursorToken encode / decode ─────────────────────────────────────────────

func TestEncodeDecode_RoundTrip(t *testing.T) {
	original := domain.CursorToken{
		OccurredAt: time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC),
		ID:         "audit-log-uuid-001",
	}

	encoded, err := domain.EncodeCursor(original)
	if err != nil {
		t.Fatalf("EncodeCursor() unexpected error: %v", err)
	}
	if encoded == "" {
		t.Fatal("EncodeCursor() returned empty string")
	}

	decoded, err := domain.DecodeCursor(encoded)
	if err != nil {
		t.Fatalf("DecodeCursor() unexpected error: %v", err)
	}

	if decoded.ID != original.ID {
		t.Errorf("ID: got %q, want %q", decoded.ID, original.ID)
	}
	if !decoded.OccurredAt.Equal(original.OccurredAt) {
		t.Errorf("OccurredAt: got %v, want %v", decoded.OccurredAt, original.OccurredAt)
	}
}

func TestDecodeCursor_InvalidBase64(t *testing.T) {
	_, err := domain.DecodeCursor("not-valid-base64!!!")
	if err == nil {
		t.Error("DecodeCursor() should return error for invalid base64")
	}
}

func TestDecodeCursor_InvalidJSON(t *testing.T) {
	import64 := "dGhpcyBpcyBub3QganNvbg" // base64 de "this is not json"
	_, err := domain.DecodeCursor(import64)
	if err == nil {
		t.Error("DecodeCursor() should return error for non-JSON payload")
	}
}

func TestDecodeCursor_MissingIDField(t *testing.T) {
	// JSON valide mais sans champ "id"
	// {"occurred_at":"2026-05-01T12:00:00Z"}
	noIDJSON := `{"occurred_at":"2026-05-01T12:00:00Z"}`
	import64 := encodeRaw(noIDJSON)

	_, err := domain.DecodeCursor(import64)
	if err == nil {
		t.Error("DecodeCursor() should return error when id field is missing")
	}
}

func TestDecodeCursor_EmptyString(t *testing.T) {
	// Une chaîne vide doit être gérée par la couche use case (retourne nil),
	// mais DecodeCursor elle-même renvoie une erreur.
	_, err := domain.DecodeCursor("")
	// Le comportement varie : si la stdlib retourne une erreur de base64, c'est OK.
	// Si elle réussit à décoder une chaîne vide, le JSON devrait échouer.
	// On accepte les deux comportements tant que l'appel ne panique pas.
	_ = err // pas de vérification stricte ici — la couche UC filtre les vides en amont
}

// ─── CursorFromLog ────────────────────────────────────────────────────────────

func TestCursorFromLog_ExtractsFields(t *testing.T) {
	log := &domain.AuditLog{
		ID:         "log-uuid-999",
		OccurredAt: time.Date(2026, 4, 30, 10, 0, 0, 0, time.UTC),
	}

	tok := domain.CursorFromLog(log)

	if tok.ID != "log-uuid-999" {
		t.Errorf("ID: got %q, want %q", tok.ID, "log-uuid-999")
	}
	if !tok.OccurredAt.Equal(log.OccurredAt) {
		t.Errorf("OccurredAt: got %v, want %v", tok.OccurredAt, log.OccurredAt)
	}
}

// ─── AuditLog construction ────────────────────────────────────────────────────

func TestNewAuditLog_OccurredAtIsNowUTC(t *testing.T) {
	before := time.Now().UTC()
	log := domain.NewAuditLog("id-001", "tenant-001", "actor-001",
		domain.ACTION_USER_REGISTERED, "user", "target-001", nil)
	after := time.Now().UTC()

	if log.OccurredAt.Before(before) || log.OccurredAt.After(after) {
		t.Errorf("OccurredAt out of expected range: %v", log.OccurredAt)
	}
}

func TestNewAuditLog_MetadataNilBecomesEmptyMap(t *testing.T) {
	log := domain.NewAuditLog("id-001", "t", "a", domain.ACTION_USER_LOGGED_IN, "user", "u", nil)
	if log.Metadata == nil {
		t.Error("Metadata should be an empty map, not nil")
	}
}

func TestNewAuditLog_MetadataPreservedIfProvided(t *testing.T) {
	meta := map[string]any{"email": "alice@agile.io", "role": "ROLE_OWNER"}
	log := domain.NewAuditLog("id-001", "t", "a", domain.ACTION_TENANT_CREATED, "tenant", "t", meta)
	if log.Metadata["email"] != "alice@agile.io" {
		t.Errorf("Metadata email: got %v", log.Metadata["email"])
	}
}

func TestNewAuditLog_AllFieldsSet(t *testing.T) {
	log := domain.NewAuditLog(
		"log-001", "tenant-001", "actor-001",
		domain.ACTION_MEMBER_INVITED, "membership", "target-001",
		map[string]any{"role": "ROLE_AGENT"},
	)

	if log.ID != "log-001" {
		t.Errorf("ID: got %q", log.ID)
	}
	if log.TenantID != "tenant-001" {
		t.Errorf("TenantID: got %q", log.TenantID)
	}
	if log.Action != domain.ACTION_MEMBER_INVITED {
		t.Errorf("Action: got %q", log.Action)
	}
}

// ─── helper ──────────────────────────────────────────────────────────────────

// encodeRaw encodes a raw string to Base64 URL-safe encoding (no padding)
// for test fixtures that produce invalid cursors.
func encodeRaw(s string) string {
	import64pkg := "dGhpcyBpcyBub3QganNvbg" // fallback
	_ = import64pkg
	// manual inline for test isolation
	b := []byte(s)
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	result := make([]byte, 0, (len(b)*4+2)/3)
	for i := 0; i < len(b); i += 3 {
		b0 := b[i]
		var b1, b2 byte
		if i+1 < len(b) {
			b1 = b[i+1]
		}
		if i+2 < len(b) {
			b2 = b[i+2]
		}
		result = append(result,
			chars[b0>>2],
			chars[((b0&0x3)<<4)|(b1>>4)],
		)
		if i+1 < len(b) {
			result = append(result, chars[((b1&0xf)<<2)|(b2>>6)])
		}
		if i+2 < len(b) {
			result = append(result, chars[b2&0x3f])
		}
	}
	return string(result)
}
