package httpdelivery

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"permatatex-inventory/internal/backup"
	"permatatex-inventory/internal/model"
	"permatatex-inventory/internal/usecase"
)

const handlerTestBackupID = "permatatex-backup-20260803T010203Z"

type fakeBackupHTTPUseCase struct {
	startErr    error
	startCalls  int
	downloads   int
	downloaded  int
	downloadDir string
	openedFile  *os.File
}

func (f *fakeBackupHTTPUseCase) Start(context.Context) (*model.BackupStartResponse, error) {
	f.startCalls++
	if f.startErr != nil {
		return nil, f.startErr
	}
	return &model.BackupStartResponse{BackupID: handlerTestBackupID, State: "running"}, nil
}

func (f *fakeBackupHTTPUseCase) Status(context.Context) (*model.BackupStatusResponse, error) {
	return &model.BackupStatusResponse{State: "idle"}, nil
}

func (f *fakeBackupHTTPUseCase) List(context.Context, model.ListQueryFilter) (*model.BackupListResponse, error) {
	return &model.BackupListResponse{Items: []model.BackupSummary{}, Pagination: model.PaginationMeta{Page: 1, Limit: 20}}, nil
}

func (f *fakeBackupHTTPUseCase) OpenDownload(_ context.Context, id string) (*backup.OpenedBackup, error) {
	f.downloads++
	if _, err := backup.ParseBackupID(id); err != nil || id != handlerTestBackupID {
		return nil, usecase.ErrBackupNotFound
	}
	file, err := os.Open(f.downloadDir)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	payload, err := os.ReadFile(f.downloadDir)
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	sum := sha256.Sum256(payload)
	f.openedFile = file
	return &backup.OpenedBackup{CompletedBackup: backup.CompletedBackup{
		ID: handlerTestBackupID, CompletedAt: time.Date(2026, 8, 3, 1, 2, 3, 0, time.UTC),
		Size: info.Size(), SHA256: hex.EncodeToString(sum[:]),
	}, File: file}, nil
}

func (f *fakeBackupHTTPUseCase) RecordDownloaded(context.Context, backup.CompletedBackup) {
	f.downloaded++
}

func TestBackupRoutesEnforceAdminAndManagerPermissions(t *testing.T) {
	payloadPath := tempDownloadFile(t, []byte("encrypted-stream"))
	tests := []struct {
		name        string
		role        string
		permissions []string
		method      string
		path        string
		wantStatus  int
	}{
		{"admin create", "ADMIN_SISTEM", []string{PermissionSystemBackupCreate}, http.MethodPost, "/api/v1/system/backups", http.StatusAccepted},
		{"all access create", "SUPER_ADMIN", []string{PermissionAllAccess}, http.MethodPost, "/api/v1/system/backups", http.StatusAccepted},
		{"manager create forbidden", "MANAGER", []string{PermissionSystemBackupRead}, http.MethodPost, "/api/v1/system/backups", http.StatusForbidden},
		{"manager status", "MANAGER", []string{PermissionSystemBackupRead}, http.MethodGet, "/api/v1/system/backups/status", http.StatusOK},
		{"manager history", "MANAGER", []string{PermissionSystemBackupRead}, http.MethodGet, "/api/v1/system/backups", http.StatusOK},
		{"admin download", "ADMIN_SISTEM", []string{PermissionSystemBackupDownload}, http.MethodGet, "/api/v1/system/backups/" + handlerTestBackupID + "/download", http.StatusOK},
		{"manager download forbidden", "MANAGER", []string{PermissionSystemBackupRead}, http.MethodGet, "/api/v1/system/backups/" + handlerTestBackupID + "/download", http.StatusForbidden},
		{"extension rejected", "ADMIN_SISTEM", []string{PermissionSystemBackupDownload}, http.MethodGet, "/api/v1/system/backups/" + handlerTestBackupID + ".tar.gz.gpg/download", http.StatusNotFound},
		{"path token rejected", "ADMIN_SISTEM", []string{PermissionSystemBackupDownload}, http.MethodGet, "/api/v1/system/backups/%2e%2e/download", http.StatusNotFound},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &fakeBackupHTTPUseCase{downloadDir: payloadPath}
			router := backupTestRouter(t, service, test.role, test.permissions)
			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(test.method, test.path, nil)
			router.ServeHTTP(recorder, req)
			if recorder.Code != test.wantStatus {
				t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
			}
			if test.name == "admin download" {
				if recorder.Body.String() != "encrypted-stream" || service.downloaded != 1 {
					t.Fatal("verified descriptor was not streamed or audited")
				}
				if recorder.Header().Get("Content-Type") != "application/octet-stream" {
					t.Fatal("unexpected download content type")
				}
				if service.openedFile == nil {
					t.Fatal("download descriptor was not opened")
				}
				if _, err := service.openedFile.Stat(); err == nil {
					t.Fatal("download descriptor remained open after success")
				}
			}
		})
	}
}

func TestBackupDownloadTransferFailureAndDisconnectSkipAuditAndCloseDescriptor(t *testing.T) {
	payloadPath := tempDownloadFile(t, []byte("encrypted-stream"))
	tests := []struct {
		name       string
		writeError error
		cancel     bool
	}{
		{name: "writer failure", writeError: errors.New("writer failed")},
		{name: "client disconnect", writeError: context.Canceled, cancel: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &fakeBackupHTTPUseCase{downloadDir: payloadPath}
			router := backupTestRouter(t, service, "ADMIN_SISTEM", []string{PermissionSystemBackupDownload})
			ctx := context.Background()
			var cancel context.CancelFunc
			if test.cancel {
				ctx, cancel = context.WithCancel(ctx)
				defer cancel()
			}
			writer := &failingHTTPWriter{
				header:     make(http.Header),
				writeLimit: 4,
				writeError: test.writeError,
				cancel:     cancel,
			}
			req := httptest.NewRequest(http.MethodGet, "/api/v1/system/backups/"+handlerTestBackupID+"/download", nil).WithContext(ctx)
			router.ServeHTTP(writer, req)
			if service.downloaded != 0 {
				t.Fatal("partial download emitted completed audit")
			}
			if service.openedFile == nil {
				t.Fatal("download descriptor was not opened")
			}
			if _, err := service.openedFile.Stat(); err == nil {
				t.Fatal("download descriptor remained open after transfer failure")
			}
		})
	}
}

type failingHTTPWriter struct {
	header     http.Header
	status     int
	written    int
	writeLimit int
	writeError error
	cancel     context.CancelFunc
}

func (w *failingHTTPWriter) Header() http.Header { return w.header }

func (w *failingHTTPWriter) WriteHeader(status int) { w.status = status }

func (w *failingHTTPWriter) Write(p []byte) (int, error) {
	if w.cancel != nil {
		w.cancel()
	}
	remaining := w.writeLimit - w.written
	if remaining < 0 {
		remaining = 0
	}
	if remaining > len(p) {
		remaining = len(p)
	}
	w.written += remaining
	return remaining, w.writeError
}

func TestBackupStartConfigurationFailureIsSafe503(t *testing.T) {
	service := &fakeBackupHTTPUseCase{startErr: usecase.ErrBackupUnavailable}
	router := backupTestRouter(t, service, "ADMIN_SISTEM", []string{PermissionSystemBackupCreate})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/system/backups", nil))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	for _, forbidden := range []string{"postgres://", "fingerprint", "BACKUP_", "\\", "/backups"} {
		if containsString(recorder.Body.String(), forbidden) {
			t.Fatalf("unsafe detail leaked: %q", forbidden)
		}
	}
}

func backupTestRouter(t *testing.T, service backupHTTPUseCase, role string, permissions []string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	handler, err := NewBackupHandler(service)
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.Use(ErrorHandlerMiddleware())
	auth := func(c *gin.Context) {
		c.Set(authorizationPayloadKey, jwt.MapClaims{
			"user_id":     float64(7),
			"role_name":   role,
			"permissions": permissions,
		})
		c.Next()
	}
	handler.RegisterRoutes(router, auth)
	return router
}

func tempDownloadFile(t *testing.T, payload []byte) string {
	t.Helper()
	path := t.TempDir() + string(os.PathSeparator) + "encrypted.gpg"
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func containsString(value, needle string) bool {
	for i := 0; i+len(needle) <= len(value); i++ {
		if value[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
