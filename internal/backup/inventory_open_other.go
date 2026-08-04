//go:build !linux

package backup

import (
	"errors"
	"os"
)

type inventoryRoot struct{ root *os.Root }

func openInventoryRoot(path string) (*inventoryRoot, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("invalid inventory root")
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, err
	}
	return &inventoryRoot{root: root}, nil
}

func (r *inventoryRoot) Open(name string) (*os.File, error) {
	before, err := r.root.Lstat(name)
	if err != nil || !before.Mode().IsRegular() || before.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("invalid inventory file")
	}
	file, err := r.root.Open(name)
	if err != nil {
		return nil, err
	}
	after, err := file.Stat()
	if err != nil || !after.Mode().IsRegular() || !os.SameFile(before, after) {
		_ = file.Close()
		return nil, errors.New("inventory file changed")
	}
	return file, nil
}

func (r *inventoryRoot) Close() error { return r.root.Close() }
