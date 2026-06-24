package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"permatatex-inventory/internal/entity"
	"permatatex-inventory/internal/model"
)

var (
	ErrDACPValidation        = errors.New("invalid data approve cutting plan payload")
	ErrDACPNotFound          = errors.New("data approve cutting plan not found")
	ErrDACPReferenceNotFound = errors.New("related data not found for data approve cutting plan")
)

type DataApproveCuttingPlanUseCase struct {
	repo   entity.Querier
	dbPool *pgxpool.Pool
}

func NewDataApproveCuttingPlanUseCase(repo entity.Querier, dbPool *pgxpool.Pool) (*DataApproveCuttingPlanUseCase, error) {
	if repo == nil {
		return nil, errors.New("data approve cutting plan repository is required")
	}
	if dbPool == nil {
		return nil, errors.New("database pool is required")
	}
	return &DataApproveCuttingPlanUseCase{repo: repo, dbPool: dbPool}, nil
}

// CreateDataApproveCuttingPlan creates a new DACP document and initializes the approval workflow.
func (u *DataApproveCuttingPlanUseCase) CreateDataApproveCuttingPlan(ctx context.Context, userID int32, req model.CreateDataApproveCuttingPlanRequest) (*model.DataApproveCuttingPlanResponse, error) {
	tanggal, err := time.Parse("2006-01-02", req.Tanggal)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid tanggal format", ErrDACPValidation)
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

	header, err := qtx.CreateDataApproveCuttingPlan(ctx, entity.CreateDataApproveCuttingPlanParams{
		NoDokumen: req.NoDokumen,
		Tanggal:   pgtype.Date{Time: tanggal, Valid: true},
		IDWo:      req.IDWo,
	})
	if err != nil {
		return nil, mapDACPDBError(err)
	}

	if err = initializeApprovalWorkflow(ctx, qtx, "DATA_APPROVE_CUTTING_PLAN", header.IDDacp, userID); err != nil {
		return nil, fmt.Errorf("failed to initialize approval workflow: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("%w: failed to commit transaction", ErrWorkOrderServiceUnavailable)
	}

	return u.GetDataApproveCuttingPlan(ctx, header.IDDacp)
}

// GetDataApproveCuttingPlan retrieves the header and aggregated row data for a DACP document.
func (u *DataApproveCuttingPlanUseCase) GetDataApproveCuttingPlan(ctx context.Context, idDacp int32) (*model.DataApproveCuttingPlanResponse, error) {
	header, err := u.repo.GetDataApproveCuttingPlanByID(ctx, idDacp)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDACPNotFound
		}
		return nil, fmt.Errorf("%w: %s", ErrWorkOrderServiceUnavailable, err.Error())
	}

	rowData, err := u.repo.GetDataApproveCuttingPlanRows(ctx, header.IDWo)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get dacp rows: %s", ErrWorkOrderServiceUnavailable, err.Error())
	}

	rows := make([]model.DataApproveCuttingPlanRow, 0, len(rowData))
	for _, r := range rowData {
		rows = append(rows, model.DataApproveCuttingPlanRow{
			Size:             r.Size,
			QtyOrder:         r.QtyOrder,
			QtyCuttingPlan:   r.QtyCuttingPlan,
			QtyCuttingActual: r.QtyCuttingActual,
			CuttingReport:    r.CuttingReport,
			BalanceAllowance: r.BalanceAllowance,
		})
	}

	return &model.DataApproveCuttingPlanResponse{
		IDDacp:    header.IDDacp,
		NoDokumen: header.NoDokumen,
		Tanggal:   formatDate(header.Tanggal),
		IDWo:      header.IDWo,
		Buyer:     header.Buyer,
		Model:     header.Model,
		Style:     header.Style,
		CreatedAt: header.CreatedAt.Time.Format(time.RFC3339),
		Rows:      rows,
	}, nil
}

// ListDataApproveCuttingPlans returns a paginated list.
func (u *DataApproveCuttingPlanUseCase) ListDataApproveCuttingPlans(ctx context.Context, filter model.TransactionListFilter) (*model.DataApproveCuttingPlanListResponse, error) {
	page, limit, offset, search, _, _ := normalizeListFilter(filter.ListQueryFilter, "id_dacp", true, nil)

	dbRows, err := u.repo.ListDataApproveCuttingPlans(ctx, entity.ListDataApproveCuttingPlansParams{
		SearchTerm: search,
		IDMitra:    nullableInt32Param(filter.IDMitra),
		PageOffset: offset,
		PageLimit:  limit,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: failed to list data approve cutting plans: %w", ErrWorkOrderServiceUnavailable, err)
	}

	items := make([]model.DataApproveCuttingPlanListItem, 0, len(dbRows))
	total := int64(0)
	for _, row := range dbRows {
		total = row.TotalCount
		items = append(items, model.DataApproveCuttingPlanListItem{
			IDDacp:    row.IDDacp,
			NoDokumen: row.NoDokumen,
			Tanggal:   formatDate(row.Tanggal),
			IDWo:      row.IDWo,
			Buyer:     row.Buyer,
			Model:     row.Model,
			CreatedAt: row.CreatedAt.Time.Format(time.RFC3339),
		})
	}

	return &model.DataApproveCuttingPlanListResponse{
		Items:      items,
		Pagination: buildPagination(total, page, limit),
	}, nil
}

func mapDACPDBError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503": // foreign_key_violation
			return fmt.Errorf("%w: %s", ErrDACPReferenceNotFound, pgErr.Detail)
		case "23505": // unique_violation
			return fmt.Errorf("%w: no dokumen sudah digunakan: %s", ErrDACPValidation, pgErr.Detail)
		}
	}
	return fmt.Errorf("%w: %s", ErrWorkOrderServiceUnavailable, err.Error())
}
