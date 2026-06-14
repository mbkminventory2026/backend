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
	ErrSpreadingCuttingPlanValidation        = errors.New("invalid spreading cutting plan payload")
	ErrSpreadingCuttingPlanNotFound          = errors.New("spreading cutting plan not found")
	ErrSpreadingCuttingPlanReferenceNotFound = errors.New("related data not found")
)

var spreadingCuttingPlanSortColumns = buildSortWhitelist("created_at", "no_dokumen", "tanggal_efektif", "model", "buyer")

type SpreadingCuttingPlanUseCase struct {
	repo   entity.Querier
	dbPool *pgxpool.Pool
}

func NewSpreadingCuttingPlanUseCase(repo entity.Querier, dbPool *pgxpool.Pool) (*SpreadingCuttingPlanUseCase, error) {
	if repo == nil {
		return nil, errors.New("spreading cutting plan repository is required")
	}
	if dbPool == nil {
		return nil, errors.New("database pool is required")
	}

	return &SpreadingCuttingPlanUseCase{
		repo:   repo,
		dbPool: dbPool,
	}, nil
}

func (u *SpreadingCuttingPlanUseCase) CreateSpreadingCuttingPlan(ctx context.Context, userID int32, req model.CreateSpreadingCuttingPlanRequest) (*model.SpreadingCuttingPlanResponse, error) {
	if len(req.Components) == 0 {
		return nil, fmt.Errorf("%w: components cannot be empty", ErrSpreadingCuttingPlanValidation)
	}

	tanggalEfektif, err := time.Parse("2006-01-02", req.TanggalEfektif)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid tanggal_efektif format", ErrSpreadingCuttingPlanValidation)
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

	// 1. Create Spreading Cutting Plan Header
	header, err := qtx.CreateSpreadingCuttingPlan(ctx, entity.CreateSpreadingCuttingPlanParams{
		NoDokumen: req.NoDokumen,
		TanggalEfektif: pgtype.Date{
			Time:  tanggalEfektif,
			Valid: true,
		},
		IDWo: req.IDWo,
	})
	if err != nil {
		return nil, mapSpreadingDBError(err)
	}

	// 2. Loop Components, Ratios, and Ratio Size Spreading
	for _, compReq := range req.Components {
		comp, compErr := qtx.CreateKomponenSpreadingCuttingPlan(ctx, entity.CreateKomponenSpreadingCuttingPlanParams{
			IDSpreadingCuttingPlan: header.IDSpreadingCuttingPlan,
			NamaKomponen:           compReq.NamaKomponen,
		})
		if compErr != nil {
			return nil, mapSpreadingDBError(compErr)
		}

		for _, ratioReq := range compReq.Ratios {
			ratio, ratioErr := qtx.CreateRatioSpreading(ctx, entity.CreateRatioSpreadingParams{
				IDKomponenSpreading:  comp.IDKomponenSpreading,
				IDWoShell:            ratioReq.IDWoShell,
				Cons:                 mustNumeric(ratioReq.Cons),
				PlanSpreadingGelaran: mustNumeric(ratioReq.PlanSpreadingGelaran),
				Allowance:            mustNumeric(ratioReq.Allowance),
				RollQty:              ratioReq.RollQty,
				SambunganRoll:        ratioReq.SambunganRoll,
				Reject:               mustNumeric(ratioReq.Reject),
				LebarKain:            mustNumeric(ratioReq.LebarKain),
				Ket:                  ratioReq.Ket,
			})
			if ratioErr != nil {
				return nil, mapSpreadingDBError(ratioErr)
			}

			// Bulk insert ratio size spreading for this ratio spreading
			sizeParams := make([]entity.CreateRatioSizeSpreadingParams, len(ratioReq.Sizes))
			for idx, sizeReq := range ratioReq.Sizes {
				sizeParams[idx] = entity.CreateRatioSizeSpreadingParams{
					IDRatioSpreading: ratio.IDRatioSpreading,
					IDWoShellSize:    sizeReq.IDWoShellSize,
					RatioPlan:        sizeReq.RatioPlan,
				}
			}

			if _, sizeErr := qtx.CreateRatioSizeSpreading(ctx, sizeParams); sizeErr != nil {
				return nil, mapSpreadingDBError(sizeErr)
			}
		}
	}

	// Initialize approval workflow
	if err = initializeApprovalWorkflow(ctx, qtx, "SPREADING_CUTTING_PLAN", header.IDSpreadingCuttingPlan, userID); err != nil {
		return nil, fmt.Errorf("failed to initialize approval workflow: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("%w: failed to commit transaction", ErrWorkOrderServiceUnavailable)
	}

	// Fetch detail to return structured response
	return u.GetSpreadingCuttingPlan(ctx, header.IDSpreadingCuttingPlan)
}

func (u *SpreadingCuttingPlanUseCase) GetSpreadingCuttingPlan(ctx context.Context, idSpreadingCuttingPlan int32) (*model.SpreadingCuttingPlanResponse, error) {
	header, err := u.repo.GetSpreadingCuttingPlanByID(ctx, idSpreadingCuttingPlan)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSpreadingCuttingPlanNotFound
		}
		return nil, fmt.Errorf("%w: %s", ErrWorkOrderServiceUnavailable, err.Error())
	}

	compRows, err := u.repo.ListKomponenBySpreadingPlanID(ctx, idSpreadingCuttingPlan)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrWorkOrderServiceUnavailable, err.Error())
	}

	components := make([]model.KomponenSpreadingResponse, len(compRows))
	for i, cRow := range compRows {
		ratioRows, rErr := u.repo.ListRatioByKomponenSpreadingID(ctx, cRow.IDKomponenSpreading)
		if rErr != nil {
			return nil, fmt.Errorf("%w: %s", ErrWorkOrderServiceUnavailable, rErr.Error())
		}

		ratios := make([]model.RatioSpreadingResponse, len(ratioRows))
		for j, rRow := range ratioRows {
			sizeRows, sErr := u.repo.ListRatioSizeByRatioSpreadingID(ctx, rRow.IDRatioSpreading)
			if sErr != nil {
				return nil, fmt.Errorf("%w: %s", ErrWorkOrderServiceUnavailable, sErr.Error())
			}

			sizes := make([]model.RatioSizeSpreadingResponse, len(sizeRows))
			for k, sRow := range sizeRows {
				sizes[k] = model.RatioSizeSpreadingResponse{
					IDRatioSizeSpreading: sRow.IDRatioSizeSpreading,
					IDRatioSpreading:     sRow.IDRatioSpreading,
					IDWoShellSize:        sRow.IDWoShellSize,
					RatioPlan:            sRow.RatioPlan,
					Size:                 sRow.Size,
					SizeQty:              sRow.SizeQty,
				}
			}

			ratios[j] = model.RatioSpreadingResponse{
				IDRatioSpreading:     rRow.IDRatioSpreading,
				IDKomponenSpreading:  rRow.IDKomponenSpreading,
				IDWoShell:            rRow.IDWoShell,
				Cons:                 numericToFloat64(rRow.Cons),
				PlanSpreadingGelaran: numericToFloat64(rRow.PlanSpreadingGelaran),
				Allowance:            numericToFloat64(rRow.Allowance),
				RollQty:              rRow.RollQty,
				SambunganRoll:        rRow.SambunganRoll,
				Reject:               numericToFloat64(rRow.Reject),
				LebarKain:            numericToFloat64(rRow.LebarKain),
				Ket:                  rRow.Ket,
				CreatedAt:            rRow.CreatedAt.Time.Format(time.RFC3339),
				Sizes:                sizes,
			}
		}

		components[i] = model.KomponenSpreadingResponse{
			IDKomponenSpreading:    cRow.IDKomponenSpreading,
			IDSpreadingCuttingPlan: cRow.IDSpreadingCuttingPlan,
			NamaKomponen:           cRow.NamaKomponen,
			CreatedAt:              cRow.CreatedAt.Time.Format(time.RFC3339),
			Ratios:                 ratios,
		}
	}

	return &model.SpreadingCuttingPlanResponse{
		IDSpreadingCuttingPlan: header.IDSpreadingCuttingPlan,
		NoDokumen:              header.NoDokumen,
		TanggalEfektif:         formatDate(header.TanggalEfektif),
		IDWo:                   header.IDWo,
		Style:                  header.Style,
		Model:                  header.Model,
		Buyer:                  header.Buyer,
		CreatedAt:              header.CreatedAt.Time.Format(time.RFC3339),
		Components:             components,
	}, nil
}

func (u *SpreadingCuttingPlanUseCase) ListSpreadingCuttingPlans(ctx context.Context, filter model.TransactionListFilter) (*model.SpreadingCuttingPlanListResponse, error) {
	page, limit, offset, search, sortBy, sortDesc := normalizeListFilter(filter.ListQueryFilter, "id_spreading_cutting_plan", true, spreadingCuttingPlanSortColumns)

	rows, err := u.repo.ListSpreadingCuttingPlans(ctx, entity.ListSpreadingCuttingPlansParams{
		SearchTerm: search,
		IDMitra:    nullableInt32Param(filter.IDMitra),
		SortBy:     sortBy,
		SortDesc:   sortDesc,
		PageLimit:  limit,
		PageOffset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: failed to list spreading cutting plans: %w", ErrWorkOrderServiceUnavailable, err)
	}

	items := make([]model.SpreadingCuttingPlanListItem, 0, len(rows))
	total := int64(0)
	for _, row := range rows {
		total = row.TotalCount
		items = append(items, model.SpreadingCuttingPlanListItem{
			IDSpreadingCuttingPlan: row.IDSpreadingCuttingPlan,
			NoDokumen:              row.NoDokumen,
			TanggalEfektif:         formatDate(row.TanggalEfektif),
			IDWo:                   row.IDWo,
			Buyer:                  row.Buyer,
			Model:                  row.Model,
			CreatedAt:              row.CreatedAt.Time.Format(time.RFC3339),
		})
	}

	return &model.SpreadingCuttingPlanListResponse{
		Items:      items,
		Pagination: buildPagination(total, page, limit),
	}, nil
}

func mapSpreadingDBError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503": // foreign_key_violation
			return fmt.Errorf("%w: %s", ErrSpreadingCuttingPlanReferenceNotFound, pgErr.Detail)
		case "23505": // unique_violation
			return fmt.Errorf("%w: %s", ErrSpreadingCuttingPlanValidation, pgErr.Detail)
		}
	}
	return fmt.Errorf("%w: %s", ErrWorkOrderServiceUnavailable, err.Error())
}
