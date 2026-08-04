package backup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestInventoryIncludesOnlyValidGeneratedPairsNewestFirst(t *testing.T) {
	destination := t.TempDir()
	validOld := writeInventoryPair(t, destination, "permatatex-backup-20260801T010203Z", []byte("old"), true)
	validNew := writeInventoryPair(t, destination, "permatatex-backup-20260803T010203Z", []byte("new"), true)
	writeInventoryPair(t, destination, "permatatex-baseline-20260804T010203Z", []byte("baseline"), true)
	writeInventoryPair(t, destination, "permatatex-backup-20260802T010203Z", []byte("bad"), false)
	if err := os.WriteFile(filepath.Join(destination, "permatatex-backup-20260802T020203Z.tar.gz.gpg"), []byte("orphan"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"permatatex-backup-20260802T030203Z.tar.gz.gpg.partial",
		"permatatex-backup-20260230T010203Z.tar.gz.gpg",
		"unfamiliar.tar.gz.gpg",
	} {
		if err := os.WriteFile(filepath.Join(destination, name), []byte("ignored"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	if runtime.GOOS != "windows" {
		target := filepath.Join(destination, validOld+packageSuffix)
		link := filepath.Join(destination, "permatatex-backup-20260805T010203Z.tar.gz.gpg")
		if err := os.Symlink(target, link); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256([]byte("old"))
		if err := os.WriteFile(link+".sha256", []byte(hex.EncodeToString(sum[:])+"  "+filepath.Base(link)+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	items, err := Inventory(context.Background(), destination)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].ID != validNew || items[1].ID != validOld {
		t.Fatalf("unexpected inventory: %#v", items)
	}
	for _, item := range items {
		if item.SHA256 == "" || item.Size <= 0 {
			t.Fatalf("unverified item returned: %#v", item)
		}
	}
}

func TestOpenCompletedRejectsMalformedIDsAndStreamsVerifiedDescriptor(t *testing.T) {
	destination := t.TempDir()
	id := writeInventoryPair(t, destination, "permatatex-backup-20260803T010203Z", []byte("encrypted-payload"), true)
	for _, invalid := range []string{"../secret", id + ".tar.gz.gpg", "permatatex-backup-20260230T010203Z"} {
		if _, err := OpenCompleted(context.Background(), destination, invalid); !errors.Is(err, ErrInventoryItem) {
			t.Fatalf("invalid ID %q accepted: %v", invalid, err)
		}
	}

	opened, err := OpenCompleted(context.Background(), destination, id)
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
	data, err := io.ReadAll(opened.File)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "encrypted-payload" {
		t.Fatalf("verified descriptor was not rewound: %q", data)
	}
	sum := sha256.Sum256(data)
	if opened.SHA256 != hex.EncodeToString(sum[:]) {
		t.Fatal("opened descriptor checksum mismatch")
	}
}

func TestOpenCompletedStreamsPrivateSnapshotAfterOriginalMutation(t *testing.T) {
	destination := t.TempDir()
	original := []byte("encrypted-original-payload")
	id := writeInventoryPair(t, destination, "permatatex-backup-20260803T010203Z", original, true)
	opened, err := OpenCompleted(context.Background(), destination, id)
	if err != nil {
		t.Fatal(err)
	}
	snapshotPath := opened.File.Name()
	mutated := []byte("encrypted-mutated!-payload")
	if len(mutated) != len(original) {
		t.Fatal("test mutation must preserve source size")
	}
	if err := os.WriteFile(filepath.Join(destination, id+packageSuffix), mutated, 0o600); err != nil {
		t.Fatal(err)
	}
	streamed, err := io.ReadAll(opened.File)
	if err != nil {
		t.Fatal(err)
	}
	if string(streamed) != string(original) {
		t.Fatalf("snapshot changed with original package: %q", streamed)
	}
	if err := opened.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(snapshotPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("snapshot pathname survived close: %v", err)
	}
}

func TestOpenCompletedRejectsMutationDuringSnapshotAndCleansFile(t *testing.T) {
	destination := t.TempDir()
	snapshotDir := t.TempDir()
	body := []byte("0123456789abcdef0123456789abcdef")
	id := writeInventoryPair(t, destination, "permatatex-backup-20260803T010203Z", body, true)
	packagePath := filepath.Join(destination, id+packageSuffix)
	originalCreate := createSnapshotFile
	originalCopy := copySnapshotData
	createSnapshotFile = func() (*os.File, error) {
		return os.CreateTemp(snapshotDir, "permatatex-backup-download-*")
	}
	copySnapshotData = func(_ context.Context, dst io.Writer, src io.Reader) (int64, error) {
		first, err := io.CopyN(dst, src, int64(len(body)/2))
		if err != nil {
			return first, err
		}
		mutated := append([]byte(nil), body...)
		copy(mutated[len(mutated)/2:], []byte("XXXXXXXXXXXXXXXX"))
		if err := os.WriteFile(packagePath, mutated, 0o600); err != nil {
			return first, err
		}
		rest, err := io.Copy(dst, src)
		return first + rest, err
	}
	defer func() {
		createSnapshotFile = originalCreate
		copySnapshotData = originalCopy
	}()
	if _, err := OpenCompleted(context.Background(), destination, id); !errors.Is(err, ErrInventoryItem) {
		t.Fatalf("mutated source accepted: %v", err)
	}
	assertNoSnapshotFiles(t, snapshotDir)
}

func TestOpenCompletedFailureCleansSnapshotAndKeepsStrictExclusions(t *testing.T) {
	destination := t.TempDir()
	snapshotDir := t.TempDir()
	invalidID := writeInventoryPair(t, destination, "permatatex-backup-20260803T010203Z", []byte("invalid"), false)
	orphanID := "permatatex-backup-20260804T010203Z"
	if err := os.WriteFile(filepath.Join(destination, orphanID+packageSuffix), []byte("orphan"), 0o600); err != nil {
		t.Fatal(err)
	}
	originalCreate := createSnapshotFile
	createSnapshotFile = func() (*os.File, error) {
		return os.CreateTemp(snapshotDir, "permatatex-backup-download-*")
	}
	defer func() { createSnapshotFile = originalCreate }()
	for _, id := range []string{invalidID, orphanID, "../malformed", "permatatex-backup-20260230T010203Z"} {
		if _, err := OpenCompleted(context.Background(), destination, id); !errors.Is(err, ErrInventoryItem) {
			t.Fatalf("excluded ID %q accepted: %v", id, err)
		}
	}
	if runtime.GOOS != "windows" {
		targetID := writeInventoryPair(t, destination, "permatatex-backup-20260805T010203Z", []byte("target"), true)
		linkID := "permatatex-backup-20260806T010203Z"
		linkName := linkID + packageSuffix
		if err := os.Symlink(filepath.Join(destination, targetID+packageSuffix), filepath.Join(destination, linkName)); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256([]byte("target"))
		if err := os.WriteFile(filepath.Join(destination, linkName+".sha256"), []byte(hex.EncodeToString(sum[:])+"  "+linkName+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := OpenCompleted(context.Background(), destination, linkID); !errors.Is(err, ErrInventoryItem) {
			t.Fatalf("symlink package accepted: %v", err)
		}
	}
	assertNoSnapshotFiles(t, snapshotDir)
}

func assertNoSnapshotFiles(t *testing.T, directory string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(directory, "permatatex-backup-download-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("snapshot files were not cleaned: %v", matches)
	}
}

func writeInventoryPair(t *testing.T, destination, id string, body []byte, valid bool) string {
	t.Helper()
	name := id + packageSuffix
	if err := os.WriteFile(filepath.Join(destination, name), body, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	checksum := hex.EncodeToString(sum[:])
	if !valid {
		checksum = "0000000000000000000000000000000000000000000000000000000000000000"
	}
	if err := os.WriteFile(filepath.Join(destination, name+".sha256"), []byte(checksum+"  "+name+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestJobUsesCallerGeneratedIDAndSignalsAcceptedAfterLock(t *testing.T) {
	cfg := testConfig(t)
	if err := os.WriteFile(filepath.Join(cfg.UploadsSource, "upload"), []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	id := "permatatex-backup-20260803T111213Z"
	job := NewJob(cfg, nil)
	job.ID = id
	job.Commands = fakeCommands{}
	job.Now = func() time.Time { return time.Date(2026, 8, 3, 11, 12, 14, 0, time.UTC) }
	accepted := false
	job.Accepted = func(event Accepted) {
		accepted = true
		if event.BackupID != id {
			t.Fatalf("accepted ID=%q", event.BackupID)
		}
		if lock, err := acquireLock(filepath.Join(cfg.Destination, lockName), "other", event.Started); !errors.Is(err, ErrLockHeld) {
			if err == nil {
				if releaseErr := releaseLock(lock); releaseErr != nil {
					t.Fatalf("release unexpected lock: %v", releaseErr)
				}
			}
			t.Fatalf("accepted fired before advisory lock: %v", err)
		}
	}
	if err := job.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !accepted {
		t.Fatal("accepted hook did not fire")
	}
	if _, err := os.Stat(filepath.Join(cfg.Destination, id+packageSuffix)); err != nil {
		t.Fatal("caller-generated ID was not used")
	}
}

func TestJobDoesNotSignalAcceptedWhenPrerequisiteValidationFails(t *testing.T) {
	cfg := testConfig(t)
	job := NewJob(cfg, nil)
	job.ID = "permatatex-backup-20260803T111213Z"
	job.Commands = fakeCommands{fail: "pg_dump"}
	accepted := false
	job.Accepted = func(Accepted) { accepted = true }
	if err := job.Run(context.Background()); !errors.Is(err, ErrPrereq) {
		t.Fatalf("unexpected error: %v", err)
	}
	if accepted {
		t.Fatal("accepted signal fired after failed prerequisite validation")
	}
}
