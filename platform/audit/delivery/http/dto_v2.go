package http

// AuditLogListResponseV2 is the cursor-based paginated response envelope.
//
// Key differences from v1:
//   - total is always -1 (no COUNT(*) is executed)
//   - offset is always 0 (keyset pagination does not use offsets)
//   - next_cursor contains an opaque token for the next page (omitted on last page)
//   - has_more signals whether additional pages exist
//
// Clients MUST use has_more / next_cursor to drive pagination and MUST NOT
// rely on total for anything other than detecting the cursor mode (value -1).
type AuditLogListResponseV2 struct {
	Data       []AuditLogResponse `json:"data"`
	Total      int                `json:"total"`                  // always -1 in cursor mode
	Limit      int                `json:"limit"`
	Offset     int                `json:"offset"`                 // always 0 in cursor mode
	NextCursor string             `json:"next_cursor,omitempty"`  // absent on last page
	HasMore    bool               `json:"has_more"`
}
