package metadata

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mvasilenko/mv-claude-code-marketplace/internal/managed"
)

// MetadataFilename is the name of the metadata file
const MetadataFilename = "claude-settings-metadata.json"

// Manager handles claudectl metadata file operations
type Manager struct {
	filePath string
}

// NewManager creates a metadata manager.
// configDir should be the claudectl config directory.
func NewManager(configDir string) *Manager {
	return &Manager{
		filePath: filepath.Join(configDir, MetadataFilename),
	}
}

// GetFilePath returns the metadata file path
func (m *Manager) GetFilePath() string {
	return m.filePath
}

// Load reads metadata from file.
// Returns nil if file doesn't exist (first run).
func (m *Manager) Load() (managed.Entries, error) {
	if _, err := os.Stat(m.filePath); os.IsNotExist(err) {
		return nil, nil
	}

	data, err := os.ReadFile(m.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	var entries managed.Entries
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("invalid metadata JSON: %w", err)
	}

	return entries, nil
}

// Save writes metadata to file atomically.
func (m *Manager) Save(entries managed.Entries) error {
	if err := os.MkdirAll(filepath.Dir(m.filePath), 0755); err != nil {
		return fmt.Errorf("failed to create metadata directory: %w", err)
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	data = append(data, '\n')

	tmpPath := m.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	if err := os.Rename(tmpPath, m.filePath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to finalize metadata: %w", err)
	}

	return nil
}
