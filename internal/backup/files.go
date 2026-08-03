package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

type uploadStats struct{ Files, Bytes int64 }
type manifestEntry struct{ Path, Sum string }

func writeUploadsArchive(ctx context.Context, root, output string, now time.Time) (stats uploadStats, entries []manifestEntry, err error) {
	secureRoot, err := openUploadRoot(root)
	if err != nil {
		return stats, nil, err
	}
	f, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return stats, nil, errors.Join(err, secureRoot.Close())
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	processErr := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			return walkErr
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("symlink in uploads tree")
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel != "." && (filepath.IsAbs(rel) || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || strings.ContainsAny(rel, "\r\n")) {
			return errors.New("unsafe uploads entry")
		}
		name := "uploads/"
		if rel != "." {
			name += filepath.ToSlash(rel)
		}
		if info.IsDir() {
			if !strings.HasSuffix(name, "/") {
				name += "/"
			}
			return tw.WriteHeader(&tar.Header{Name: name, Typeflag: tar.TypeDir, Mode: 0o700, ModTime: now.UTC()})
		}
		if !info.Mode().IsRegular() {
			return errors.New("unsupported uploads entry")
		}
		in, err := secureRoot.Open(filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		opened, statErr := in.Stat()
		if statErr == nil && (!opened.Mode().IsRegular() || !os.SameFile(info, opened) || opened.Size() != info.Size()) {
			statErr = errors.New("uploads entry changed")
		}
		if statErr != nil {
			closeErr := in.Close()
			return errors.Join(statErr, closeErr)
		}
		header := &tar.Header{Name: name, Typeflag: tar.TypeReg, Mode: 0o600, Size: info.Size(), ModTime: now.UTC()}
		if err := tw.WriteHeader(header); err != nil {
			closeErr := in.Close()
			return errors.Join(err, closeErr)
		}
		hash := sha256.New()
		written, copyErr := copyContext(ctx, io.MultiWriter(tw, hash), in)
		after, afterErr := in.Stat()
		closeErr := in.Close()
		if copyErr != nil || afterErr != nil || closeErr != nil {
			return errors.Join(copyErr, afterErr, closeErr)
		}
		if written != info.Size() || !os.SameFile(info, after) || after.Size() != info.Size() || !after.ModTime().Equal(info.ModTime()) {
			return errors.New("uploads entry changed")
		}
		stats.Files++
		stats.Bytes += written
		entries = append(entries, manifestEntry{Path: filepath.ToSlash(rel), Sum: hex.EncodeToString(hash.Sum(nil))})
		return nil
	})
	closeErr := errors.Join(closeArchive(tw, gz, f), secureRoot.Close())
	if processErr != nil || closeErr != nil {
		return uploadStats{}, nil, errors.Join(processErr, closeErr)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return stats, entries, nil
}

func writeManifest(path string, entries []manifestEntry) error {
	var data strings.Builder
	for _, entry := range entries {
		fmt.Fprintf(&data, "%s  %s\n", entry.Sum, entry.Path)
	}
	return writeSyncedFile(path, []byte(data.String()))
}

func writeTarGz(ctx context.Context, output, topLevel string, files []string, now time.Time) error {
	f, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	processErr := tw.WriteHeader(&tar.Header{Name: topLevel + "/", Typeflag: tar.TypeDir, Mode: 0o700, ModTime: now.UTC()})
	for _, path := range files {
		if processErr != nil {
			break
		}
		if err := ctx.Err(); err != nil {
			processErr = err
			break
		}
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() {
			processErr = errors.Join(err, errors.New("invalid package component"))
			break
		}
		if err := tw.WriteHeader(&tar.Header{Name: topLevel + "/" + filepath.Base(path), Mode: 0o600, Size: info.Size(), Typeflag: tar.TypeReg, ModTime: now.UTC()}); err != nil {
			processErr = err
			break
		}
		in, err := os.Open(path)
		if err != nil {
			processErr = err
			break
		}
		written, copyErr := copyContext(ctx, tw, in)
		closeErr := in.Close()
		if copyErr != nil || closeErr != nil || written != info.Size() {
			processErr = errors.Join(copyErr, closeErr, componentSizeError(written, info.Size()))
			break
		}
	}
	return errors.Join(processErr, closeArchive(tw, gz, f))
}

func componentSizeError(got, want int64) error {
	if got != want {
		return errors.New("package component changed")
	}
	return nil
}

func copyContext(ctx context.Context, dst io.Writer, src io.Reader) (int64, error) {
	buf := make([]byte, 32*1024)
	var total int64
	for {
		if err := ctx.Err(); err != nil {
			return total, err
		}
		n, readErr := src.Read(buf)
		if n > 0 {
			written, writeErr := dst.Write(buf[:n])
			total += int64(written)
			if writeErr != nil {
				return total, writeErr
			}
			if written != n {
				return total, io.ErrShortWrite
			}
		}
		if readErr == io.EOF {
			return total, nil
		}
		if readErr != nil {
			return total, readErr
		}
	}
}

func closeArchive(tw *tar.Writer, gz *gzip.Writer, f *os.File) error {
	return errors.Join(tw.Close(), gz.Close(), f.Sync(), f.Close())
}
func writeSyncedFile(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	n, writeErr := f.Write(data)
	if writeErr == nil && n != len(data) {
		writeErr = io.ErrShortWrite
	}
	return errors.Join(writeErr, f.Sync(), f.Close())
}
func fileSHA256(ctx context.Context, path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	_, copyErr := copyContext(ctx, h, f)
	return hex.EncodeToString(h.Sum(nil)), errors.Join(copyErr, f.Close())
}
func writeChecksums(ctx context.Context, output string, files []string) error {
	var data strings.Builder
	for _, path := range files {
		sum, err := fileSHA256(ctx, path)
		if err != nil {
			return err
		}
		fmt.Fprintf(&data, "%s  %s\n", sum, filepath.Base(path))
	}
	return writeSyncedFile(output, []byte(data.String()))
}
func syncExistingFile(path string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	return errors.Join(f.Sync(), f.Close())
}
