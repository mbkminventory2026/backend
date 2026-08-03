//go:build linux

package backup

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSecureUploadOpenRejectsSymlinkReplacement(t *testing.T) {
	root := t.TempDir()
	original := filepath.Join(root, "upload")
	saved := filepath.Join(root, "saved")
	if err := os.WriteFile(original, []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	secureRoot, err := openUploadRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer secureRoot.Close()
	if err := os.Rename(original, saved); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("saved", original); err != nil {
		t.Fatal(err)
	}
	if file, err := secureRoot.Open("upload"); err == nil {
		file.Close()
		t.Fatal("secure upload open followed a replacement symlink")
	}
}

func TestAdvisoryLockReleaseDoesNotUnlinkReplacementPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), lockName)
	lock, err := acquireLock(path, "first", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	replacement := []byte("replacement metadata\n")
	if err := os.WriteFile(path, replacement, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := releaseLock(lock); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal("replacement pathname was removed")
	}
	if string(content) != string(replacement) {
		t.Fatal("replacement pathname was modified")
	}
}
