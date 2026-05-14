// Package logger provides the centralised, structured logging infrastructure
// for the Agile Factory backend.
//
// # Fan-out architecture
//
// Every log record is simultaneously written to:
//
//   - stdout            — always (text in dev, JSON in prod/staging)
//   - logs/app.log      — all levels, JSON (full audit trail)
//   - logs/error.log    — ERROR and above only, JSON (incident triage)
//   - logs/server.log   — records tagged source="http", JSON (access log)
//
// # File rotation
//
// Files are opened with O_APPEND. External tools (logrotate, Docker log
// drivers) can rotate them safely. Signal-based re-open can be added by the
// caller using the returned closer + os.Signal.
//
// # Environment variables
//
//	LOG_LEVEL   — debug | info | warn | error  (default: info)
//	LOG_DIR     — directory for log files       (default: ./logs)
//	APP_ENV     — development | staging | production
//
// Usage:
//
//	log, closeFn, err := logger.New(logger.Config{
//	    Level:       cfg.LogLevel,
//	    Environment: cfg.Environment,
//	    Dir:         cfg.LogDir,
//	})
//	if err != nil { … }
//	defer closeFn()
//	slog.SetDefault(log)
package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// ─── configuration ──────────────────────────────────────────────────────────

// Config holds the logger configuration.
// It intentionally mirrors only what the logger needs to avoid import cycles
// with kernel/config.
type Config struct {
	// Level is the minimum log level string: debug | info | warn | error.
	Level string
	// Environment determines stdout handler format (text vs JSON).
	Environment string
	// Dir is the directory where log files are created.
	// Defaults to "./logs" when empty.
	Dir string
}

// ─── public constructor ──────────────────────────────────────────────────────

// New creates a fan-out *slog.Logger writing to stdout and to per-category
// log files under cfg.Dir.
//
// The returned closer function must be called on graceful shutdown to flush
// and close all file handles.
//
// Backward-compatible note: the legacy NewSimple(levelStr, env) is kept for
// call sites that do not yet pass a Config.
func New(cfg Config) (*slog.Logger, func(), error) {
	level := parseLevel(cfg.Level)
	dir := cfg.Dir
	if dir == "" {
		dir = "logs"
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, nil, fmt.Errorf("logger: cannot create log directory %q: %w", dir, err)
	}

	appFile, err := openLogFile(filepath.Join(dir, "app.log"))
	if err != nil {
		return nil, nil, err
	}
	errFile, err := openLogFile(filepath.Join(dir, "error.log"))
	if err != nil {
		_ = appFile.Close()
		return nil, nil, err
	}
	srvFile, err := openLogFile(filepath.Join(dir, "server.log"))
	if err != nil {
		_ = appFile.Close()
		_ = errFile.Close()
		return nil, nil, err
	}

	// stdout: text in dev, JSON in prod/staging.
	stdoutOpts := &slog.HandlerOptions{Level: level}
	var stdoutHandler slog.Handler
	if isProd(cfg.Environment) {
		stdoutHandler = slog.NewJSONHandler(os.Stdout, stdoutOpts)
	} else {
		stdoutHandler = slog.NewTextHandler(os.Stdout, stdoutOpts)
	}

	// app.log — all levels, JSON, with source location.
	appHandler := slog.NewJSONHandler(appFile, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	})

	// error.log — ERROR and above only.
	errHandler := slog.NewJSONHandler(errFile, &slog.HandlerOptions{
		Level:     slog.LevelError,
		AddSource: true,
	})

	// server.log — records that carry source="http" (written by the HTTP
	// middleware via log.With("source", "http")).
	srvHandler := &filteredHandler{
		inner: slog.NewJSONHandler(srvFile, &slog.HandlerOptions{
			Level:     slog.LevelDebug,
			AddSource: false,
		}),
		attrKey: "source",
		attrVal: "http",
	}

	fan := &fanHandler{
		handlers: []slog.Handler{
			stdoutHandler,
			appHandler,
			errHandler,
			srvHandler,
		},
	}

	closer := func() {
		_ = appFile.Close()
		_ = errFile.Close()
		_ = srvFile.Close()
	}

	return slog.New(fan), closer, nil
}

// NewSimple is the original constructor preserved for backward compatibility.
// Writes only to stdout. Prefer New() in production.
func NewSimple(levelStr, env string) *slog.Logger {
	level := parseLevel(levelStr)
	opts := &slog.HandlerOptions{Level: level}
	if isProd(env) {
		return slog.New(slog.NewJSONHandler(os.Stdout, opts))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, opts))
}

// ─── fanHandler ─────────────────────────────────────────────────────────────

// fanHandler implements slog.Handler by forwarding every record to all child
// handlers. Each child decides independently whether to emit the record based
// on its own level filter.
type fanHandler struct {
	handlers []slog.Handler
}

func (f *fanHandler) Enabled(_ context.Context, level slog.Level) bool {
	for _, h := range f.handlers {
		if h.Enabled(context.Background(), level) {
			return true
		}
	}
	return false
}

func (f *fanHandler) Handle(ctx context.Context, r slog.Record) error {
	var firstErr error
	for _, h := range f.handlers {
		if h.Enabled(ctx, r.Level) {
			if err := h.Handle(ctx, r.Clone()); err != nil && firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

func (f *fanHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := make([]slog.Handler, len(f.handlers))
	for i, h := range f.handlers {
		next[i] = h.WithAttrs(attrs)
	}
	return &fanHandler{handlers: next}
}

func (f *fanHandler) WithGroup(name string) slog.Handler {
	next := make([]slog.Handler, len(f.handlers))
	for i, h := range f.handlers {
		next[i] = h.WithGroup(name)
	}
	return &fanHandler{handlers: next}
}

// ─── filteredHandler ────────────────────────────────────────────────────────

// filteredHandler forwards records to inner only when the record contains a
// specific top-level attribute key=value pair.
//
// Used to route HTTP request logs to server.log without touching other handlers.
// The HTTP logger middleware should call:
//
//	log.With("source", "http").Info("request", …)
type filteredHandler struct {
	inner   slog.Handler
	attrKey string
	attrVal string
}

func (fh *filteredHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return fh.inner.Enabled(ctx, level)
}

func (fh *filteredHandler) Handle(ctx context.Context, r slog.Record) error {
	var match bool
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == fh.attrKey && a.Value.String() == fh.attrVal {
			match = true
			return false
		}
		return true
	})
	if !match {
		return nil
	}
	return fh.inner.Handle(ctx, r)
}

func (fh *filteredHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// Check if the pinned attribute is being baked in via WithAttrs.
	// If so, all future records from this child will always match.
	pinned := false
	for _, a := range attrs {
		if a.Key == fh.attrKey && a.Value.String() == fh.attrVal {
			pinned = true
			break
		}
	}
	if pinned {
		// Once pinned, skip the filter and always forward.
		return fh.inner.WithAttrs(attrs)
	}
	return &filteredHandler{
		inner:   fh.inner.WithAttrs(attrs),
		attrKey: fh.attrKey,
		attrVal: fh.attrVal,
	}
}

func (fh *filteredHandler) WithGroup(name string) slog.Handler {
	return &filteredHandler{
		inner:   fh.inner.WithGroup(name),
		attrKey: fh.attrKey,
		attrVal: fh.attrVal,
	}
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func openLogFile(path string) (io.WriteCloser, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("logger: cannot open log file %q: %w", path, err)
	}
	return f, nil
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func isProd(env string) bool {
	return env == "production" || env == "staging"
}
