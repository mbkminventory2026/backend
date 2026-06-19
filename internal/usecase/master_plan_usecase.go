package usecase

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"permatatex-inventory/internal/entity"
	"permatatex-inventory/internal/model"
)

type masterPlanItemRaw struct {
	idMasterPlanItem int32
	idMasterPlan     int32
	idWoShell        int32
	idWo             int32
	noUrut           int32
	buyer            string
	style            string
	qty              int32
	color            string
	deskripsi        string
	createdAt        pgtype.Timestamptz
}

var (
	ErrMasterPlanValidation        = errors.New("invalid master plan payload")
	ErrMasterPlanNotFound          = errors.New("master plan not found")
	ErrMasterPlanItemNotFound      = errors.New("master plan item not found")
	ErrMasterPlanReferenceNotFound = errors.New("related data not found")
	ErrMasterPlanDuplicate         = errors.New("duplicate entry in master plan")
)

type MasterPlanUseCase struct {
	repo entity.Querier
}

func NewMasterPlanUseCase(repo entity.Querier) (*MasterPlanUseCase, error) {
	if repo == nil {
		return nil, errors.New("master plan repository is required")
	}
	return &MasterPlanUseCase{repo: repo}, nil
}

func (u *MasterPlanUseCase) CreateMasterPlan(ctx context.Context, userID int32, req model.CreateMasterPlanRequest) (*model.MasterPlanResponse, error) {
	plan, err := u.repo.CreateMasterPlan(ctx, entity.CreateMasterPlanParams{
		IDDepartemen:     req.IDDepartemen,
		IDProductionLine: req.IDProductionLine,
		Nama:             req.Nama,
		CreatedBy:        pgtype.Int4{Int32: userID, Valid: true},
	})
	if err != nil {
		return nil, mapMasterPlanDBError(err)
	}

	for i, itemReq := range req.Items {
		noUrut := itemReq.NoUrut
		if noUrut == 0 {
			nextNoUrut, convErr := intToInt32(i + 1)
			if convErr != nil {
				return nil, convErr
			}
			noUrut = nextNoUrut
		}
		if _, addErr := u.repo.AddMasterPlanItem(ctx, entity.AddMasterPlanItemParams{
			IDMasterPlan: plan.IDMasterPlan,
			IDWoShell:    itemReq.IDWoShell,
			NoUrut:       noUrut,
		}); addErr != nil {
			return nil, mapMasterPlanDBError(addErr)
		}
	}

	return u.GetMasterPlan(ctx, plan.IDMasterPlan)
}

func (u *MasterPlanUseCase) GetMasterPlan(ctx context.Context, id int32) (*model.MasterPlanResponse, error) {
	header, err := u.repo.GetMasterPlanByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMasterPlanNotFound
		}
		return nil, fmt.Errorf("%w: %s", ErrWorkOrderServiceUnavailable, err.Error())
	}

	itemRows, err := u.repo.ListMasterPlanItems(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrWorkOrderServiceUnavailable, err.Error())
	}

	items := make([]model.MasterPlanItemResponse, 0, len(itemRows))
	for _, ir := range itemRows {
		detail, buildErr := u.buildItemResponse(ctx, masterPlanItemRaw{
			idMasterPlanItem: ir.IDMasterPlanItem,
			idMasterPlan:     ir.IDMasterPlan,
			idWoShell:        ir.IDWoShell,
			idWo:             ir.IDWo,
			noUrut:           ir.NoUrut,
			buyer:            ir.Buyer,
			style:            ir.Style,
			qty:              ir.Qty,
			color:            ir.Color,
			deskripsi:        ir.Deskripsi,
			createdAt:        ir.CreatedAt,
		})
		if buildErr != nil {
			return nil, buildErr
		}
		items = append(items, *detail)
	}

	return &model.MasterPlanResponse{
		IDMasterPlan:     header.IDMasterPlan,
		IDDepartemen:     header.IDDepartemen,
		NamaDepartemen:   header.NamaDepartemen,
		IDProductionLine: header.IDProductionLine,
		NamaLine:         header.NamaLine,
		Nama:             header.Nama,
		CreatedAt:        header.CreatedAt.Time.Format(time.RFC3339),
		Items:            items,
	}, nil
}

func (u *MasterPlanUseCase) ListMasterPlans(ctx context.Context, filter model.ListQueryFilter) (*model.MasterPlanListResponse, error) {
	page, limit, offset, search, _, _ := normalizeListFilter(filter, "created_at", true, nil)

	rows, err := u.repo.ListMasterPlans(ctx, entity.ListMasterPlansParams{
		SearchTerm: search,
		PageLimit:  limit,
		PageOffset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: failed to list master plans: %w", ErrWorkOrderServiceUnavailable, err)
	}

	items := make([]model.MasterPlanListItem, 0, len(rows))
	total := int64(0)
	for _, row := range rows {
		total = row.TotalCount
		items = append(items, model.MasterPlanListItem{
			IDMasterPlan:     row.IDMasterPlan,
			IDDepartemen:     row.IDDepartemen,
			NamaDepartemen:   row.NamaDepartemen,
			IDProductionLine: row.IDProductionLine,
			NamaLine:         row.NamaLine,
			Nama:             row.Nama,
			CreatedAt:        row.CreatedAt.Time.Format(time.RFC3339),
		})
	}

	return &model.MasterPlanListResponse{
		Items:      items,
		Pagination: buildPagination(total, page, limit),
	}, nil
}

func (u *MasterPlanUseCase) UpdateMasterPlan(ctx context.Context, id int32, req model.UpdateMasterPlanRequest) (*model.MasterPlanResponse, error) {
	if _, err := u.repo.UpdateMasterPlan(ctx, entity.UpdateMasterPlanParams{
		IDMasterPlan: id,
		Nama:         req.Nama,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMasterPlanNotFound
		}
		return nil, mapMasterPlanDBError(err)
	}
	return u.GetMasterPlan(ctx, id)
}

func (u *MasterPlanUseCase) DeleteMasterPlan(ctx context.Context, id int32) error {
	if _, err := u.repo.GetMasterPlanByID(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrMasterPlanNotFound
		}
		return fmt.Errorf("%w: %s", ErrWorkOrderServiceUnavailable, err.Error())
	}
	return u.repo.DeleteMasterPlan(ctx, id)
}

func (u *MasterPlanUseCase) AddItem(ctx context.Context, planID int32, req model.AddMasterPlanItemRequest) (*model.MasterPlanItemResponse, error) {
	if _, err := u.repo.GetMasterPlanByID(ctx, planID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMasterPlanNotFound
		}
		return nil, fmt.Errorf("%w: %s", ErrWorkOrderServiceUnavailable, err.Error())
	}

	noUrut := req.NoUrut
	if noUrut == 0 {
		existingItems, err := u.repo.ListMasterPlanItems(ctx, planID)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrWorkOrderServiceUnavailable, err.Error())
		}
		nextNoUrut, convErr := intToInt32(len(existingItems) + 1)
		if convErr != nil {
			return nil, convErr
		}
		noUrut = nextNoUrut
	}

	item, err := u.repo.AddMasterPlanItem(ctx, entity.AddMasterPlanItemParams{
		IDMasterPlan: planID,
		IDWoShell:    req.IDWoShell,
		NoUrut:       noUrut,
	})
	if err != nil {
		return nil, mapMasterPlanDBError(err)
	}

	detail, err := u.repo.GetMasterPlanItemByID(ctx, item.IDMasterPlanItem)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrWorkOrderServiceUnavailable, err.Error())
	}

	return u.buildItemResponse(ctx, masterPlanItemRaw{
		idMasterPlanItem: detail.IDMasterPlanItem,
		idMasterPlan:     detail.IDMasterPlan,
		idWoShell:        detail.IDWoShell,
		idWo:             detail.IDWo,
		noUrut:           detail.NoUrut,
		buyer:            detail.Buyer,
		style:            detail.Style,
		qty:              detail.Qty,
		color:            detail.Color,
		deskripsi:        detail.Deskripsi,
		createdAt:        detail.CreatedAt,
	})
}

func (u *MasterPlanUseCase) RemoveItem(ctx context.Context, planID, itemID int32) error {
	affected, err := u.repo.RemoveMasterPlanItem(ctx, entity.RemoveMasterPlanItemParams{
		IDMasterPlanItem: itemID,
		IDMasterPlan:     planID,
	})
	if err != nil {
		return fmt.Errorf("%w: %s", ErrWorkOrderServiceUnavailable, err.Error())
	}
	if affected == 0 {
		return ErrMasterPlanItemNotFound
	}
	return nil
}

func (u *MasterPlanUseCase) UpsertTargetHarian(ctx context.Context, planID, itemID int32, req model.UpsertTargetHarianRequest) error {
	if err := u.verifyItemBelongsToPlan(ctx, planID, itemID); err != nil {
		return err
	}
	for _, entry := range req.Entries {
		t, parseErr := time.Parse("2006-01-02", entry.Tanggal)
		if parseErr != nil {
			return fmt.Errorf("%w: invalid tanggal format %s", ErrMasterPlanValidation, entry.Tanggal)
		}
		if _, upsErr := u.repo.UpsertTargetHarian(ctx, entity.UpsertTargetHarianParams{
			IDMasterPlanItem: itemID,
			Column2:          pgtype.Date{Time: t, Valid: true},
			Target:           entry.Target,
		}); upsErr != nil {
			return fmt.Errorf("%w: %s", ErrWorkOrderServiceUnavailable, upsErr.Error())
		}
	}
	return nil
}

func (u *MasterPlanUseCase) UpsertOutputHarian(ctx context.Context, planID, itemID int32, req model.UpsertOutputHarianRequest) error {
	if err := u.verifyItemBelongsToPlan(ctx, planID, itemID); err != nil {
		return err
	}
	for _, entry := range req.Entries {
		t, parseErr := time.Parse("2006-01-02", entry.Tanggal)
		if parseErr != nil {
			return fmt.Errorf("%w: invalid tanggal format %s", ErrMasterPlanValidation, entry.Tanggal)
		}
		if _, upsErr := u.repo.UpsertOutputHarian(ctx, entity.UpsertOutputHarianParams{
			IDMasterPlanItem: itemID,
			Column2:          pgtype.Date{Time: t, Valid: true},
			Output:           entry.Output,
		}); upsErr != nil {
			return fmt.Errorf("%w: %s", ErrWorkOrderServiceUnavailable, upsErr.Error())
		}
	}
	return nil
}

func (u *MasterPlanUseCase) UpsertTargetProses(ctx context.Context, planID, itemID int32, req model.UpsertTargetProsesRequest) error {
	if err := u.verifyItemBelongsToPlan(ctx, planID, itemID); err != nil {
		return err
	}
	t, err := time.Parse("2006-01-02", req.Tanggal)
	if err != nil {
		return fmt.Errorf("%w: invalid tanggal format", ErrMasterPlanValidation)
	}
	if _, upsErr := u.repo.UpsertTargetProses(ctx, entity.UpsertTargetProsesParams{
		IDMasterPlanItem: itemID,
		Column2:          pgtype.Date{Time: t, Valid: true},
		NamaProses:       req.NamaProses,
	}); upsErr != nil {
		return fmt.Errorf("%w: %s", ErrWorkOrderServiceUnavailable, upsErr.Error())
	}
	return nil
}

func (u *MasterPlanUseCase) DeleteTargetProses(ctx context.Context, planID, itemID int32, tanggal string) error {
	if err := u.verifyItemBelongsToPlan(ctx, planID, itemID); err != nil {
		return err
	}
	t, err := time.Parse("2006-01-02", tanggal)
	if err != nil {
		return fmt.Errorf("%w: invalid tanggal format", ErrMasterPlanValidation)
	}
	return u.repo.DeleteTargetProses(ctx, entity.DeleteTargetProsesParams{
		IDMasterPlanItem: itemID,
		Column2:          pgtype.Date{Time: t, Valid: true},
	})
}

// verifyItemBelongsToPlan checks that itemID's id_master_plan matches planID.
func (u *MasterPlanUseCase) verifyItemBelongsToPlan(ctx context.Context, planID, itemID int32) error {
	detail, err := u.repo.GetMasterPlanItemByID(ctx, itemID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrMasterPlanItemNotFound
		}
		return fmt.Errorf("%w: %s", ErrWorkOrderServiceUnavailable, err.Error())
	}
	if detail.IDMasterPlan != planID {
		return ErrMasterPlanItemNotFound
	}
	return nil
}

func (u *MasterPlanUseCase) buildItemResponse(ctx context.Context, raw masterPlanItemRaw) (*model.MasterPlanItemResponse, error) {
	itemID := raw.idMasterPlanItem

	targetRows, err := u.repo.ListTargetHarianByItem(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrWorkOrderServiceUnavailable, err.Error())
	}
	targets := make([]model.TargetHarianResponse, 0, len(targetRows))
	for _, r := range targetRows {
		targets = append(targets, model.TargetHarianResponse{
			Tanggal: r.Tanggal.Time.Format("2006-01-02"),
			Target:  r.Target,
		})
	}

	outputRows, err := u.repo.ListOutputHarianByItem(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrWorkOrderServiceUnavailable, err.Error())
	}
	outputs := make([]model.OutputHarianResponse, 0, len(outputRows))
	for _, r := range outputRows {
		outputs = append(outputs, model.OutputHarianResponse{
			Tanggal: r.Tanggal.Time.Format("2006-01-02"),
			Output:  r.Output,
		})
	}

	prosesRows, err := u.repo.ListTargetProsesByItem(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrWorkOrderServiceUnavailable, err.Error())
	}
	proses := make([]model.TargetProsesResponse, 0, len(prosesRows))
	for _, r := range prosesRows {
		proses = append(proses, model.TargetProsesResponse{
			Tanggal:    r.Tanggal.Time.Format("2006-01-02"),
			NamaProses: r.NamaProses,
		})
	}

	return &model.MasterPlanItemResponse{
		IDMasterPlanItem: raw.idMasterPlanItem,
		IDMasterPlan:     raw.idMasterPlan,
		IDWoShell:        raw.idWoShell,
		IDWo:             raw.idWo,
		NoUrut:           raw.noUrut,
		Buyer:            raw.buyer,
		Style:            raw.style,
		Qty:              raw.qty,
		Color:            raw.color,
		Deskripsi:        raw.deskripsi,
		CreatedAt:        raw.createdAt.Time.Format(time.RFC3339),
		TargetHarian:     targets,
		OutputHarian:     outputs,
		TargetProses:     proses,
	}, nil
}

func mapMasterPlanDBError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503":
			return fmt.Errorf("%w: %s", ErrMasterPlanReferenceNotFound, pgErr.Detail)
		case "23505":
			return fmt.Errorf("%w: %s", ErrMasterPlanDuplicate, pgErr.Detail)
		}
	}
	return fmt.Errorf("%w: %s", ErrWorkOrderServiceUnavailable, err.Error())
}

func intToInt32(v int) (int32, error) {
	if v < 0 || v > math.MaxInt32 {
		return 0, ErrMasterPlanValidation
	}
	return int32(v), nil
}
