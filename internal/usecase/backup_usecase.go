package usecase

import (
	"context"
	"errors"
	"time"

	"permatatex-inventory/internal/backup"
	"permatatex-inventory/internal/model"
)

type BackupUseCase struct {
	manager     *BackupJobManager
	destination string
}

func NewBackupUseCase(manager *BackupJobManager, destination string) (*BackupUseCase, error) {
	if manager == nil {
		return nil, errors.New("backup job manager is required")
	}
	return &BackupUseCase{manager: manager, destination: destination}, nil
}

func (u *BackupUseCase) Start(ctx context.Context) (*model.BackupStartResponse, error) {
	return u.manager.Start(ctx)
}

func (u *BackupUseCase) Status(ctx context.Context) (*model.BackupStatusResponse, error) {
	status := u.manager.Status()
	items, err := backup.Inventory(ctx, u.destination)
	if err != nil {
		return nil, ErrBackupUnavailable
	}
	if len(items) > 0 {
		latest := completedBackupSummary(items[0])
		status.Latest = &latest
	}
	return &status, nil
}

func (u *BackupUseCase) List(ctx context.Context, filter model.ListQueryFilter) (*model.BackupListResponse, error) {
	items, err := backup.Inventory(ctx, u.destination)
	if err != nil {
		return nil, ErrBackupUnavailable
	}
	page := filter.Page
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 500 {
		limit = 500
	}
	if page <= 0 {
		page = 1
	}
	offset := int64(page-1) * int64(limit)
	end := offset + int64(limit)
	if offset > int64(len(items)) {
		offset = int64(len(items))
	}
	if end > int64(len(items)) {
		end = int64(len(items))
	}
	result := make([]model.BackupSummary, 0, end-offset)
	for _, item := range items[offset:end] {
		result = append(result, completedBackupSummary(item))
	}
	return &model.BackupListResponse{
		Items:      result,
		Pagination: buildPagination(int64(len(items)), page, limit),
	}, nil
}

func (u *BackupUseCase) OpenDownload(ctx context.Context, id string) (*backup.OpenedBackup, error) {
	opened, err := backup.OpenCompleted(ctx, u.destination, id)
	if err != nil {
		return nil, ErrBackupNotFound
	}
	return opened, nil
}

func (u *BackupUseCase) RecordDownloaded(ctx context.Context, item backup.CompletedBackup) {
	size := item.Size
	u.manager.recordAuditAsync(context.WithoutCancel(ctx), "backup_downloaded", item.ID, "completed", time.Now().UTC(), &model.BackupLastResult{
		BackupID:      item.ID,
		State:         "completed",
		EncryptedSize: &size,
		SHA256:        item.SHA256,
	})
}

func (u *BackupUseCase) Shutdown(ctx context.Context) error {
	return u.manager.Shutdown(ctx)
}

func completedBackupSummary(item backup.CompletedBackup) model.BackupSummary {
	return model.BackupSummary{
		BackupID:          item.ID,
		CompletedAt:       item.CompletedAt.UTC().Format(time.RFC3339),
		EncryptedSize:     item.Size,
		SHA256:            item.SHA256,
		DownloadAvailable: true,
	}
}
