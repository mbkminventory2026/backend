package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"permatatex-inventory/internal/entity"
	"permatatex-inventory/internal/model"
)

var (
	ErrMaterialListNotFound     = errors.New("material list not found")
	ErrMaterialListItemNotFound = errors.New("material list item not found")
	ErrMaterialListLocked       = errors.New("material list is locked")
	ErrMaterialListWOMismatch   = errors.New("material list does not belong to this work order")
	ErrMaterialListUnavailable  = errors.New("material list service unavailable")
	ErrMaterialListValidation   = errors.New("invalid material list payload")
)

type MaterialListUseCase struct {
	repo   entity.Querier
	dbPool *pgxpool.Pool
}

func NewMaterialListUseCase(repo entity.Querier, dbPool *pgxpool.Pool) (*MaterialListUseCase, error) {
	if repo == nil {
		return nil, errors.New("material list repository is required")
	}
	if dbPool == nil {
		return nil, errors.New("database pool is required")
	}
	return &MaterialListUseCase{repo: repo, dbPool: dbPool}, nil
}

func (u *MaterialListUseCase) CreateMaterialList(ctx context.Context, idWo int32, req model.CreateMaterialListRequest) (*model.MaterialListResponse, error) {
	ml, err := u.repo.CreateMaterialList(ctx, entity.CreateMaterialListParams{
		IDWo: idWo,
		Name: req.Name,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMaterialListUnavailable, err)
	}
	return &model.MaterialListResponse{
		ID:        ml.IDMaterialList,
		IDWo:      ml.IDWo,
		Name:      ml.Name,
		IsLocked:  ml.IsLocked,
		CreatedAt: ml.CreatedAt.Time.Format(time.RFC3339),
		Items:     []model.MaterialListItemResponse{},
	}, nil
}

func (u *MaterialListUseCase) ListByWO(ctx context.Context, idWo int32, unlockedOnly bool) (*model.MaterialListListResponse, error) {
	var rows []listedML
	if unlockedOnly {
		raw, err := u.repo.ListUnlockedMaterialListsByWO(ctx, idWo)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrMaterialListUnavailable, err)
		}
		for _, r := range raw {
			rows = append(rows, listedML{IDMaterialList: r.IDMaterialList, IDWo: r.IDWo, Name: r.Name, IsLocked: r.IsLocked, CreatedAt: r.CreatedAt})
		}
	} else {
		raw, err := u.repo.ListMaterialListsByWorkOrderID(ctx, idWo)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrMaterialListUnavailable, err)
		}
		for _, r := range raw {
			rows = append(rows, listedML{IDMaterialList: r.IDMaterialList, IDWo: r.IDWo, Name: r.Name, IsLocked: r.IsLocked, CreatedAt: r.CreatedAt})
		}
	}

	items := make([]model.MaterialListResponse, 0, len(rows))
	for _, ml := range rows {
		mliRows, err := u.repo.ListMaterialListItemsByML(ctx, ml.IDMaterialList)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrMaterialListUnavailable, err)
		}
		items = append(items, buildMLResponse(ml, mliRows))
	}
	return &model.MaterialListListResponse{Items: items}, nil
}

func (u *MaterialListUseCase) Get(ctx context.Context, id int32) (*model.MaterialListResponse, error) {
	ml, err := u.repo.GetMaterialList(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMaterialListNotFound
		}
		return nil, fmt.Errorf("%w: %v", ErrMaterialListUnavailable, err)
	}
	mliRows, err := u.repo.ListMaterialListItemsByML(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMaterialListUnavailable, err)
	}
	resp := buildMLResponse(listedML{
		IDMaterialList: ml.IDMaterialList,
		IDWo:           ml.IDWo,
		Name:           ml.Name,
		IsLocked:       ml.IsLocked,
		CreatedAt:      ml.CreatedAt,
	}, mliRows)
	return &resp, nil
}

func (u *MaterialListUseCase) Update(ctx context.Context, id int32, req model.UpdateMaterialListRequest) (*model.MaterialListResponse, error) {
	ml, err := u.repo.UpdateMaterialList(ctx, entity.UpdateMaterialListParams{
		IDMaterialList: id,
		Name:           req.Name,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, lockedOrNotFound(u.repo, ctx, id)
		}
		return nil, fmt.Errorf("%w: %v", ErrMaterialListUnavailable, err)
	}
	return u.Get(ctx, ml.IDMaterialList)
}

func (u *MaterialListUseCase) Delete(ctx context.Context, id int32) error {
	if _, err := u.repo.GetMaterialList(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrMaterialListNotFound
		}
		return fmt.Errorf("%w: %v", ErrMaterialListUnavailable, err)
	}
	// DeleteMaterialList :exec — driver returns no rows; lock guard in WHERE clause silently no-ops.
	if err := u.repo.DeleteMaterialList(ctx, id); err != nil {
		return fmt.Errorf("%w: %v", ErrMaterialListUnavailable, err)
	}
	// Verify it was actually deleted (otherwise locked).
	if _, err := u.repo.GetMaterialList(ctx, id); err == nil {
		return ErrMaterialListLocked
	}
	return nil
}

func (u *MaterialListUseCase) CreateItem(ctx context.Context, idML int32, req model.CreateMaterialListItemBody) (*model.MaterialListItemResponse, error) {
	ml, err := u.repo.GetMaterialList(ctx, idML)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMaterialListNotFound
		}
		return nil, fmt.Errorf("%w: %v", ErrMaterialListUnavailable, err)
	}
	if ml.IsLocked {
		return nil, ErrMaterialListLocked
	}

	mli, err := u.repo.CreateMaterialListItem(ctx, entity.CreateMaterialListItemParams{
		IDMaterialList: idML,
		Item:           req.Item,
		Description:    req.Description,
		Qty:            req.Qty,
		Unit:           req.Unit,
		EstPrice:       mustNumeric(req.EstPrice),
		IDWoShell:      nullableInt32Param(req.IDWoShell),
		IDWoTrim:       nullableInt32Param(req.IDWoTrim),
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMaterialListUnavailable, err)
	}

	resp := model.MaterialListItemResponse{
		ID:          mli.IDMaterialListItem,
		Item:        mli.Item,
		Description: mli.Description,
		Qty:         mli.Qty,
		Unit:        mli.Unit,
		EstPrice:    numericToFloat64(mli.EstPrice),
		CreatedAt:   mli.CreatedAt.Time.Format(time.RFC3339),
	}
	if mli.IDWoShell.Valid {
		v := mli.IDWoShell.Int32
		resp.IDWoShell = &v
	}
	if mli.IDWoTrim.Valid {
		v := mli.IDWoTrim.Int32
		resp.IDWoTrim = &v
	}
	return &resp, nil
}

func (u *MaterialListUseCase) UpdateItem(ctx context.Context, id int32, req model.UpdateMaterialListItemBody) (*model.MaterialListItemResponse, error) {
	_, err := u.repo.UpdateMaterialListItem(ctx, entity.UpdateMaterialListItemParams{
		Item:               req.Item,
		Description:        req.Description,
		Qty:                req.Qty,
		Unit:               req.Unit,
		EstPrice:           mustNumeric(req.EstPrice),
		IDWoShell:          nullableInt32Param(req.IDWoShell),
		IDWoTrim:           nullableInt32Param(req.IDWoTrim),
		IDMaterialListItem: id,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, itemLockedOrNotFound(u.repo, ctx, id)
		}
		return nil, fmt.Errorf("%w: %v", ErrMaterialListUnavailable, err)
	}

	mli, err := u.repo.GetMaterialListItem(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMaterialListUnavailable, err)
	}

	resp := model.MaterialListItemResponse{
		ID:            mli.IDMaterialListItem,
		Item:          mli.Item,
		Description:   mli.Description,
		Qty:           mli.Qty,
		Unit:          mli.Unit,
		EstPrice:      numericToFloat64(mli.EstPrice),
		CreatedAt:     mli.CreatedAt.Time.Format(time.RFC3339),
		QtySuratJalan: mli.QtySuratJalan,
		QtyReceived:   mli.QtyReceived,
	}
	if mli.IDWoShell.Valid {
		v := mli.IDWoShell.Int32
		resp.IDWoShell = &v
	}
	if mli.IDWoTrim.Valid {
		v := mli.IDWoTrim.Int32
		resp.IDWoTrim = &v
	}
	return &resp, nil
}

func (u *MaterialListUseCase) DeleteItem(ctx context.Context, id int32) error {
	existing, err := u.repo.GetMaterialListItem(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrMaterialListItemNotFound
		}
		return fmt.Errorf("%w: %v", ErrMaterialListUnavailable, err)
	}
	ml, err := u.repo.GetMaterialList(ctx, existing.IDMaterialList)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrMaterialListUnavailable, err)
	}
	if ml.IsLocked {
		return ErrMaterialListLocked
	}
	if err := u.repo.DeleteMaterialListItem(ctx, id); err != nil {
		return fmt.Errorf("%w: %v", ErrMaterialListUnavailable, err)
	}
	return nil
}

type listedML struct {
	IDMaterialList int32
	IDWo           int32
	Name           string
	IsLocked       bool
	CreatedAt      pgtype.Timestamptz
}

func buildMLResponse(ml listedML, mliRows []entity.ListMaterialListItemsByMLRow) model.MaterialListResponse {
	items := make([]model.MaterialListItemResponse, 0, len(mliRows))
	for _, ir := range mliRows {
		ri := model.MaterialListItemResponse{
			ID:            ir.IDMaterialListItem,
			Item:          ir.Item,
			Description:   ir.Description,
			Qty:           ir.Qty,
			Unit:          ir.Unit,
			EstPrice:      numericToFloat64(ir.EstPrice),
			CreatedAt:     ir.CreatedAt.Time.Format(time.RFC3339),
			QtySuratJalan: ir.QtySuratJalan,
			QtyReceived:   ir.QtyReceived,
		}
		if ir.IDWoShell.Valid {
			v := ir.IDWoShell.Int32
			ri.IDWoShell = &v
		}
		if ir.IDWoTrim.Valid {
			v := ir.IDWoTrim.Int32
			ri.IDWoTrim = &v
		}
		items = append(items, ri)
	}
	return model.MaterialListResponse{
		ID:        ml.IDMaterialList,
		IDWo:      ml.IDWo,
		Name:      ml.Name,
		IsLocked:  ml.IsLocked,
		CreatedAt: ml.CreatedAt.Time.Format(time.RFC3339),
		Items:     items,
	}
}

func lockedOrNotFound(repo entity.Querier, ctx context.Context, id int32) error {
	ml, err := repo.GetMaterialList(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrMaterialListNotFound
		}
		return fmt.Errorf("%w: %v", ErrMaterialListUnavailable, err)
	}
	if ml.IsLocked {
		return ErrMaterialListLocked
	}
	return ErrMaterialListNotFound
}

func itemLockedOrNotFound(repo entity.Querier, ctx context.Context, id int32) error {
	existing, err := repo.GetMaterialListItem(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrMaterialListItemNotFound
		}
		return fmt.Errorf("%w: %v", ErrMaterialListUnavailable, err)
	}
	ml, err := repo.GetMaterialList(ctx, existing.IDMaterialList)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrMaterialListUnavailable, err)
	}
	if ml.IsLocked {
		return ErrMaterialListLocked
	}
	return ErrMaterialListItemNotFound
}
