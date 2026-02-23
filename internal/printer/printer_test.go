package printer

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrinter(t *testing.T) {
	tests := []struct {
		action  func(p *Printer)
		json    bool
		name    string
		quiet   bool
		verbose bool
		wantErr string
		wantOut string
	}{
		{
			name:    "info prints message",
			action:  func(p *Printer) { p.Info("hello") },
			wantOut: "→",
		},
		{
			name:    "success prints checkmark",
			action:  func(p *Printer) { p.Success("done") },
			wantOut: "✓",
		},
		{
			name:    "warning prints to stderr",
			action:  func(p *Printer) { p.Warning("careful") },
			wantErr: "⚠",
		},
		{
			name:    "quiet suppresses info",
			quiet:   true,
			action:  func(p *Printer) { p.Info("hello") },
			wantOut: "",
		},
		{
			name:    "quiet suppresses success",
			quiet:   true,
			action:  func(p *Printer) { p.Success("done") },
			wantOut: "",
		},
		{
			name:    "final error always prints",
			quiet:   true,
			action:  func(p *Printer) { p.FinalError("bad") },
			wantErr: "✗",
		},
		{
			name:    "bullet prints item",
			action:  func(p *Printer) { p.Bullet("item") },
			wantOut: "•",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			p := New(tt.json, tt.quiet, tt.verbose)
			p.writer = &stdout
			p.errWriter = &stderr

			tt.action(p)

			if tt.wantOut != "" {
				assert.Contains(t, stdout.String(), tt.wantOut)
			} else {
				assert.Empty(t, stdout.String())
			}
			if tt.wantErr != "" {
				assert.Contains(t, stderr.String(), tt.wantErr)
			}
		})
	}
}

func TestVerbosePrinter(t *testing.T) {
	tests := []struct {
		action  func(p *Printer)
		name    string
		quiet   bool
		verbose bool
		wantOut string
	}{
		{
			name:    "verbose info prints when verbose enabled",
			verbose: true,
			action:  func(p *Printer) { p.Verbose().Info("detail") },
			wantOut: "→",
		},
		{
			name:    "verbose info silent when verbose disabled",
			verbose: false,
			action:  func(p *Printer) { p.Verbose().Info("detail") },
			wantOut: "",
		},
		{
			name:    "verbose info silent when quiet",
			verbose: true,
			quiet:   true,
			action:  func(p *Printer) { p.Verbose().Info("detail") },
			wantOut: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			p := New(false, tt.quiet, tt.verbose)
			p.writer = &stdout

			tt.action(p)

			if tt.wantOut != "" {
				assert.Contains(t, stdout.String(), tt.wantOut)
			} else {
				assert.Empty(t, stdout.String())
			}
		})
	}
}

func TestPrinterFlags(t *testing.T) {
	tests := []struct {
		json    bool
		name    string
		quiet   bool
		verbose bool
	}{
		{name: "all false", json: false, quiet: false, verbose: false},
		{name: "json true", json: true, quiet: false, verbose: false},
		{name: "quiet true", json: false, quiet: true, verbose: false},
		{name: "verbose true", json: false, quiet: false, verbose: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(tt.json, tt.quiet, tt.verbose)
			assert.Equal(t, tt.json, p.IsJSON())
			assert.Equal(t, tt.quiet, p.IsQuiet())
			assert.Equal(t, tt.verbose, p.IsVerbose())
		})
	}
}
