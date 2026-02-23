// Package logger provides logging utilities for claudectl, including sensitive data redaction
// for structured logging using Go's slog library with the masq redaction library.
package logger

import (
	"context"
	"io"
	"log/slog"
)

// contextKey is an unexported type for context keys to avoid collisions.
type contextKey string

const loggerContextKey contextKey = "logger"

// WithLogger returns a new context with the logger attached.
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey, logger)
}

// FromContext extracts the logger from context. If no logger is found or context is nil,
// returns a discard logger (no-op).
func FromContext(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return slog.New(slog.NewJSONHandler(io.Discard, nil))
	}
	if logger, ok := ctx.Value(loggerContextKey).(*slog.Logger); ok && logger != nil {
		return logger
	}
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}
