//go:build windows

package backup

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

type advisoryLock struct {
	file       *os.File
	overlapped windows.Overlapped
}

func openAdvisoryLock(path string) (*advisoryLock, error) {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, err
	}
	info, statErr := file.Stat()
	if statErr != nil {
		return nil, closeFailedLock(file, statErr)
	}
	if !info.Mode().IsRegular() {
		return nil, closeFailedLock(file, errors.New("invalid lock file"))
	}
	lock := &advisoryLock{file: file}
	err = windows.LockFileEx(windows.Handle(file.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, &lock.overlapped)
	if err != nil {
		if errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
			return nil, closeFailedLock(file, ErrLockHeld)
		}
		return nil, closeFailedLock(file, err)
	}
	return lock, nil
}

func closeFailedLock(file *os.File, primary error) error {
	if err := file.Close(); err != nil {
		return errors.Join(primary, ErrCleanup)
	}
	return primary
}

func releaseAdvisoryLock(lock *advisoryLock) error {
	return errors.Join(windows.UnlockFileEx(windows.Handle(lock.file.Fd()), 0, 1, 0, &lock.overlapped), lock.file.Close())
}
