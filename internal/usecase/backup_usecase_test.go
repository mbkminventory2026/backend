package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"permatatex-inventory/internal/backup"
	"permatatex-inventory/internal/model"
)

func TestBackupHistoryPaginationNewestFirstAndDownloadAudit(t *testing.T) {
	destination := t.TempDir()
	for _, stamp := range []string{"20260801T010203Z", "20260802T010203Z", "20260803T010203Z"} {
		writeUseCaseBackupPair(t, destination, stamp)
	}
	audit := &recordedAudit{}
	manager, err := NewBackupJobManager(func(string, func(backup.Accepted)) BackupJob {
		return backupJobFunc(func(context.Context) error { return nil })
	}, destination, audit, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewBackupUseCase(manager, destination)
	if err != nil {
		t.Fatal(err)
	}

	page, err := service.List(context.Background(), model.ListQueryFilter{Page: 2, Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].BackupID != "permatatex-backup-20260801T010203Z" || page.Pagination.TotalItems != 3 {
		t.Fatalf("unexpected second page: %#v", page)
	}
	status, err := service.Status(context.Background())
	if err != nil || status.Latest == nil || status.Latest.BackupID != "permatatex-backup-20260803T010203Z" {
		t.Fatalf("latest inventory mismatch: %#v %v", status, err)
	}

	opened, err := service.OpenDownload(context.Background(), status.Latest.BackupID)
	if err != nil {
		t.Fatal(err)
	}
	ctx := WithAuditLogContext(context.Background(), AuditLogContext{ActorUserID: int32Pointer(9), ActorRole: "ADMIN_SISTEM", Method: "GET", Route: "/api/v1/system/backups/:backup_id/download"})
	service.RecordDownloaded(ctx, opened.CompletedBackup)
	_ = opened.Close()
	waitFor(t, func() bool { return strings.Contains(strings.Join(audit.events(), ","), "backup_downloaded") })

	recordsJSON, err := json.Marshal(audit.snapshot())
	if err != nil {
		t.Fatal(err)
	}
	text := string(recordsJSON)
	if !strings.Contains(text, "backup_downloaded") {
		t.Fatal("download lifecycle event was not audited")
	}
	for _, forbidden := range []string{destination, "postgres://", "password", "fingerprint", "public.asc", "upload"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("audit metadata leaked forbidden value %q", forbidden)
		}
	}
}

func writeUseCaseBackupPair(t *testing.T, destination, stamp string) {
	t.Helper()
	id := "permatatex-backup-" + stamp
	name := id + ".tar.gz.gpg"
	body := []byte(stamp)
	if err := os.WriteFile(filepath.Join(destination, name), body, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	if err := os.WriteFile(filepath.Join(destination, name+".sha256"), []byte(hex.EncodeToString(sum[:])+"  "+name+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func int32Pointer(value int32) *int32 { return &value }

func TestBackupStatusUsesOnlyIdleRunningAcrossNewManager(t *testing.T) {
	manager := testBackupManager(t, func(id string, accepted func(backup.Accepted)) BackupJob {
		return backupJobFunc(func(context.Context) error {
			accepted(backup.Accepted{BackupID: id, Started: time.Now().UTC()})
			return nil
		})
	}, nil)
	if status := manager.Status(); status.State != "idle" {
		t.Fatalf("new manager state=%s", status.State)
	}
}
