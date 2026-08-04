package usecase

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"permatatex-inventory/internal/backup"
	"permatatex-inventory/internal/model"
)

type backupJobFunc func(context.Context) error

func (f backupJobFunc) Run(ctx context.Context) error { return f(ctx) }

type recordedAudit struct {
	mu      sync.Mutex
	records []model.AuditLogRecordRequest
}

type blockingAudit struct {
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

func (a *blockingAudit) Record(context.Context, model.AuditLogRecordRequest) error {
	a.once.Do(func() { close(a.entered) })
	<-a.release
	return nil
}

type failingAudit struct{ panicValue any }

func (a failingAudit) Record(context.Context, model.AuditLogRecordRequest) error {
	if a.panicValue != nil {
		panic(a.panicValue)
	}
	return errors.New("private audit failure")
}

func (r *recordedAudit) Record(_ context.Context, req model.AuditLogRecordRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records = append(r.records, req)
	return nil
}

func (r *recordedAudit) events() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]string, 0, len(r.records))
	for _, record := range r.records {
		if after, ok := record.AfterData.(map[string]any); ok {
			if event, ok := after["event"].(string); ok {
				result = append(result, event)
			}
		}
	}
	return result
}

func (r *recordedAudit) snapshot() []model.AuditLogRecordRequest {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]model.AuditLogRecordRequest(nil), r.records...)
}

func TestBackupManagerWaitsForAcceptedAndRejectsDuplicate(t *testing.T) {
	allowAccept := make(chan struct{})
	finish := make(chan struct{})
	createdID := make(chan string, 1)
	audit := &recordedAudit{}
	manager := testBackupManager(t, func(id string, accepted func(backup.Accepted)) BackupJob {
		createdID <- id
		return backupJobFunc(func(context.Context) error {
			<-allowAccept
			accepted(backup.Accepted{BackupID: id, Started: time.Date(2026, 8, 3, 1, 2, 3, 0, time.UTC)})
			<-finish
			return nil
		})
	}, audit)

	type startOutcome struct {
		result *model.BackupStartResponse
		err    error
	}
	started := make(chan startOutcome, 1)
	go func() {
		result, err := manager.Start(context.Background())
		started <- startOutcome{result: result, err: err}
	}()
	id := <-createdID
	if _, err := backup.ParseBackupID(id); err != nil {
		t.Fatalf("manager did not generate a valid server ID: %q", id)
	}
	select {
	case <-started:
		t.Fatal("manager returned before accepted signal")
	case <-time.After(30 * time.Millisecond):
	}
	close(allowAccept)
	outcome := <-started
	if outcome.err != nil || outcome.result == nil || outcome.result.BackupID != id || outcome.result.State != "running" {
		t.Fatalf("unexpected accepted response: %#v %v", outcome.result, outcome.err)
	}
	if _, err := manager.Start(context.Background()); !errors.Is(err, ErrBackupRunning) {
		t.Fatalf("duplicate start=%v", err)
	}
	close(finish)
	waitFor(t, func() bool { return manager.Status().State == "idle" })
	waitFor(t, func() bool { return strings.Contains(strings.Join(audit.events(), ","), "backup_completed") })
	status := manager.Status()
	if status.LastResult == nil || status.LastResult.State != "completed" {
		t.Fatalf("async completion not recorded: %#v", status)
	}
	events := strings.Join(audit.events(), ",")
	for _, expected := range []string{"backup_requested", "backup_accepted", "backup_rejected_running", "backup_completed"} {
		if !strings.Contains(events, expected) {
			t.Fatalf("missing lifecycle event %s in %s", expected, events)
		}
	}
}

func TestBackupManagerMapsPreAcceptedFailures(t *testing.T) {
	for name, runErr := range map[string]struct {
		err  error
		want error
	}{
		"lock":         {backup.ErrLockHeld, ErrBackupLockHeld},
		"prerequisite": {backup.ErrPrereq, ErrBackupUnavailable},
	} {
		t.Run(name, func(t *testing.T) {
			audit := &recordedAudit{}
			manager := testBackupManager(t, func(string, func(backup.Accepted)) BackupJob {
				return backupJobFunc(func(context.Context) error { return runErr.err })
			}, audit)
			if _, err := manager.Start(context.Background()); !errors.Is(err, runErr.want) {
				t.Fatalf("got %v want %v", err, runErr.want)
			}
			if manager.Status().State != "idle" {
				t.Fatal("pre-accepted failure left manager running")
			}
			if name == "lock" {
				waitFor(t, func() bool {
					return strings.Contains(strings.Join(audit.events(), ","), "backup_rejected_lock_held")
				})
			}
		})
	}
}

func TestBackupManagerAsyncFailurePanicAndAuditAreSafe(t *testing.T) {
	for name, jobErr := range map[string]struct {
		err       error
		panicJob  bool
		wantState string
	}{
		"packaging": {backup.ErrPackage, false, "packaging_failure"},
		"panic":     {nil, true, "failed"},
	} {
		t.Run(name, func(t *testing.T) {
			audit := &recordedAudit{}
			manager := testBackupManager(t, func(id string, accepted func(backup.Accepted)) BackupJob {
				return backupJobFunc(func(context.Context) error {
					accepted(backup.Accepted{BackupID: id, Started: time.Now().UTC()})
					if jobErr.panicJob {
						panic("secret panic detail")
					}
					return jobErr.err
				})
			}, audit)
			if _, err := manager.Start(context.Background()); err != nil {
				t.Fatal(err)
			}
			waitFor(t, func() bool { return manager.Status().LastResult != nil })
			waitFor(t, func() bool { return strings.Contains(strings.Join(audit.events(), ","), "backup_failed") })
			status := manager.Status()
			if status.State != "idle" || status.LastResult.State != jobErr.wantState {
				t.Fatalf("unsafe terminal state: %#v", status)
			}
			events := strings.Join(audit.events(), ",")
			for _, expected := range []string{"backup_requested", "backup_accepted", "backup_failed"} {
				if !strings.Contains(events, expected) {
					t.Fatalf("missing audit event %s in %s", expected, events)
				}
			}
			if strings.Contains(events, "secret") {
				t.Fatal("panic content leaked into audit metadata")
			}
		})
	}
}

func TestBackupManagerShutdownCancelsAndWaits(t *testing.T) {
	cancelled := make(chan struct{})
	manager := testBackupManager(t, func(id string, accepted func(backup.Accepted)) BackupJob {
		return backupJobFunc(func(ctx context.Context) error {
			accepted(backup.Accepted{BackupID: id, Started: time.Now().UTC()})
			<-ctx.Done()
			close(cancelled)
			return ctx.Err()
		})
	}, nil)
	if _, err := manager.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := manager.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case <-cancelled:
	default:
		t.Fatal("active job was not cancelled before shutdown returned")
	}
	if _, err := manager.Start(context.Background()); !errors.Is(err, ErrBackupUnavailable) {
		t.Fatal("shutdown manager accepted a new start")
	}
}

func TestBackupManagerPanicBeforeAcceptedReturnsSafeUnavailable(t *testing.T) {
	var logs bytes.Buffer
	manager, err := NewBackupJobManager(func(string, func(backup.Accepted)) BackupJob {
		return backupJobFunc(func(context.Context) error { panic("private panic detail") })
	}, "", nil, slog.New(slog.NewTextHandler(&logs, nil)))
	if err != nil {
		t.Fatal(err)
	}
	manager.now = func() time.Time { return time.Date(2026, 8, 3, 1, 2, 3, 0, time.UTC) }
	if _, err := manager.Start(context.Background()); !errors.Is(err, ErrBackupUnavailable) {
		t.Fatalf("panic result=%v", err)
	}
	status := manager.Status()
	if status.State != "idle" || status.LastResult == nil || status.LastResult.State != "failed" {
		t.Fatalf("panic state=%#v", status)
	}
	if strings.Contains(logs.String(), "private panic detail") {
		t.Fatal("panic contents leaked to operational logs")
	}
}

func TestBackupManagerFactoryPanicDoesNotWedgeLaterStart(t *testing.T) {
	var calls int
	manager := testBackupManager(t, func(id string, accepted func(backup.Accepted)) BackupJob {
		calls++
		if calls == 1 {
			panic("private factory panic")
		}
		return backupJobFunc(func(context.Context) error {
			accepted(backup.Accepted{BackupID: id, Started: time.Now().UTC()})
			return nil
		})
	}, nil)
	if _, err := manager.Start(context.Background()); !errors.Is(err, ErrBackupUnavailable) {
		t.Fatalf("factory panic result=%v", err)
	}
	status := manager.Status()
	if status.State != "idle" || status.LastResult == nil || status.LastResult.State != "failed" {
		t.Fatalf("factory panic left unsafe state: %#v", status)
	}
	if _, err := manager.Start(context.Background()); err != nil {
		t.Fatalf("later start remained wedged: %v", err)
	}
	waitFor(t, func() bool { return manager.Status().State == "idle" })
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := manager.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown after factory panic=%v", err)
	}
}

func TestBackupManagerAuditRecordersNeverGateLifecycle(t *testing.T) {
	blocker := &blockingAudit{entered: make(chan struct{}), release: make(chan struct{})}
	tests := []struct {
		name    string
		audit   backupAuditRecorder
		release func()
	}{
		{name: "blocks", audit: blocker, release: func() { close(blocker.release) }},
		{name: "returns error", audit: failingAudit{}},
		{name: "panics", audit: failingAudit{panicValue: "private audit panic"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			jobCompleted := make(chan struct{})
			manager := testBackupManager(t, func(id string, accepted func(backup.Accepted)) BackupJob {
				return backupJobFunc(func(context.Context) error {
					accepted(backup.Accepted{BackupID: id, Started: time.Now().UTC()})
					close(jobCompleted)
					return nil
				})
			}, test.audit)
			started := make(chan error, 1)
			go func() {
				_, err := manager.Start(context.Background())
				started <- err
			}()
			select {
			case err := <-started:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(250 * time.Millisecond):
				t.Fatal("audit recorder blocked start acceptance")
			}
			select {
			case <-jobCompleted:
			case <-time.After(250 * time.Millisecond):
				t.Fatal("audit recorder blocked job progress")
			}
			waitFor(t, func() bool { return manager.Status().LastResult != nil })
			ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
			if err := manager.Shutdown(ctx); err != nil {
				cancel()
				t.Fatalf("audit recorder blocked shutdown: %v", err)
			}
			cancel()
			if test.name == "blocks" {
				select {
				case <-blocker.entered:
				case <-time.After(250 * time.Millisecond):
					t.Fatal("blocking audit recorder was never invoked")
				}
			}
			if test.release != nil {
				test.release()
			}
		})
	}
}

func TestBackupManagerShutdownHonorsExistingDeadline(t *testing.T) {
	release := make(chan struct{})
	manager := testBackupManager(t, func(id string, accepted func(backup.Accepted)) BackupJob {
		return backupJobFunc(func(context.Context) error {
			accepted(backup.Accepted{BackupID: id, Started: time.Now().UTC()})
			<-release
			return nil
		})
	}, nil)
	if _, err := manager.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	started := time.Now()
	if err := manager.Shutdown(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("shutdown result=%v", err)
	}
	if time.Since(started) > 250*time.Millisecond {
		t.Fatal("shutdown blocked beyond application deadline")
	}
	close(release)
}

func TestNewBackupManagerStartsIdleWithoutDurableInterruptedClaim(t *testing.T) {
	manager := testBackupManager(t, func(string, func(backup.Accepted)) BackupJob {
		return backupJobFunc(func(context.Context) error { return nil })
	}, nil)
	status := manager.Status()
	if status.State != "idle" || status.LastResult != nil || status.BackupID != "" {
		t.Fatalf("new manager claimed prior state: %#v", status)
	}
}

func TestBackupManagerInvalidEngineConfigurationFailsBeforeAcceptance(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	manager, err := NewBackupJobManager(func(id string, accepted func(backup.Accepted)) BackupJob {
		job := backup.NewJob(backup.Config{}, logger)
		job.ID = id
		job.Accepted = accepted
		return job
	}, "", nil, logger)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Start(context.Background()); !errors.Is(err, ErrBackupUnavailable) {
		t.Fatalf("invalid configuration result=%v", err)
	}
	status := manager.Status()
	if status.State != "idle" || status.LastResult == nil || status.LastResult.State != "prerequisite_failure" {
		t.Fatalf("invalid configuration status=%#v", status)
	}
}

func testBackupManager(t *testing.T, factory BackupJobFactory, audit backupAuditRecorder) *BackupJobManager {
	t.Helper()
	manager, err := NewBackupJobManager(factory, "", audit, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	manager.now = func() time.Time { return time.Date(2026, 8, 3, 1, 2, 3, 0, time.UTC) }
	return manager
}

func waitFor(t *testing.T, predicate func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for !predicate() {
		if time.Now().After(deadline) {
			t.Fatal("condition was not met")
		}
		time.Sleep(time.Millisecond)
	}
}
