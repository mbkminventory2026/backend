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
	return createMaterialListItem(ctx, u.repo, idML, req)
}

// materialListItemCreateRepository is deliberately the narrow create surface
// shared by the ordinary CRUD path and the transactional XLSX importer.
type materialListItemCreateRepository interface {
	GetMaterialList(context.Context, int32) (entity.GetMaterialListRow, error)
	CreateMaterialListItem(context.Context, entity.CreateMaterialListItemParams) (entity.CreateMaterialListItemRow, error)
	materialListApplicabilityRepository
	materialListSourceRepository
	materialListConsumptionPrefillRepository
}

func createMaterialListItem(ctx context.Context, repo materialListItemCreateRepository, idML int32, req model.CreateMaterialListItemBody) (*model.MaterialListItemResponse, error) {
	return createMaterialListItemWithNumeric(ctx, repo, idML, req, mustNumeric(req.EstPrice), nil)
}

// createMaterialListItemWithNumeric retains the ordinary CreateItem validation,
// source/applicability checks, and prefill behavior while permitting import to
// preserve DECIMAL input without routing it through float64.
func createMaterialListItemWithNumeric(ctx context.Context, repo materialListItemCreateRepository, idML int32, req model.CreateMaterialListItemBody, estPrice pgtype.Numeric, explicitCons *pgtype.Numeric) (*model.MaterialListItemResponse, error) {
	ml, err := repo.GetMaterialList(ctx, idML)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMaterialListNotFound
		}
		return nil, fmt.Errorf("%w: %v", ErrMaterialListUnavailable, err)
	}
	if ml.IsLocked {
		return nil, ErrMaterialListLocked
	}
	if err := validateMaterialListItemApplicability(ctx, repo, idML, materialListItemApplicability{
		Category:     req.Category,
		QtyWoScope:   req.QtyWoScope,
		IDQtyWoShell: req.IDQtyWoShell,
		IDQtyWoSize:  req.IDQtyWoSize,
	}); err != nil {
		return nil, err
	}
	idWoShell := nullableInt32Param(req.IDWoShell)
	idWoTrim := nullableInt32Param(req.IDWoTrim)
	if err := validateMaterialListItemSources(ctx, repo, idML, idWoShell, idWoTrim); err != nil {
		return nil, err
	}
	var consPerPC pgtype.Numeric
	if explicitCons != nil {
		consPerPC = *explicitCons
	} else {
		var err error
		consPerPC, err = materialListConsPerPCForCreate(ctx, repo, req.ConsPerPC, idWoShell, idWoTrim)
		if err != nil {
			return nil, err
		}
	}

	mli, err := repo.CreateMaterialListItem(ctx, entity.CreateMaterialListItemParams{
		IDMaterialList: idML,
		Item:           req.Item,
		Description:    req.Description,
		Qty:            req.Qty,
		Unit:           req.Unit,
		EstPrice:       estPrice,
		IDWoShell:      idWoShell,
		IDWoTrim:       idWoTrim,
		Category:       nullableTextParam(req.Category),
		ConsPerPc:      consPerPC,
		QtyWoScope:     nullableTextParam(req.QtyWoScope),
		IDQtyWoShell:   nullableInt32Param(req.IDQtyWoShell),
		IDQtyWoSize:    nullableInt32Param(req.IDQtyWoSize),
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMaterialListUnavailable, err)
	}

	resp := materialListItemResponse(mli.IDMaterialListItem, mli.Item, mli.Description, mli.Qty, mli.Unit, mli.EstPrice, mli.IDWoShell, mli.IDWoTrim, mli.Category, mli.ConsPerPc, mli.QtyWoScope, mli.IDQtyWoShell, mli.IDQtyWoSize, mli.CreatedAt, 0, 0)
	return &resp, nil
}

func (u *MaterialListUseCase) UpdateItem(ctx context.Context, id int32, req model.UpdateMaterialListItemBody) (*model.MaterialListItemResponse, error) {
	existing, err := u.repo.GetMaterialListItem(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMaterialListItemNotFound
		}
		return nil, fmt.Errorf("%w: %v", ErrMaterialListUnavailable, err)
	}
	ml, err := u.repo.GetMaterialList(ctx, existing.IDMaterialList)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMaterialListUnavailable, err)
	}
	if ml.IsLocked {
		return nil, ErrMaterialListLocked
	}
	if err := validateUpdatedConsPerPC(req.ConsPerPC); err != nil {
		return nil, err
	}
	if err := validateMaterialListItemSources(ctx, u.repo, existing.IDMaterialList, nullableInt32Param(req.IDWoShell), nullableInt32Param(req.IDWoTrim)); err != nil {
		return nil, err
	}
	if err := validateMaterialListItemApplicability(ctx, u.repo, existing.IDMaterialList, materialListItemApplicability{
		Category:     optionalStringValue(existing.Category, req.Category),
		QtyWoScope:   optionalStringValue(existing.QtyWoScope, req.QtyWoScope),
		IDQtyWoShell: optionalInt32Value(existing.IDQtyWoShell, req.IDQtyWoShell),
		IDQtyWoSize:  optionalInt32Value(existing.IDQtyWoSize, req.IDQtyWoSize),
	}); err != nil {
		return nil, err
	}

	_, err = u.repo.UpdateMaterialListItem(ctx, entity.UpdateMaterialListItemParams{
		Item:               req.Item,
		Description:        req.Description,
		Qty:                req.Qty,
		Unit:               req.Unit,
		EstPrice:           mustNumeric(req.EstPrice),
		IDWoShell:          nullableInt32Param(req.IDWoShell),
		IDWoTrim:           nullableInt32Param(req.IDWoTrim),
		SetCategory:        req.Category.Set,
		Category:           nullableTextParam(req.Category.Value),
		SetConsPerPc:       req.ConsPerPC.Set,
		ConsPerPc:          materialListNumericPtr(req.ConsPerPC.Value),
		SetQtyWoScope:      req.QtyWoScope.Set,
		QtyWoScope:         nullableTextParam(req.QtyWoScope.Value),
		SetIDQtyWoShell:    req.IDQtyWoShell.Set,
		IDQtyWoShell:       nullableInt32Param(req.IDQtyWoShell.Value),
		SetIDQtyWoSize:     req.IDQtyWoSize.Set,
		IDQtyWoSize:        nullableInt32Param(req.IDQtyWoSize.Value),
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
	resp := materialListItemResponse(mli.IDMaterialListItem, mli.Item, mli.Description, mli.Qty, mli.Unit, mli.EstPrice, mli.IDWoShell, mli.IDWoTrim, mli.Category, mli.ConsPerPc, mli.QtyWoScope, mli.IDQtyWoShell, mli.IDQtyWoSize, mli.CreatedAt, mli.QtySuratJalan, mli.QtyReceived)
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
		ri := materialListItemResponse(ir.IDMaterialListItem, ir.Item, ir.Description, ir.Qty, ir.Unit, ir.EstPrice, ir.IDWoShell, ir.IDWoTrim, ir.Category, ir.ConsPerPc, ir.QtyWoScope, ir.IDQtyWoShell, ir.IDQtyWoSize, ir.CreatedAt, ir.QtySuratJalan, ir.QtyReceived)
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

func (u *MaterialListUseCase) ListMaterialListsPaginated(ctx context.Context, search string, lockedOnly bool, limit, offset int32) (*model.MaterialListPageResponse, error) {
	rows, err := u.repo.ListMaterialListsPaginated(ctx, entity.ListMaterialListsPaginatedParams{
		LockedOnly: lockedOnly,
		Search:     search,
		Lim:        limit,
		Off:        offset,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMaterialListUnavailable, err)
	}
	items := make([]model.MaterialListPageItem, 0, len(rows))
	total := int64(0)
	for _, row := range rows {
		total = row.TotalCount
		items = append(items, model.MaterialListPageItem{
			IDMaterialList:   row.IDMaterialList,
			IDWo:             row.IDWo,
			Name:             row.Name,
			IsLocked:         row.IsLocked,
			CreatedAt:        row.CreatedAt.Time.Format(time.RFC3339),
			Buyer:            row.Buyer,
			Model:            row.Model,
			WoQty:            row.WoQty,
			ItemCount:        row.ItemCount,
			TotalQtySj:       row.TotalQtySj,
			TotalQtyReceived: row.TotalQtyReceived,
		})
	}
	page := int32(1)
	if limit > 0 {
		page = offset/limit + 1
	}
	return &model.MaterialListPageResponse{
		Items:      items,
		Pagination: buildPagination(total, page, limit),
	}, nil
}

func (u *MaterialListUseCase) GetItemDetail(ctx context.Context, id int32) (*model.MaterialListItemDetailResponse, error) {
	row, err := u.repo.GetMaterialListItemDetail(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMaterialListItemNotFound
		}
		return nil, fmt.Errorf("%w: %v", ErrMaterialListUnavailable, err)
	}
	return &model.MaterialListItemDetailResponse{
		IDMaterialListItem: row.IDMaterialListItem,
		IDMaterialList:     row.IDMaterialList,
		Item:               row.Item,
		Description:        row.Description,
		Qty:                row.Qty,
		Unit:               row.Unit,
		EstPrice:           numericToFloat64(row.EstPrice),
		Category:           nullableTextPtr(row.Category),
		ConsPerPC:          numericToFloat64Ptr(row.ConsPerPc),
		QtyWoScope:         nullableTextPtr(row.QtyWoScope),
		IDQtyWoShell:       nullableInt32PgTypePtr(row.IDQtyWoShell),
		IDQtyWoSize:        nullableInt32PgTypePtr(row.IDQtyWoSize),
		CreatedAt:          row.CreatedAt.Time.Format(time.RFC3339),
		QtySuratJalan:      row.QtySuratJalan,
		QtyReceived:        row.QtyReceived,
		MlName:             row.MlName,
		MlIsLocked:         row.MlIsLocked,
		IDWo:               row.IDWo,
		Buyer:              row.Buyer,
		Model:              row.Model,
	}, nil
}
