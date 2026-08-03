//go:build linux

package backup

import (
	"os"

	"golang.org/x/sys/unix"
)

type secureUploadRoot struct {
	fd int
}

func openUploadRoot(root string) (*secureUploadRoot, error) {
	fd, err := unix.Open(root, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	return &secureUploadRoot{fd: fd}, nil
}

func (r *secureUploadRoot) Open(relative string) (*os.File, error) {
	fd, err := unix.Openat2(r.fd, relative, &unix.OpenHow{
		Flags:   unix.O_RDONLY | unix.O_CLOEXEC | unix.O_NOFOLLOW,
		Resolve: unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS,
	})
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), "upload"), nil
}

func (r *secureUploadRoot) Close() error {
	return unix.Close(r.fd)
}
