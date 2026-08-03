//go:build linux

package backup

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

type advisoryLock struct {
	file *os.File
}

func openAdvisoryLock(path string) (*advisoryLock, error) {
	fd, err := unix.Open(path, unix.O_RDWR|unix.O_CREAT|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), "backup-lock")
	var stat unix.Stat_t
	if err := unix.Fstat(fd, &stat); err != nil {
		return nil, closeFailedLock(file, err)
	}
	if stat.Mode&unix.S_IFMT != unix.S_IFREG {
		return nil, closeFailedLock(file, errors.New("invalid lock file"))
	}
	if err := unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
			return nil, closeFailedLock(file, ErrLockHeld)
		}
		return nil, closeFailedLock(file, err)
	}
	return &advisoryLock{file: file}, nil
}

func closeFailedLock(file *os.File, primary error) error {
	if err := file.Close(); err != nil {
		return errors.Join(primary, ErrCleanup)
	}
	return primary
}

func releaseAdvisoryLock(lock *advisoryLock) error {
	return errors.Join(unix.Flock(int(lock.file.Fd()), unix.LOCK_UN), lock.file.Close())
}
