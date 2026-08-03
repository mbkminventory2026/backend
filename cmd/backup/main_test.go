package main

import (
	"errors"
	"testing"

	"permatatex-inventory/internal/backup"
)

func TestCleanupFailureClassIsUnambiguous(t *testing.T) {
	err := errors.Join(errors.New("earlier failure"), backup.ErrCleanup)
	if code := backup.ExitCode(err); code != 6 {
		t.Fatalf("cleanup exit code=%d", code)
	}
	if class := failureClass(err, 6); class != "cleanup" {
		t.Fatalf("cleanup failure class=%q", class)
	}
}
