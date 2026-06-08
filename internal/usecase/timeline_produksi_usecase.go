package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"permatatex-inventory/internal/entity"
	"permatatex-inventory/internal/model"
)

var (
	ErrTimelinePlanValidation        = errors.New("invalid timeline plan payload")
	ErrTimelinePlanNotFound          = errors.New("timeline plan not found")
	ErrTimelinePlanReferenceNotFound = errors.New("related data not found")
	ErrWOShellPlanNotFound           = errors.New("wo shell plan not found")
)

type TimelineProduksiUseCase struct {
	repo   entity.Querier
	dbPool *pgxpool.Pool
}

func NewTimelineProduksiUseCase(repo entity.Querier, dbPool *pgxpool.Pool) (*TimelineProduksiUseCase, error) {
	if repo == nil {
		return nil, errors.New("timeline repository is required")
	}
	if dbPool == nil {
		return nil, errors.New("database pool is required")
	}

	return &TimelineProduksiUseCase{
		repo:   repo,
		dbPool: dbPool,
	}, nil
}

func (u *TimelineProduksiUseCase) CreateTimelinePlan(ctx context.Context, userID int32, req model.CreateTimelinePlanRequest) (*model.TimelinePlanResponse, error) {
	if len(req.ShellPlans) == 0 {
		return nil, fmt.Errorf("%w: shell plans cannot be empty", ErrTimelinePlanValidation)
	}

	// Validate disusun date
	disusunTime, err := time.Parse("2006-01-02", req.TanggalDisusun)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid tanggal_disusun format", ErrTimelinePlanValidation)
	}

	// Validate date formats inside shell plans
	for _, sp := range req.ShellPlans {
		if sp.TglGelarCutting != nil && *sp.TglGelarCutting != "" {
			if _, err := time.Parse("2006-01-02", *sp.TglGelarCutting); err != nil {
				return nil, fmt.Errorf("%w: invalid tgl_gelar_cutting format", ErrTimelinePlanValidation)
			}
		}
		if sp.TglEmbroo != nil && *sp.TglEmbroo != "" {
			if _, err := time.Parse("2006-01-02", *sp.TglEmbroo); err != nil {
				return nil, fmt.Errorf("%w: invalid tgl_embroo format", ErrTimelinePlanValidation)
			}
		}
		if sp.TglLoadingSewing != nil && *sp.TglLoadingSewing != "" {
			if _, err := time.Parse("2006-01-02", *sp.TglLoadingSewing); err != nil {
				return nil, fmt.Errorf("%w: invalid tgl_loading_sewing format", ErrTimelinePlanValidation)
			}
		}
		if sp.TglFinishingPacking != nil && *sp.TglFinishingPacking != "" {
			if _, err := time.Parse("2006-01-02", *sp.TglFinishingPacking); err != nil {
				return nil, fmt.Errorf("%w: invalid tgl_finishing_packing format", ErrTimelinePlanValidation)
			}
		}
	}

	tx, err := u.dbPool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to start transaction", ErrWorkOrderServiceUnavailable)
	}
	defer func() {
		rollbackErr := tx.Rollback(ctx)
		if rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			err = rollbackErr
		}
	}()

	qtx := entity.New(tx)

	// Create header
	timelineHeader, err := qtx.CreateTimelinePlan(ctx, entity.CreateTimelinePlanParams{
		IDPoClient: req.IDPoClient,
		TanggalDisusun: pgtype.Date{
			Time:  disusunTime,
			Valid: true,
		},
		Notes: req.Notes,
	})
	if err != nil {
		return nil, mapTimelineDBError(err)
	}

	// Prepare shell plans params for bulk insert
	shellPlansParams := make([]entity.CreateWOShellPlanParams, len(req.ShellPlans))
	for i, sp := range req.ShellPlans {
		statusCutting := "PENDING"
		if sp.StatusGelarCutting != "" {
			statusCutting = sp.StatusGelarCutting
		}
		statusEmbroo := "PENDING"
		if sp.StatusEmbroo != "" {
			statusEmbroo = sp.StatusEmbroo
		}
		statusSewing := "PENDING"
		if sp.StatusLoadingSewing != "" {
			statusSewing = sp.StatusLoadingSewing
		}
		statusFinishing := "PENDING"
		if sp.StatusFinishingPacking != "" {
			statusFinishing = sp.StatusFinishingPacking
		}

		shellPlansParams[i] = entity.CreateWOShellPlanParams{
			IDTimeline:             timelineHeader.IDTimeline,
			IDWoShell:              sp.IDWoShell,
			InLine:                 sp.InLine,
			TglGelarCutting:        parseOptionalDate(sp.TglGelarCutting),
			StatusGelarCutting:     statusCutting,
			TglEmbroo:              parseOptionalDate(sp.TglEmbroo),
			StatusEmbroo:           statusEmbroo,
			TglLoadingSewing:       parseOptionalDate(sp.TglLoadingSewing),
			StatusLoadingSewing:    statusSewing,
			TglFinishingPacking:    parseOptionalDate(sp.TglFinishingPacking),
			StatusFinishingPacking: statusFinishing,
		}
	}

	// Bulk copy
	_, err = qtx.CreateWOShellPlan(ctx, shellPlansParams)
	if err != nil {
		return nil, mapTimelineDBError(err)
	}

	// Initialize approval workflow
	if err = initializeApprovalWorkflow(ctx, qtx, "TIMELINE_PRODUKSI", timelineHeader.IDTimeline, userID); err != nil {
		return nil, fmt.Errorf("failed to initialize approval workflow: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("%w: failed to commit transaction", ErrWorkOrderServiceUnavailable)
	}

	// Fetch detail to return complete structure
	return u.GetTimelinePlan(ctx, timelineHeader.IDTimeline)
}

func (u *TimelineProduksiUseCase) GetTimelinePlan(ctx context.Context, idTimeline int32) (*model.TimelinePlanResponse, error) {
	header, err := u.repo.GetTimelinePlanByID(ctx, idTimeline)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTimelinePlanNotFound
		}
		return nil, fmt.Errorf("%w: %s", ErrWorkOrderServiceUnavailable, err.Error())
	}

	rows, err := u.repo.GetWOShellPlansByTimelineID(ctx, idTimeline)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrWorkOrderServiceUnavailable, err.Error())
	}

	shellPlans := make([]model.WOShellPlanResponse, len(rows))
	for i, r := range rows {
		shellPlans[i] = model.WOShellPlanResponse{
			IDWoShellPlan:          r.IDWoShellPlan,
			IDTimeline:             r.IDTimeline,
			IDWoShell:              r.IDWoShell,
			InLine:                 r.InLine,
			TglGelarCutting:        formatDate(r.TglGelarCutting),
			StatusGelarCutting:     r.StatusGelarCutting,
			TglEmbroo:              formatDate(r.TglEmbroo),
			StatusEmbroo:           r.StatusEmbroo,
			TglLoadingSewing:       formatDate(r.TglLoadingSewing),
			StatusLoadingSewing:    r.StatusLoadingSewing,
			TglFinishingPacking:    formatDate(r.TglFinishingPacking),
			StatusFinishingPacking: r.StatusFinishingPacking,
			Deskripsi:              r.Deskripsi,
			Color:                  r.Color,
		}
	}

	return &model.TimelinePlanResponse{
		IDTimeline:     header.IDTimeline,
		IDPoClient:     header.IDPoClient,
		TanggalDisusun: formatDate(header.TanggalDisusun),
		Notes:          header.Notes,
		CreatedAt:      header.CreatedAt.Time.Format(time.RFC3339),
		ShellPlans:     shellPlans,
	}, nil
}

func (u *TimelineProduksiUseCase) UpdateWOShellPlanStatus(ctx context.Context, idWOShellPlan int32, req model.UpdateWOShellPlanStatusRequest) error {
	// Call sqlc generated update
	// Note: We use original positional parameters (Column2 - Column5)
	err := u.repo.UpdateWOShellPlanStatus(ctx, entity.UpdateWOShellPlanStatusParams{
		IDWoShellPlan: idWOShellPlan,
		Column2:       req.StatusGelarCutting,
		Column3:       req.StatusEmbroo,
		Column4:       req.StatusLoadingSewing,
		Column5:       req.StatusFinishingPacking,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrWOShellPlanNotFound
		}
		return mapTimelineDBError(err)
	}

	return nil
}

var timelineProduksiSortColumns = map[string]struct{}{
	"id_timeline": {},
	"created_at":  {},
}

func (u *TimelineProduksiUseCase) GetTimelinePlans(ctx context.Context, filter model.ListQueryFilter) (*model.TimelinePlanListResponse, error) {
	page, limit, offset, search, sortBy, sortDesc := normalizeListFilter(filter, "id_timeline", true, timelineProduksiSortColumns)

	rows, err := u.repo.ListTimelinePlans(ctx, entity.ListTimelinePlansParams{
		SearchTerm: search,
		SortBy:     sortBy,
		SortDesc:   sortDesc,
		PageLimit:  limit,
		PageOffset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrWorkOrderServiceUnavailable, err.Error())
	}

	totalItems, err := u.repo.CountTimelinePlans(ctx, search)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrWorkOrderServiceUnavailable, err.Error())
	}

	items := make([]model.TimelinePlanListItem, len(rows))
	for i, r := range rows {
		items[i] = model.TimelinePlanListItem{
			IDTimeline:       r.IDTimeline,
			IDPoClient:       r.IDPoClient,
			ClientName:       r.ClientName,
			PoInternalNumber: r.PoNumber,
			TanggalDisusun:   formatDate(r.TanggalDisusun),
			Notes:            r.Notes,
			CreatedAt:        r.CreatedAt.Time.Format(time.RFC3339),
		}
	}

	return &model.TimelinePlanListResponse{
		Items:      items,
		Pagination: buildPagination(totalItems, page, limit),
	}, nil
}

func parseOptionalDate(d *string) pgtype.Date {
	if d == nil || *d == "" {
		return pgtype.Date{Valid: false}
	}
	t, err := time.Parse("2006-01-02", *d)
	if err != nil {
		return pgtype.Date{Valid: false}
	}
	return pgtype.Date{Time: t, Valid: true}
}

func formatDate(d pgtype.Date) string {
	if !d.Valid {
		return ""
	}
	return d.Time.Format("2006-01-02")
}

func mapTimelineDBError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503": // foreign_key_violation
			return fmt.Errorf("%w: %s", ErrTimelinePlanReferenceNotFound, pgErr.Detail)
		case "23505": // unique_violation
			return fmt.Errorf("%w: %s", ErrTimelinePlanValidation, pgErr.Detail)
		}
	}
	return fmt.Errorf("%w: %s", ErrWorkOrderServiceUnavailable, err.Error())
}
