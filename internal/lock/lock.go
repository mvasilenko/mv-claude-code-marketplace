package lock

import (
	"fmt"
	"os"
	"path/filepath"
)

// FileLock represents a file-based lock
type FileLock interface {
	Close() error           // Close file handle
	TryLock() (bool, error) // Non-blocking lock acquisition
	Unlock() error          // Release lock
}

// fileLockImpl is the platform-specific implementation
type fileLockImpl struct {
	file *os.File
	path string
}

// NewFileLock creates a new file lock
func NewFileLock(lockPath string) (FileLock, error) {
	if err := os.MkdirAll(filepath.Dir(lockPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create lock directory: %w", err)
	}

	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open lock file: %w", err)
	}

	return &fileLockImpl{
		file: file,
		path: lockPath,
	}, nil
}

// Close closes the file handle
func (l *fileLockImpl) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}
