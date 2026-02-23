package printer

import (
	"fmt"
	"io"
	"os"

	"github.com/mvasilenko/mv-claude-code-marketplace/internal/style"
)

// Printer handles CLI output with verbose/quiet/json awareness
type Printer struct {
	errWriter io.Writer
	json      bool
	quiet     bool
	verbose   bool
	writer    io.Writer
}

// New creates a printer with the given flags
func New(json, quiet, verbose bool) *Printer {
	return &Printer{
		errWriter: os.Stderr,
		json:      json,
		quiet:     quiet,
		verbose:   verbose,
		writer:    os.Stdout,
	}
}

// Output methods (print unless quiet)

func (p *Printer) Info(msg string) {
	if !p.quiet {
		_, _ = fmt.Fprintln(p.writer, style.InfoMsg(msg))
	}
}

func (p *Printer) Success(msg string) {
	if !p.quiet {
		_, _ = fmt.Fprintln(p.writer, style.SuccessMsg(msg))
	}
}

func (p *Printer) Warning(msg string) {
	if !p.quiet {
		_, _ = fmt.Fprintln(p.errWriter, style.WarningMsg(msg))
	}
}

func (p *Printer) Section(title string) {
	if !p.quiet {
		_, _ = fmt.Fprintln(p.writer, style.Section(title))
	}
}

func (p *Printer) FinalSuccess(msg string) {
	if !p.quiet {
		_, _ = fmt.Fprintln(p.writer, style.FinalSuccess(msg))
	}
}

func (p *Printer) FinalError(msg string) {
	_, _ = fmt.Fprintln(p.errWriter, style.FinalError(msg))
}

func (p *Printer) Bullet(msg string) {
	if !p.quiet {
		_, _ = fmt.Fprintln(p.writer, style.BulletItem(msg))
	}
}

// Verbose returns a VerbosePrinter for verbose-only output
func (p *Printer) Verbose() *VerbosePrinter {
	return &VerbosePrinter{p: p}
}

// Flag accessors
func (p *Printer) IsJSON() bool    { return p.json }
func (p *Printer) IsQuiet() bool   { return p.quiet }
func (p *Printer) IsVerbose() bool { return p.verbose }

// VerbosePrinter only prints when verbose mode is enabled
type VerbosePrinter struct {
	p *Printer
}

func (vp *VerbosePrinter) Info(msg string) {
	if vp.p.verbose && !vp.p.quiet {
		_, _ = fmt.Fprintln(vp.p.writer, style.VerboseMsg(msg))
	}
}

func (vp *VerbosePrinter) Success(msg string) {
	if vp.p.verbose && !vp.p.quiet {
		_, _ = fmt.Fprintln(vp.p.writer, style.SuccessMsg(msg))
	}
}

func (vp *VerbosePrinter) Warning(msg string) {
	if vp.p.verbose && !vp.p.quiet {
		_, _ = fmt.Fprintln(vp.p.errWriter, style.WarningMsg(msg))
	}
}
