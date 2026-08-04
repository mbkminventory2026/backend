package httpdelivery

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"permatatex-inventory/internal/backup"
	"permatatex-inventory/internal/model"
	"permatatex-inventory/internal/usecase"
	"permatatex-inventory/pkg/response"
)

type backupHTTPUseCase interface {
	Start(context.Context) (*model.BackupStartResponse, error)
	Status(context.Context) (*model.BackupStatusResponse, error)
	List(context.Context, model.ListQueryFilter) (*model.BackupListResponse, error)
	OpenDownload(context.Context, string) (*backup.OpenedBackup, error)
	RecordDownloaded(context.Context, backup.CompletedBackup)
}

type BackupHandler struct{ useCase backupHTTPUseCase }

func NewBackupHandler(useCase backupHTTPUseCase) (*BackupHandler, error) {
	if useCase == nil {
		return nil, errors.New("backup usecase is required")
	}
	return &BackupHandler{useCase: useCase}, nil
}

func (h *BackupHandler) RegisterRoutes(router gin.IRouter, authMiddleware gin.HandlerFunc) {
	group := router.Group("/api/v1/system/backups").Use(authMiddleware, RequireInternalUser())
	group.POST("", RequireAdminSistemUser(), RequirePermission(PermissionSystemBackupCreate), h.Start)
	group.GET("/status", RequirePermission(PermissionSystemBackupRead), h.Status)
	group.GET("", RequirePermission(PermissionSystemBackupRead), h.List)
	group.GET("/:backup_id/download", RequireAdminSistemUser(), RequirePermission(PermissionSystemBackupDownload), h.Download)
}

// Start godoc
// @Summary      Start encrypted backup
// @Description  Requires the ADMIN_SISTEM role and SYSTEM_BACKUP_CREATE permission. Starts one in-process encrypted backup and returns only after prerequisites and the advisory lock are accepted.
// @Tags         System Backups
// @Produce      json
// @Security     BearerAuth
// @Success      202  {object}  model.BackupStartSuccessDoc
// @Failure      401  {object}  response.BaseResponse
// @Failure      403  {object}  response.BaseResponse
// @Failure      409  {object}  response.BaseResponse
// @Failure      503  {object}  response.BaseResponse
// @Router       /api/v1/system/backups [post]
func (h *BackupHandler) Start(c *gin.Context) {
	result, err := h.useCase.Start(withAuditLogContext(c))
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrBackupRunning), errors.Is(err, usecase.ErrBackupLockHeld):
			AbortWithError(c, NewHTTPError(http.StatusConflict, "backup is already running", nil))
		default:
			AbortWithError(c, NewHTTPError(http.StatusServiceUnavailable, "backup service is unavailable", nil))
		}
		return
	}
	response.Success(c, http.StatusAccepted, "backup accepted", result)
}

// Status godoc
// @Summary      Get backup status
// @Description  Requires SYSTEM_BACKUP_READ permission. Returns process-local running state, optional last result, and the latest checksum-valid encrypted backup.
// @Tags         System Backups
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  model.BackupStatusSuccessDoc
// @Failure      401  {object}  response.BaseResponse
// @Failure      403  {object}  response.BaseResponse
// @Failure      503  {object}  response.BaseResponse
// @Router       /api/v1/system/backups/status [get]
func (h *BackupHandler) Status(c *gin.Context) {
	result, err := h.useCase.Status(c.Request.Context())
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusServiceUnavailable, "backup inventory is unavailable", nil))
		return
	}
	response.Success(c, http.StatusOK, "backup status retrieved", result)
}

// List godoc
// @Summary      List completed backups
// @Description  Requires SYSTEM_BACKUP_READ permission. Lists checksum-valid generated encrypted backup pairs, newest first.
// @Tags         System Backups
// @Produce      json
// @Security     BearerAuth
// @Param        page      query     int  false  "Page number"
// @Param        pageSize  query     int  false  "Page size"
// @Param        limit     query     int  false  "Limit fallback"
// @Success      200       {object}  model.BackupListSuccessDoc
// @Failure      400       {object}  response.BaseResponse
// @Failure      401       {object}  response.BaseResponse
// @Failure      403       {object}  response.BaseResponse
// @Failure      503       {object}  response.BaseResponse
// @Router       /api/v1/system/backups [get]
func (h *BackupHandler) List(c *gin.Context) {
	filter, err := parseListQuery(c, 20)
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid list query", nil))
		return
	}
	result, err := h.useCase.List(c.Request.Context(), filter)
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusServiceUnavailable, "backup inventory is unavailable", nil))
		return
	}
	setTotalCountHeader(c, result.Pagination.TotalItems)
	response.Success(c, http.StatusOK, "backup history retrieved", result)
}

// Download godoc
// @Summary      Download completed encrypted backup
// @Description  Requires the ADMIN_SISTEM role and SYSTEM_BACKUP_DOWNLOAD permission. Streams one private checksum-verified encrypted backup snapshot.
// @Tags         System Backups
// @Produce      application/octet-stream
// @Security     BearerAuth
// @Param        backup_id  path      string  true  "Server-generated backup ID"
// @Success      200        {file}    binary
// @Failure      401        {object}  response.BaseResponse
// @Failure      403        {object}  response.BaseResponse
// @Failure      404        {object}  response.BaseResponse
// @Router       /api/v1/system/backups/{backup_id}/download [get]
func (h *BackupHandler) Download(c *gin.Context) {
	id := c.Param("backup_id")
	if _, err := backup.ParseBackupID(id); err != nil {
		AbortWithError(c, NewHTTPError(http.StatusNotFound, "backup not found", nil))
		return
	}
	opened, err := h.useCase.OpenDownload(c.Request.Context(), id)
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusNotFound, "backup not found", nil))
		return
	}
	closed := false
	defer func() {
		if !closed {
			if err := opened.Close(); err != nil {
				slog.Default().Warn("backup download cleanup failed", slog.String("failure_class", "snapshot_cleanup"))
			}
		}
	}()

	fileName := id + ".tar.gz.gpg"
	headers := c.Writer.Header()
	headers.Set("Content-Type", "application/octet-stream")
	headers.Set("Content-Length", strconv.FormatInt(opened.Size, 10))
	headers.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))
	c.Status(http.StatusOK)
	written, copyErr := io.CopyN(c.Writer, opened.File, opened.Size)
	if copyErr != nil || written != opened.Size {
		slog.Default().Warn("backup download transfer failed", slog.String("failure_class", downloadFailureClass(c, copyErr, written, opened.Size)))
		return
	}
	if err := opened.Close(); err != nil {
		closed = true
		slog.Default().Warn("backup download cleanup failed", slog.String("failure_class", "snapshot_cleanup"))
		return
	}
	closed = true
	h.useCase.RecordDownloaded(withAuditLogContext(c), opened.CompletedBackup)
}

func downloadFailureClass(c *gin.Context, err error, written, expected int64) string {
	if c.Request.Context().Err() != nil || errors.Is(err, context.Canceled) {
		return "client_disconnected"
	}
	if written != expected {
		return "incomplete_transfer"
	}
	return "write_failure"
}
