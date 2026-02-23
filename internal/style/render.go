package style

import (
	"fmt"
	"io"
	"os"
)

// CommandOutputIndent is the default indentation for external command output
const CommandOutputIndent = "      "

// NewCommandOutputWriter creates an IndentWriter for stdout with default indentation
func NewCommandOutputWriter() *IndentWriter {
	return NewIndentWriter(os.Stdout, CommandOutputIndent)
}

// NewCommandErrorWriter creates an IndentWriter for stderr with default indentation
func NewCommandErrorWriter() *IndentWriter {
	return NewIndentWriter(os.Stderr, CommandOutputIndent)
}

// Section renders a section header with blank line before
func Section(title string) string {
	return fmt.Sprintf("\n%s", Bold.Render(title))
}

// SuccessMsg renders a success message with green checkmark
func SuccessMsg(msg string) string {
	return fmt.Sprintf("  %s %s", Success.Render(SymbolSuccess), msg)
}

// ErrorMsg renders an error message with red cross
func ErrorMsg(msg string) string {
	return fmt.Sprintf("  %s %s", Error.Render(SymbolError), msg)
}

// ErrorMsgWithErr renders an error message with error details
func ErrorMsgWithErr(msg string, err error) string {
	return fmt.Sprintf("  %s %s: %v", Error.Render(SymbolError), msg, err)
}

// WarningMsg renders a warning message with yellow symbol (indented)
func WarningMsg(msg string) string {
	return fmt.Sprintf("  %s %s", Warning.Render(SymbolWarning), msg)
}

// InfoMsg renders an info message with cyan arrow
func InfoMsg(msg string) string {
	return fmt.Sprintf("  %s %s", Info.Render(SymbolArrow), msg)
}

// VerboseMsg renders a verbose/debug message in dim style
func VerboseMsg(msg string) string {
	return "  " + Dim.Render(fmt.Sprintf("%s %s", SymbolArrow, msg))
}

// Command renders a command in bold (for instructions)
func Command(cmd string) string {
	return Bold.Render(cmd)
}

// BulletItem renders a bullet point item
func BulletItem(item string) string {
	return fmt.Sprintf("  %s %s", Success.Render(SymbolBullet), item)
}

// FinalSuccess renders a final success message (no indentation, fully colored)
func FinalSuccess(msg string) string {
	return Success.Render(fmt.Sprintf("%s %s", SymbolSuccess, msg))
}

// FinalError renders a final/global error message (no indentation, fully colored)
func FinalError(msg string) string {
	return Error.Render(fmt.Sprintf("%s Error: %s", SymbolError, msg))
}

// IndentWriter wraps a writer and prefixes each line with indentation and styling
type IndentWriter struct {
	atLineStart bool
	buf         []byte
	indent      string
	writer      io.Writer
}

// NewIndentWriter creates a writer that indents each line
func NewIndentWriter(w io.Writer, indent string) *IndentWriter {
	return &IndentWriter{
		atLineStart: true,
		buf:         make([]byte, 0, 256),
		indent:      indent,
		writer:      w,
	}
}

// Write implements io.Writer, adding indentation and styling at the start of each line
func (iw *IndentWriter) Write(p []byte) (n int, err error) {
	n = len(p)

	for _, b := range p {
		if b == '\r' {
			iw.flushLine()
			_, _ = iw.writer.Write([]byte{'\r'})
			iw.atLineStart = true
			continue
		}

		if b == '\n' {
			iw.flushLine()
			_, _ = iw.writer.Write([]byte{'\n'})
			iw.atLineStart = true
			continue
		}

		iw.buf = append(iw.buf, b)
	}

	return n, nil
}

// flushLine writes the buffered content with indent and styling
func (iw *IndentWriter) flushLine() {
	if len(iw.buf) == 0 {
		return
	}
	if iw.atLineStart {
		_, _ = io.WriteString(iw.writer, iw.indent)
	}
	styled := CommandOutput.Render(string(iw.buf))
	_, _ = io.WriteString(iw.writer, styled)
	iw.buf = iw.buf[:0]
	iw.atLineStart = false
}

// Flush writes any remaining buffered content
func (iw *IndentWriter) Flush() {
	iw.flushLine()
}
