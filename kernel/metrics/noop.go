package metrics

import "time"

// NoopRecorder is a Recorder that discards all observations.
// Use in tests or when observability is disabled.
type NoopRecorder struct{}

func (NoopRecorder) RecordRequest(_, _ string, _ int, _ time.Duration) {}
func (NoopRecorder) RecordDBPoolStats(_, _ int)                        {}
func (NoopRecorder) RecordAuditWrite(_ bool)                           {}
