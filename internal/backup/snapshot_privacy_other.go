//go:build !linux

package backup

import "os"

func privatizeSnapshot(file *os.File) (string, error) {
	return file.Name(), nil
}
