package lock

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFileLock(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "test.lock")
	fl, err := NewFileLock(lockPath)
	require.NoError(t, err)
	require.NotNil(t, fl)
	defer fl.Close() //nolint:errcheck
}

func TestFileLock_TryLockAndUnlock(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "test.lock")
	fl, err := NewFileLock(lockPath)
	require.NoError(t, err)
	defer fl.Close() //nolint:errcheck

	// First lock should succeed
	acquired, err := fl.TryLock()
	require.NoError(t, err)
	assert.True(t, acquired)

	// Unlock should succeed
	err = fl.Unlock()
	require.NoError(t, err)

	// Re-lock should succeed after unlock
	acquired, err = fl.TryLock()
	require.NoError(t, err)
	assert.True(t, acquired)

	err = fl.Unlock()
	require.NoError(t, err)
}

func TestFileLock_ConcurrentLock(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "test.lock")

	fl1, err := NewFileLock(lockPath)
	require.NoError(t, err)
	defer fl1.Close() //nolint:errcheck

	fl2, err := NewFileLock(lockPath)
	require.NoError(t, err)
	defer fl2.Close() //nolint:errcheck

	// First lock succeeds
	acquired, err := fl1.TryLock()
	require.NoError(t, err)
	assert.True(t, acquired)

	// Second lock should fail (non-blocking)
	acquired, err = fl2.TryLock()
	require.NoError(t, err)
	assert.False(t, acquired)

	// After unlock, second lock should succeed
	err = fl1.Unlock()
	require.NoError(t, err)

	acquired, err = fl2.TryLock()
	require.NoError(t, err)
	assert.True(t, acquired)

	err = fl2.Unlock()
	require.NoError(t, err)
}

func TestFileLock_Close(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "test.lock")
	fl, err := NewFileLock(lockPath)
	require.NoError(t, err)

	err = fl.Close()
	require.NoError(t, err)
}
