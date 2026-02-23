package metadata

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mvasilenko/mv-claude-code-marketplace/internal/managed"
)

func TestManager_LoadSave(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Load on missing file returns nil
	entries, err := mgr.Load()
	require.NoError(t, err)
	assert.Nil(t, entries)

	// Save entries
	toSave := managed.Entries{
		"env":               {"VAR1", "VAR2"},
		"permissions.allow": {"Bash(git:*)"},
	}
	err = mgr.Save(toSave)
	require.NoError(t, err)

	// Load returns saved entries
	loaded, err := mgr.Load()
	require.NoError(t, err)
	assert.Equal(t, toSave, loaded)
}

func TestManager_GetFilePath(t *testing.T) {
	mgr := NewManager("/tmp/test-config")
	assert.Equal(t, filepath.Join("/tmp/test-config", MetadataFilename), mgr.GetFilePath())
}

func TestManager_SaveCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	nestedDir := filepath.Join(tmpDir, "nested", "dir")
	mgr := NewManager(nestedDir)

	err := mgr.Save(managed.Entries{"key": {"val"}})
	require.NoError(t, err)

	loaded, err := mgr.Load()
	require.NoError(t, err)
	assert.Equal(t, managed.Entries{"key": {"val"}}, loaded)
}
