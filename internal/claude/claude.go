// Package claude provides a wrapper around the Claude CLI for marketplace operations.
package claude

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mvasilenko/mv-claude-code-marketplace/internal/platform"
)

// ErrNotFound is returned when the claude CLI is not found in PATH.
var ErrNotFound = fmt.Errorf("claude CLI not found in PATH")

// CLI defines the interface for Claude CLI operations.
type CLI interface {
	AddMarketplace(path string, output io.Writer) error
	RemoveMarketplace(name string, output io.Writer) error
	UpdateMarketplace(name string, output io.Writer) error
}

// cli is the concrete implementation of CLI.
type cli struct {
	configDir string
}

// New creates a new Claude CLI wrapper.
// configDir is the Claude config directory (e.g., ~/.claude).
// Returns ErrNotFound if the claude CLI is not available in PATH.
func New(configDir string, plat platform.Platform) (CLI, error) {
	if !plat.CommandExists("claude") {
		return nil, ErrNotFound
	}
	return &cli{configDir: configDir}, nil
}

// AddMarketplace registers a marketplace with Claude using its local path.
// The path must be an absolute path to the marketplace directory.
func (c *cli) AddMarketplace(path string, output io.Writer) error {
	// Claude CLI only accepts relative paths starting with "./"
	parentDir := filepath.Dir(path)
	folderName := filepath.Base(path)
	relativePath := "./" + folderName

	cmd := c.command("plugin", "marketplace", "add", relativePath)
	cmd.Dir = parentDir

	return c.runWithOutput(cmd, output)
}

// RemoveMarketplace unregisters a marketplace from Claude.
func (c *cli) RemoveMarketplace(name string, output io.Writer) error {
	cmd := c.command("plugin", "marketplace", "remove", name)
	return c.runWithOutput(cmd, output)
}

// UpdateMarketplace updates a marketplace's plugins in Claude.
func (c *cli) UpdateMarketplace(name string, output io.Writer) error {
	cmd := c.command("plugin", "marketplace", "update", name)
	return c.runWithOutput(cmd, output)
}

// command creates an exec.Cmd for running the claude CLI with proper environment setup.
func (c *cli) command(args ...string) *exec.Cmd {
	cmd := exec.Command("claude", args...)
	if c.configDir != "" {
		cmd.Env = append(os.Environ(), fmt.Sprintf("CLAUDE_CONFIG_DIR=%s", c.configDir))
	}
	return cmd
}

// runWithOutput executes a command and handles output capturing.
func (c *cli) runWithOutput(cmd *exec.Cmd, output io.Writer) error {
	var stderrBuf strings.Builder

	if output != nil {
		cmd.Stdout = output
		cmd.Stderr = io.MultiWriter(output, &stderrBuf)
	} else {
		cmd.Stderr = &stderrBuf
	}

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderrBuf.String())
		if errMsg != "" {
			return fmt.Errorf("%w\nOutput: %s", err, errMsg)
		}
		return err
	}
	return nil
}
