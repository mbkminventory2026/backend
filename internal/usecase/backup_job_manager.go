package usecase

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"permatatex-inventory/internal/backup"
	"permatatex-inventory/internal/model"
)

var (
	ErrBackupRunning     = errors.New("backup is already running")
	ErrBackupLockHeld    = errors.New("backup lock is already held")
	ErrBackupUnavailable = errors.New("backup service is unavailable")
	ErrBackupNotFound    = errors.New("backup not found")
)

const (
	backupStateIdle    = "idle"
	backupStateRunning = "running"
	backupAuditTimeout = 2 * time.Second
)

type BackupJob interface {
	Run(context.Context) error
}

type BackupJobFactory func(id string, accepted func(backup.Accepted)) BackupJob

type backupAuditRecorder interface {
	Record(context.Context, model.AuditLogRecordRequest) error
}

type BackupJobManager struct {
	mu          sync.Mutex
	wg          sync.WaitGroup
	factory     BackupJobFactory
	audit       backupAuditRecorder
	logger      *slog.Logger
	now         func() time.Time
	destination string
	stopping    bool
	active      bool
	running     bool
	currentID   string
	startedAt   time.Time
	cancel      context.CancelFunc
	lastResult  *model.BackupLastResult
}

func NewBackupJobManager(factory BackupJobFactory, destination string, audit backupAuditRecorder, logger *slog.Logger) (*BackupJobManager, error) {
	if factory == nil {
		return nil, errors.New("backup job factory is required")
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &BackupJobManager{factory: factory, destination: destination, audit: audit, logger: logger, now: time.Now}, nil
}

func (m *BackupJobManager) Start(ctx context.Context) (*model.BackupStartResponse, error) {
	auditCtx := context.WithoutCancel(ctx)
	now := m.now().UTC()

	m.mu.Lock()
	if m.stopping {
		m.mu.Unlock()
		m.recordAuditAsync(auditCtx, "backup_requested", "", "requested", now, nil)
		return nil, ErrBackupUnavailable
	}
	if m.active {
		activeID := m.currentID
		m.mu.Unlock()
		m.recordAuditAsync(auditCtx, "backup_requested", "", "requested", now, nil)
		m.recordAuditAsync(auditCtx, "backup_rejected_running", activeID, "running", now, nil)
		return nil, ErrBackupRunning
	}

	id := backup.BackupID(now)
	jobCtx, cancel := context.WithCancel(auditCtx)
	acceptedCh := make(chan struct{})
	returnedCh := make(chan error, 1)
	m.active = true
	m.running = false
	m.currentID = id
	m.startedAt = now
	m.cancel = cancel
	m.wg.Add(1)
	m.mu.Unlock()

	job, created := m.createJob(id, func(event backup.Accepted) {
		m.mu.Lock()
		if !m.active || m.currentID != id || m.running {
			m.mu.Unlock()
			return
		}
		m.running = true
		m.startedAt = event.Started.UTC()
		m.mu.Unlock()
		close(acceptedCh)
		m.recordAuditAsync(jobCtx, "backup_accepted", id, "running", event.Started.UTC(), nil)
	})
	if !created {
		cancel()
		m.logger.Error("backup job factory panic recovered", slog.String("backup_id", id))
		m.finish(auditCtx, id, errors.New("backup job factory panic"), false)
		m.wg.Done()
		return nil, ErrBackupUnavailable
	}

	go m.runJob(jobCtx, auditCtx, id, job, returnedCh)
	m.recordAuditAsync(auditCtx, "backup_requested", id, "requested", now, nil)

	select {
	case <-acceptedCh:
		return &model.BackupStartResponse{BackupID: id, State: backupStateRunning}, nil
	case err := <-returnedCh:
		switch {
		case errors.Is(err, backup.ErrLockHeld):
			return nil, ErrBackupLockHeld
		default:
			return nil, ErrBackupUnavailable
		}
	}
}

func (m *BackupJobManager) createJob(id string, accepted func(backup.Accepted)) (job BackupJob, created bool) {
	defer func() {
		if recover() != nil {
			job = nil
			created = false
		}
	}()
	return m.factory(id, accepted), true
}

func (m *BackupJobManager) runJob(jobCtx, auditCtx context.Context, id string, job BackupJob, returned chan<- error) {
	defer m.wg.Done()
	defer func() {
		if recover() != nil {
			m.mu.Lock()
			accepted := m.running && m.currentID == id
			m.mu.Unlock()
			m.logger.Error("backup job panic recovered", slog.String("backup_id", id))
			m.finish(auditCtx, id, errors.New("backup job panic"), accepted)
			if !accepted {
				returned <- ErrBackupUnavailable
			}
		}
	}()

	// The manager's running flag is changed only by the engine hook. Reading it
	// after Run is sufficient to distinguish a pre-accept return.
	err := job.Run(jobCtx)
	m.mu.Lock()
	accepted := m.running && m.currentID == id
	m.mu.Unlock()
	m.finish(auditCtx, id, err, accepted)
	if !accepted {
		returned <- err
	}
}

func (m *BackupJobManager) finish(ctx context.Context, id string, err error, accepted bool) {
	finishedAt := m.now().UTC()
	state := classifyBackupResult(err)
	result := &model.BackupLastResult{BackupID: id, State: state, FinishedAt: finishedAt.Format(time.RFC3339)}
	if err == nil {
		if opened, openErr := backup.OpenCompleted(context.Background(), m.destination, id); openErr == nil {
			size := opened.Size
			result.EncryptedSize = &size
			result.SHA256 = opened.SHA256
			_ = opened.Close()
		}
	}

	var cancel context.CancelFunc
	m.mu.Lock()
	if m.currentID == id {
		cancel = m.cancel
		m.active = false
		m.running = false
		m.currentID = ""
		m.startedAt = time.Time{}
		m.cancel = nil
		m.lastResult = result
	}
	m.mu.Unlock()
	if cancel != nil {
		cancel()
	}

	if errors.Is(err, backup.ErrLockHeld) && !accepted {
		m.recordAuditAsync(ctx, "backup_rejected_lock_held", id, "lock_held", finishedAt, nil)
		return
	}
	if err == nil {
		m.recordAuditAsync(ctx, "backup_completed", id, state, finishedAt, result)
		return
	}
	m.recordAuditAsync(ctx, "backup_failed", id, state, finishedAt, result)
	m.logger.Error("backup job failed", slog.String("backup_id", id), slog.String("failure_class", state))
}

func (m *BackupJobManager) Status() model.BackupStatusResponse {
	m.mu.Lock()
	defer m.mu.Unlock()
	status := model.BackupStatusResponse{State: backupStateIdle}
	if m.running {
		status.State = backupStateRunning
		status.BackupID = m.currentID
		status.StartedAt = m.startedAt.UTC().Format(time.RFC3339)
	}
	if m.lastResult != nil {
		copyResult := *m.lastResult
		status.LastResult = &copyResult
	}
	return status
}

func (m *BackupJobManager) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	m.stopping = true
	cancel := m.cancel
	m.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func classifyBackupResult(err error) string {
	if err == nil {
		return "completed"
	}
	if errors.Is(err, backup.ErrCleanup) {
		return "cleanup_failure"
	}
	switch {
	case errors.Is(err, backup.ErrLockHeld):
		return "lock_held"
	case errors.Is(err, backup.ErrConfig), errors.Is(err, backup.ErrPrereq):
		return "prerequisite_failure"
	case errors.Is(err, backup.ErrPackage):
		return "packaging_failure"
	case errors.Is(err, backup.ErrFinalize):
		return "finalization_failure"
	default:
		return "failed"
	}
}

func (m *BackupJobManager) recordAudit(ctx context.Context, event, id, state string, at time.Time, result *model.BackupLastResult) {
	defer func() {
		if recover() != nil {
			m.logger.Error("backup audit panic recovered", slog.String("backup_id", id), slog.String("event", event))
		}
	}()
	if m.audit == nil {
		return
	}
	auditCtx, _ := GetAuditLogContext(ctx)
	after := map[string]any{
		"event":     event,
		"backup_id": id,
		"state":     state,
		"timestamp": at.UTC().Format(time.RFC3339),
	}
	if result != nil {
		if result.EncryptedSize != nil {
			after["encrypted_size_bytes"] = *result.EncryptedSize
		}
		if result.SHA256 != "" {
			after["sha256"] = result.SHA256
		}
	}
	action := "UPDATE"
	if event == "backup_requested" {
		action = "CREATE"
	}
	if err := m.audit.Record(ctx, model.AuditLogRecordRequest{
		ActorUserID: auditCtx.ActorUserID,
		ActorRole:   auditCtx.ActorRole,
		Action:      action,
		Module:      "system-backup",
		EntityType:  "backups",
		EntityID:    id,
		EntityLabel: "Encrypted backup",
		Method:      auditCtx.Method,
		Route:       auditCtx.Route,
		AfterData:   after,
	}); err != nil {
		m.logger.Error("backup audit write failed", slog.String("backup_id", id), slog.String("event", event))
	}
}

func (m *BackupJobManager) recordAuditAsync(ctx context.Context, event, id, state string, at time.Time, result *model.BackupLastResult) {
	if m.audit == nil {
		return
	}
	auditBase := context.WithoutCancel(ctx)
	go func() {
		auditCtx, cancel := context.WithTimeout(auditBase, backupAuditTimeout)
		defer cancel()
		m.recordAudit(auditCtx, event, id, state, at, result)
	}()
}
