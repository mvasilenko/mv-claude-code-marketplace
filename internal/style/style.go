package style

import (
	"os"

	"github.com/charmbracelet/lipgloss"
)

// Semantic styles for CLI output
var (
	// Bold style - for emphasis on commands, paths
	Bold = lipgloss.NewStyle().Bold(true)

	// CommandOutput style - dim italic for external command output
	CommandOutput = lipgloss.NewStyle().Faint(true).Italic(true)

	// Dim style - for secondary information
	Dim = lipgloss.NewStyle().Faint(true)

	// Error style - bright red for errors, failures
	Error = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))

	// Info style - cyan for informational messages, progress
	Info = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))

	// Success style - green for checkmarks, completion messages
	Success = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))

	// Warning style - yellow for warnings, cautions
	Warning = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
)

// ColorsEnabled returns whether color output should be used.
// Respects NO_COLOR environment variable.
func ColorsEnabled() bool {
	return os.Getenv("NO_COLOR") == ""
}
