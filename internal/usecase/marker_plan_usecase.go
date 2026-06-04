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
	ErrMarkerPlanValidation        = errors.New("invalid marker plan payload")
	ErrMarkerPlanNotFound          = errors.New("marker plan not found")
	ErrMarkerPlanReferenceNotFound = errors.New("related data not found")
)

type MarkerPlanUseCase struct {
	repo   entity.Querier
	dbPool *pgxpool.Pool
}

func NewMarkerPlanUseCase(repo entity.Querier, dbPool *pgxpool.Pool) (*MarkerPlanUseCase, error) {
	if repo == nil {
		return nil, errors.New("marker plan repository is required")
	}
	if dbPool == nil {
		return nil, errors.New("database pool is required")
	}

	return &MarkerPlanUseCase{
		repo:   repo,
		dbPool: dbPool,
	}, nil
}

func (u *MarkerPlanUseCase) CreateMarkerPlan(ctx context.Context, req model.CreateMarkerPlanRequest) (*model.MarkerPlanResponse, error) {
	if len(req.Components) == 0 {
		return nil, fmt.Errorf("%w: components cannot be empty", ErrMarkerPlanValidation)
	}

	tanggalEfektif, err := time.Parse("2006-01-02", req.TanggalEfektif)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid tanggal_efektif format", ErrMarkerPlanValidation)
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

	// 1. Create Marker Plan Header
	header, err := qtx.CreateMarkerPlan(ctx, entity.CreateMarkerPlanParams{
		NoDokumen: req.NoDokumen,
		TanggalEfektif: pgtype.Date{
			Time:  tanggalEfektif,
			Valid: true,
		},
		IDWoShell: req.IDWoShell,
	})
	if err != nil {
		return nil, mapMarkerDBError(err)
	}

	// 2. Loop Components, Ratios, and Ratio Size Markers
	for _, compReq := range req.Components {
		comp, compErr := qtx.CreateKomponenMarkerPlan(ctx, entity.CreateKomponenMarkerPlanParams{
			IDMarkerPlan: header.IDMarkerPlan,
			NamaKomponen: compReq.NamaKomponen,
		})
		if compErr != nil {
			return nil, mapMarkerDBError(compErr)
		}

		for _, ratioReq := range compReq.Ratios {
			ratio, ratioErr := qtx.CreateRatioMarker(ctx, entity.CreateRatioMarkerParams{
				IDKomponenMarker:     comp.IDKomponenMarker,
				IDWoShell:            ratioReq.IDWoShell,
				Cons:                 mustNumeric(ratioReq.Cons),
				PlanSpreadingGelaran: mustNumeric(ratioReq.PlanSpreadingGelaran),
				PanjangMarker:        mustNumeric(ratioReq.PanjangMarker),
				EfficiencyMarker:     mustNumeric(ratioReq.EfficiencyMarker),
				Allowance:            mustNumeric(ratioReq.Allowance),
				ConsBuyer:            mustNumericPtr(ratioReq.ConsBuyer),
				RollQty:              ratioReq.RollQty,
				SambunganRoll:        ratioReq.SambunganRoll,
			})
			if ratioErr != nil {
				return nil, mapMarkerDBError(ratioErr)
			}

			// Bulk insert ratio size markers for this ratio marker
			sizeParams := make([]entity.CreateRatioSizeMarkerParams, len(ratioReq.Sizes))
			for idx, sizeReq := range ratioReq.Sizes {
				sizeParams[idx] = entity.CreateRatioSizeMarkerParams{
					IDRatioMarker: ratio.IDRatioMarker,
					IDWoShellSize: sizeReq.IDWoShellSize,
					QtyPlan:       sizeReq.QtyPlan,
				}
			}

			if _, sizeErr := qtx.CreateRatioSizeMarker(ctx, sizeParams); sizeErr != nil {
				return nil, mapMarkerDBError(sizeErr)
			}
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("%w: failed to commit transaction", ErrWorkOrderServiceUnavailable)
	}

	// Fetch detail to return structured response
	return u.GetMarkerPlan(ctx, header.IDMarkerPlan)
}

func (u *MarkerPlanUseCase) GetMarkerPlan(ctx context.Context, idMarkerPlan int32) (*model.MarkerPlanResponse, error) {
	header, err := u.repo.GetMarkerPlanByID(ctx, idMarkerPlan)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMarkerPlanNotFound
		}
		return nil, fmt.Errorf("%w: %s", ErrWorkOrderServiceUnavailable, err.Error())
	}

	compRows, err := u.repo.ListKomponenByMarkerPlanID(ctx, idMarkerPlan)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrWorkOrderServiceUnavailable, err.Error())
	}

	components := make([]model.KomponenMarkerPlanResponse, len(compRows))
	for i, cRow := range compRows {
		ratioRows, rErr := u.repo.ListRatioByKomponenID(ctx, cRow.IDKomponenMarker)
		if rErr != nil {
			return nil, fmt.Errorf("%w: %s", ErrWorkOrderServiceUnavailable, rErr.Error())
		}

		ratios := make([]model.RatioMarkerResponse, len(ratioRows))
		for j, rRow := range ratioRows {
			sizeRows, sErr := u.repo.ListRatioSizeByRatioID(ctx, rRow.IDRatioMarker)
			if sErr != nil {
				return nil, fmt.Errorf("%w: %s", ErrWorkOrderServiceUnavailable, sErr.Error())
			}

			sizes := make([]model.RatioSizeMarkerResponse, len(sizeRows))
			for k, sRow := range sizeRows {
				sizes[k] = model.RatioSizeMarkerResponse{
					IDRatioSizeMarker: sRow.IDRatioSizeMarker,
					IDRatioMarker:     sRow.IDRatioMarker,
					IDWoShellSize:     sRow.IDWoShellSize,
					QtyPlan:           sRow.QtyPlan,
					Size:              sRow.Size,
				}
			}

			ratios[j] = model.RatioMarkerResponse{
				IDRatioMarker:        rRow.IDRatioMarker,
				IDKomponenMarker:     rRow.IDKomponenMarker,
				IDWoShell:            rRow.IDWoShell,
				Cons:                 numericToFloat64(rRow.Cons),
				PlanSpreadingGelaran: numericToFloat64(rRow.PlanSpreadingGelaran),
				PanjangMarker:        numericToFloat64(rRow.PanjangMarker),
				EfficiencyMarker:     numericToFloat64(rRow.EfficiencyMarker),
				Allowance:            numericToFloat64(rRow.Allowance),
				ConsBuyer:            numericToFloat64Ptr(rRow.ConsBuyer),
				RollQty:              rRow.RollQty,
				SambunganRoll:        rRow.SambunganRoll,
				CreatedAt:            rRow.CreatedAt.Time.Format(time.RFC3339),
				Sizes:                sizes,
			}
		}

		components[i] = model.KomponenMarkerPlanResponse{
			IDKomponenMarker: cRow.IDKomponenMarker,
			IDMarkerPlan:     cRow.IDMarkerPlan,
			NamaKomponen:     cRow.NamaKomponen,
			CreatedAt:        cRow.CreatedAt.Time.Format(time.RFC3339),
			Ratios:           ratios,
		}
	}

	return &model.MarkerPlanResponse{
		IDMarkerPlan:   header.IDMarkerPlan,
		NoDokumen:      header.NoDokumen,
		TanggalEfektif: formatDate(header.TanggalEfektif),
		IDWoShell:      header.IDWoShell,
		CreatedAt:      header.CreatedAt.Time.Format(time.RFC3339),
		Components:     components,
	}, nil
}

func mustNumericPtr(value *float64) pgtype.Numeric {
	if value == nil {
		return pgtype.Numeric{Valid: false}
	}
	return mustNumeric(*value)
}

func numericToFloat64Ptr(value pgtype.Numeric) *float64 {
	if !value.Valid {
		return nil
	}
	floatVal, err := value.Float64Value()
	if err != nil || !floatVal.Valid {
		return nil
	}
	v := floatVal.Float64
	return &v
}

func mapMarkerDBError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503": // foreign_key_violation
			return fmt.Errorf("%w: %s", ErrMarkerPlanReferenceNotFound, pgErr.Detail)
		case "23505": // unique_violation
			return fmt.Errorf("%w: %s", ErrMarkerPlanValidation, pgErr.Detail)
		}
	}
	return fmt.Errorf("%w: %s", ErrWorkOrderServiceUnavailable, err.Error())
}


