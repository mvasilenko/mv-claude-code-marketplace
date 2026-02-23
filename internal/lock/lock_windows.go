//go:build windows

package lock

import (
	"fmt"
	"syscall"

	"golang.org/x/sys/windows"
)

// TryLock attempts to acquire the lock (non-blocking)
func (l *fileLockImpl) TryLock() (bool, error) {
	if l.file == nil {
		return false, fmt.Errorf("lock file not open")
	}

	handle := windows.Handle(l.file.Fd())
	overlapped := &windows.Overlapped{}

	err := windows.LockFileEx(
		handle,
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0,
		1,
		0,
		overlapped,
	)

	if err != nil {
		if err == windows.ERROR_LOCK_VIOLATION || err == syscall.ERROR_IO_PENDING {
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

	handle := windows.Handle(l.file.Fd())
	overlapped := &windows.Overlapped{}

	err := windows.UnlockFileEx(
		handle,
		0,
		1,
		0,
		overlapped,
	)

	if err != nil {
		return fmt.Errorf("failed to unlock: %w", err)
	}

	return nil
}
