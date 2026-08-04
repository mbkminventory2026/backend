package backup

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"sync"
	"time"
)

const (
	backupIDLayout = "20060102T150405Z"
	packageSuffix  = ".tar.gz.gpg"
)

var (
	backupIDPattern    = regexp.MustCompile(`^permatatex-backup-(\d{8}T\d{6}Z)$`)
	checksumPattern    = regexp.MustCompile(`^[0-9a-f]{64}$`)
	ErrInventoryItem   = errors.New("backup inventory item not found")
	createSnapshotFile = func() (*os.File, error) {
		return os.CreateTemp("", "permatatex-backup-download-*")
	}
	copySnapshotData = copyContext
)

// CompletedBackup contains only safe facts independently verified from an
// encrypted package and its checksum sidecar.
type CompletedBackup struct {
	ID          string
	CompletedAt time.Time
	Size        int64
	SHA256      string
}

// OpenedBackup keeps the exact verified descriptor that callers must stream.
type OpenedBackup struct {
	CompletedBackup
	File        *os.File
	cleanupPath string
	closeOnce   sync.Once
	closeErr    error
}

// Close releases the private encrypted snapshot and removes its pathname on
// platforms that cannot unlink an open file.
func (o *OpenedBackup) Close() error {
	if o == nil {
		return nil
	}
	o.closeOnce.Do(func() {
		if o.File != nil {
			o.closeErr = o.File.Close()
		}
		if o.cleanupPath != "" {
			if err := os.Remove(o.cleanupPath); err != nil && !errors.Is(err, os.ErrNotExist) {
				o.closeErr = errors.Join(o.closeErr, err)
			}
		}
	})
	return o.closeErr
}

type completedSource struct {
	file        *os.File
	root        *inventoryRoot
	info        os.FileInfo
	id          string
	completedAt time.Time
	expected    string
}

func (s *completedSource) Close() error {
	return errors.Join(s.file.Close(), s.root.Close())
}

func ParseBackupID(id string) (time.Time, error) {
	matches := backupIDPattern.FindStringSubmatch(id)
	if matches == nil {
		return time.Time{}, ErrInventoryItem
	}
	parsed, err := time.Parse(backupIDLayout, matches[1])
	if err != nil || parsed.Location() != time.UTC || BackupID(parsed) != id {
		return time.Time{}, ErrInventoryItem
	}
	return parsed.UTC(), nil
}

func PackageName(id string) (string, error) {
	if _, err := ParseBackupID(id); err != nil {
		return "", err
	}
	return id + packageSuffix, nil
}

// Inventory returns only complete, checksum-valid generated backups, newest
// first. An unavailable destination is treated as an empty inventory.
func Inventory(ctx context.Context, destination string) ([]CompletedBackup, error) {
	if !safeInventoryDestination(destination) {
		return []CompletedBackup{}, nil
	}
	entries, err := os.ReadDir(destination)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, os.ErrPermission) {
			return []CompletedBackup{}, nil
		}
		return nil, err
	}
	items := make([]CompletedBackup, 0)
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		matches := backupName.FindStringSubmatch(entry.Name())
		if matches == nil {
			continue
		}
		id := entry.Name()[:len(entry.Name())-len(packageSuffix)]
		item, err := inspectCompleted(ctx, destination, id)
		if err != nil {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].CompletedAt.Equal(items[j].CompletedAt) {
			return items[i].ID > items[j].ID
		}
		return items[i].CompletedAt.After(items[j].CompletedAt)
	})
	return items, nil
}

// OpenCompleted validates a pair while copying encrypted bytes into a private
// mode-0600 snapshot. Only the verified snapshot descriptor is returned.
func OpenCompleted(ctx context.Context, destination, id string) (*OpenedBackup, error) {
	source, err := openCompletedSource(ctx, destination, id)
	if err != nil {
		return nil, ErrInventoryItem
	}
	defer source.Close()

	snapshot, err := createSnapshotFile()
	if err != nil {
		return nil, ErrInventoryItem
	}
	if err := snapshot.Chmod(0o600); err != nil {
		_ = cleanupSnapshot(snapshot)
		return nil, ErrInventoryItem
	}
	keepSnapshot := false
	defer func() {
		if !keepSnapshot {
			_ = cleanupSnapshot(snapshot)
		}
	}()
	hash := sha256.New()
	written, err := copySnapshotData(ctx, io.MultiWriter(snapshot, hash), source.file)
	if err != nil || written != source.info.Size() {
		return nil, ErrInventoryItem
	}
	actual := hex.EncodeToString(hash.Sum(nil))
	if actual != source.expected {
		return nil, ErrInventoryItem
	}
	after, err := source.file.Stat()
	if err != nil || !sameCompletedFile(source.info, after) {
		return nil, ErrInventoryItem
	}
	if err := snapshot.Sync(); err != nil {
		return nil, ErrInventoryItem
	}
	if _, err := snapshot.Seek(0, io.SeekStart); err != nil {
		return nil, ErrInventoryItem
	}
	cleanupPath, err := privatizeSnapshot(snapshot)
	if err != nil {
		return nil, ErrInventoryItem
	}
	keepSnapshot = true
	return &OpenedBackup{
		CompletedBackup: CompletedBackup{ID: source.id, CompletedAt: source.completedAt, Size: written, SHA256: actual},
		File:            snapshot,
		cleanupPath:     cleanupPath,
	}, nil
}

func inspectCompleted(ctx context.Context, destination, id string) (CompletedBackup, error) {
	source, err := openCompletedSource(ctx, destination, id)
	if err != nil {
		return CompletedBackup{}, ErrInventoryItem
	}
	defer source.Close()

	hash := sha256.New()
	if _, err := copyContext(ctx, hash, source.file); err != nil {
		return CompletedBackup{}, ErrInventoryItem
	}
	actual := hex.EncodeToString(hash.Sum(nil))
	if actual != source.expected {
		return CompletedBackup{}, ErrInventoryItem
	}
	after, err := source.file.Stat()
	if err != nil || !sameCompletedFile(source.info, after) {
		return CompletedBackup{}, ErrInventoryItem
	}
	return CompletedBackup{ID: source.id, CompletedAt: source.completedAt, Size: source.info.Size(), SHA256: actual}, nil
}

func openCompletedSource(ctx context.Context, destination, id string) (*completedSource, error) {
	completedAt, err := ParseBackupID(id)
	if err != nil || !safeInventoryDestination(destination) || ctx.Err() != nil {
		return nil, ErrInventoryItem
	}
	name := id + packageSuffix
	root, err := openInventoryRoot(destination)
	if err != nil {
		return nil, ErrInventoryItem
	}
	packageFile, err := root.Open(name)
	if err != nil {
		_ = root.Close()
		return nil, ErrInventoryItem
	}
	fail := func() (*completedSource, error) {
		_ = packageFile.Close()
		_ = root.Close()
		return nil, ErrInventoryItem
	}
	packageInfo, err := packageFile.Stat()
	if err != nil || !packageInfo.Mode().IsRegular() {
		return fail()
	}

	sidecar, err := root.Open(name + ".sha256")
	if err != nil {
		return fail()
	}
	sidecarInfo, err := sidecar.Stat()
	if err != nil || !sidecarInfo.Mode().IsRegular() {
		_ = sidecar.Close()
		return fail()
	}
	expected, sidecarErr := readExactSidecar(sidecar, name)
	closeErr := sidecar.Close()
	if sidecarErr != nil || closeErr != nil {
		return fail()
	}
	return &completedSource{file: packageFile, root: root, info: packageInfo, id: id, completedAt: completedAt, expected: expected}, nil
}

func sameCompletedFile(before, after os.FileInfo) bool {
	return after.Mode().IsRegular() && after.Size() == before.Size() && after.ModTime().Equal(before.ModTime()) && os.SameFile(before, after)
}

func cleanupSnapshot(file *os.File) error {
	if file == nil {
		return nil
	}
	path := file.Name()
	return errors.Join(file.Close(), removeSnapshotPath(path))
}

func removeSnapshotPath(path string) error {
	err := os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func readExactSidecar(file *os.File, packageName string) (string, error) {
	reader := bufio.NewReader(io.LimitReader(file, 256))
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	expectedSuffix := "  " + packageName + "\n"
	if len(data) != 64+len(expectedSuffix) || string(data[64:]) != expectedSuffix {
		return "", ErrInventoryItem
	}
	checksum := string(data[:64])
	if !checksumPattern.MatchString(checksum) {
		return "", ErrInventoryItem
	}
	return checksum, nil
}

func safeInventoryDestination(destination string) bool {
	if destination == "" || !filepath.IsAbs(destination) {
		return false
	}
	info, err := os.Lstat(destination)
	return err == nil && info.IsDir() && info.Mode()&os.ModeSymlink == 0
}
