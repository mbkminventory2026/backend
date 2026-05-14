package usecase

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"

	"permatatex-inventory/internal/entity"
)

type ActivityLogEntry struct {
	UserID      *int32
	Action      string
	TableName   string
	Description string
}

type ActivityLogService struct {
	queries *entity.Queries
	logger  *slog.Logger
	queue   chan ActivityLogEntry
	wg      sync.WaitGroup
}

func NewActivityLogService(queries *entity.Queries, logger *slog.Logger) (*ActivityLogService, error) {
	if queries == nil {
		return nil, errors.New("queries is required")
	}
	if logger == nil {
		logger = slog.Default()
	}

	service := &ActivityLogService{
		queries: queries,
		logger:  logger,
		queue:   make(chan ActivityLogEntry, 256),
	}

	service.wg.Add(1)
	go service.runWorker()

	return service, nil
}

func (s *ActivityLogService) Record(entry ActivityLogEntry) {
	if strings.TrimSpace(entry.Action) == "" || strings.TrimSpace(entry.TableName) == "" {
		return
	}

	select {
	case s.queue <- entry:
	default:
		s.logger.Warn("activity log queue full; dropping log entry",
			slog.String("action", entry.Action),
			slog.String("table", entry.TableName),
		)
	}
}

func (s *ActivityLogService) Shutdown(ctx context.Context) error {
	close(s.queue)

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

func (s *ActivityLogService) runWorker() {
	defer s.wg.Done()

	for entry := range s.queue {
		if err := s.persist(context.Background(), entry); err != nil {
			s.logger.Error("failed to persist activity log", slog.String("error", err.Error()))
		}
	}
}

func (s *ActivityLogService) persist(ctx context.Context, entry ActivityLogEntry) error {
	logRow, err := s.queries.CreateAktivitasLog(ctx, entry.Action)
	if err != nil {
		return err
	}

	actorName := "anonymous"
	if entry.UserID != nil {
		user, userErr := s.queries.GetUserByID(ctx, *entry.UserID)
		if userErr == nil {
			actorName = user.Username
		} else if !errors.Is(userErr, pgx.ErrNoRows) {
			s.logger.Warn("failed to resolve log actor", slog.String("error", userErr.Error()))
			actorName = "user"
		} else {
			actorName = "user"
		}
	}

	_, err = s.queries.CreateAktivitasLogDetail(ctx, entity.CreateAktivitasLogDetailParams{
		Nama:      actorName,
		TableName: entry.TableName,
		Deskripsi: entry.Description,
		IDLog:     logRow.IDLog,
	})
	return err
}

func BuildActivityDescription(method, route string, status int, duration time.Duration) string {
	return strings.TrimSpace(method + " " + route + " completed with status " + strconv.Itoa(status) + " in " + duration.String())
}
