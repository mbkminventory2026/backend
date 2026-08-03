//go:build !linux && !windows

package backup

import (
	"errors"
	"os"
	"sync"
)

// Non-Linux support is only for local compilation. Production locking is the
// descriptor-held flock implementation in lock_linux.go.
type advisoryLock struct {
	file *os.File
	path string
}

var localLocks = struct {
	sync.Mutex
	held map[string]bool
}{held: make(map[string]bool)}

func openAdvisoryLock(path string) (*advisoryLock, error) {
	localLocks.Lock()
	defer localLocks.Unlock()
	if localLocks.held[path] {
		return nil, ErrLockHeld
	}
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, err
	}
	localLocks.held[path] = true
	return &advisoryLock{file: file, path: path}, nil
}

func releaseAdvisoryLock(lock *advisoryLock) error {
	localLocks.Lock()
	delete(localLocks.held, lock.path)
	localLocks.Unlock()
	return errors.Join(lock.file.Close())
}
