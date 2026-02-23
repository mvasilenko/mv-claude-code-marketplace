package logger

import (
	"log/slog"
	"regexp"

	"github.com/m-mizutani/masq"
)

// NewRedactionReplaceAttr creates a masq ReplaceAttr function with patterns
// for sensitive data like AWS keys, API tokens, and passwords.
func NewRedactionReplaceAttr() func(groups []string, a slog.Attr) slog.Attr {
	return masq.New(
		// AWS credentials patterns
		masq.WithRegex(regexp.MustCompile(`AKIA[A-Z0-9]{16}`)),
		masq.WithRegex(regexp.MustCompile(`(?i)aws_secret_access_key[=:]\s*\S+`)),
		masq.WithRegex(regexp.MustCompile(`(?i)aws_session_token[=:]\s*\S+`)),

		// Generic API keys and tokens
		masq.WithRegex(regexp.MustCompile(`(?i)(api[_-]?key|token|bearer)[=:]\s*['\"]?[a-zA-Z0-9_\-\.]{20,}['\"]?`)),

		// Generic password patterns
		masq.WithRegex(regexp.MustCompile(`(?i)(password|passwd|pwd)[=:]\s*\S+`)),

		// Anthropic API keys
		masq.WithRegex(regexp.MustCompile(`sk-ant-[a-zA-Z0-9\-_]{95,}`)),
	)
}
