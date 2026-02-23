package logger

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithLoggerAndFromContext(t *testing.T) {
	tests := []struct {
		ctx    context.Context
		name   string
		setLog bool
	}{
		{
			name:   "returns logger from context",
			ctx:    context.Background(),
			setLog: true,
		},
		{
			name:   "returns discard logger when not set",
			ctx:    context.Background(),
			setLog: false,
		},
		{
			name:   "returns discard logger for nil context",
			ctx:    nil,
			setLog: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setLog && tt.ctx != nil {
				var buf bytes.Buffer
				l := slog.New(slog.NewJSONHandler(&buf, nil))
				ctx := WithLogger(tt.ctx, l)
				got := FromContext(ctx)
				require.NotNil(t, got)
				got.Info("test message")
				assert.Contains(t, buf.String(), "test message")
			} else {
				got := FromContext(tt.ctx)
				require.NotNil(t, got)
				// Should not panic — discard logger
				got.Info("should not panic")
			}
		})
	}
}

func TestNewRedactionReplaceAttr(t *testing.T) {
	replaceAttr := NewRedactionReplaceAttr()
	require.NotNil(t, replaceAttr)

	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		ReplaceAttr: replaceAttr,
	})
	l := slog.New(handler)

	// Log a message with a sensitive value
	l.Info("test", "key", "AKIAIOSFODNN7EXAMPLE")
	output := buf.String()

	// The AWS key should be redacted
	assert.NotContains(t, output, "AKIAIOSFODNN7EXAMPLE")
	assert.Contains(t, output, "test")
}
