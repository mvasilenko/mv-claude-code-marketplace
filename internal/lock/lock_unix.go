//go:build unix

package lock

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// TryLock attempts to acquire the lock (non-blocking)
func (l *fileLockImpl) TryLock() (bool, error) {
	if l.file == nil {
		return false, fmt.Errorf("lock file not open")
	}

	err := unix.Flock(int(l.file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if err != nil {
		if err == unix.EWOULDBLOCK {
			return false, nil
		}
		return false, fmt.Errorf("failed to lock file: %w", err)
	}

	return true, nil
}

// Unlock releases the lock
func (l *fileLockImpl) Unlock() error {
	if l.file == nil {
		return nil
	}

	if err := unix.Flock(int(l.file.Fd()), unix.LOCK_UN); err != nil {
		return fmt.Errorf("failed to unlock: %w", err)
	}

	return nil
}
