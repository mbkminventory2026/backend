//go:build linux

package backup

import (
	"os"

	"golang.org/x/sys/unix"
)

type inventoryRoot struct{ fd int }

func openInventoryRoot(path string) (*inventoryRoot, error) {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	return &inventoryRoot{fd: fd}, nil
}

func (r *inventoryRoot) Open(name string) (*os.File, error) {
	fd, err := unix.Openat(r.fd, name, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), "backup-inventory"), nil
}

func (r *inventoryRoot) Close() error { return unix.Close(r.fd) }
