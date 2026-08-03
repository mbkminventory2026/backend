//go:build !linux

package backup

import (
	"os"
	"path/filepath"
)

// This fallback exists for local non-Linux compilation and tests. Production
// uses secureopen_linux.go inside the backup container.
type secureUploadRoot struct {
	root string
}

func openUploadRoot(root string) (*secureUploadRoot, error) {
	return &secureUploadRoot{root: root}, nil
}

func (r *secureUploadRoot) Open(relative string) (*os.File, error) {
	return os.Open(filepath.Join(r.root, filepath.FromSlash(relative)))
}

func (r *secureUploadRoot) Close() error {
	return nil
}
