//go:build linux

package backup

import "os"

func privatizeSnapshot(file *os.File) (string, error) {
	if err := os.Remove(file.Name()); err != nil {
		return "", err
	}
	return "", nil
}
