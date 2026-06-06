package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"permatatex-inventory/internal/entity"
	"permatatex-inventory/internal/model"
)

var (
	ErrWarehouseValidation         = errors.New("invalid warehouse payload")
	ErrWarehouseReferenceNotFound  = errors.New("related data not found")
	ErrWarehouseServiceUnavailable = errors.New("warehouse service unavailable")
	ErrWarehouseNotFound           = errors.New("warehouse transaction not found")
	ErrSuratJalanTypeUnsupported   = errors.New("unsupported surat jalan type")
	ErrWarehouseInsufficientStock  = errors.New("insufficient stock balance")

	packingListSortColumns        = buildSortWhitelist("created_at", "id_packing_list", "total_garment_per_box", "total_reject", "buyer", "model")
	suratJalanClientSortColumns   = buildSortWhitelist("created_at", "id_surat_jalan_client", "tanggal", "qty", "keterangan", "material_description", "id_wo")
	suratJalanInternalSortColumns = buildSortWhitelist("created_at", "id_surat_jalan_internal")
)

type WarehouseDeliveryUseCase struct {
	repo   entity.Querier
	dbPool *pgxpool.Pool
}

func NewWarehouseDeliveryUseCase(repo entity.Querier, dbPool *pgxpool.Pool) (*WarehouseDeliveryUseCase, error) {
	if repo == nil {
		return nil, errors.New("warehouse repository is required")
	}
	if dbPool == nil {
		return nil, errors.New("database pool is required")
	}

	return &WarehouseDeliveryUseCase{
		repo:   repo,
		dbPool: dbPool,
	}, nil
}

func (u *WarehouseDeliveryUseCase) ReceiveInventory(ctx context.Context, req model.ReceiveInventoryRequest) (*model.ReceiveInventoryResponse, error) {
	if err := validateDate(req.Tanggal); err != nil {
		return nil, ErrWarehouseValidation
	}

	item, err := u.repo.ReceiveInventory(ctx, entity.ReceiveInventoryParams{
		Tanggal:                mustDate(req.Tanggal),
		Qty:                    req.Qty,
		Keterangan:             req.Keterangan,
		IDMaterialListItem:     req.IDMaterialList,
		IDRekonsiliasiMaterial: req.IDRekonsiliasiMaterial,
	})
	if err != nil {
		return nil, mapWarehouseDBError(err)
	}

	return &model.ReceiveInventoryResponse{
		IDReceived:                   item.IDReceived,
		Tanggal:                      item.Tanggal.Time.Format("2006-01-02"),
		Qty:                          item.Qty,
		Keterangan:                   item.Keterangan,
		IDMaterialList:               item.IDMaterialList,
		IDRekonsiliasiMaterialTerima: item.IDRekonsiliasiMaterialTerima,
		IDRekonsiliasiMaterial:       item.IDRekonsiliasiMaterial,
		ActualKirim:                  item.ActualKirim,
		Balance:                      item.Balance,
		CreatedAt:                    item.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (u *WarehouseDeliveryUseCase) IssueInventory(ctx context.Context, req model.IssueInventoryRequest) (*model.IssueInventoryResponse, error) {
	current, err := u.repo.GetRekonsiliasiMaterialStock(ctx, req.IDRekonsiliasiMaterial)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWarehouseNotFound
		}
		return nil, fmt.Errorf("%w: failed to get stock balance", ErrWarehouseServiceUnavailable)
	}
	if current.Balance < req.Qty {
		return nil, ErrWarehouseInsufficientStock
	}

	item, err := u.repo.IssueInventory(ctx, entity.IssueInventoryParams{
		Qty:                    req.Qty,
		IDRekonsiliasiMaterial: req.IDRekonsiliasiMaterial,
	})
	if err != nil {
		return nil, mapWarehouseDBError(err)
	}

	return &model.IssueInventoryResponse{
		IDRekonsiliasiMaterial: item.IDRekonsiliasiMaterial,
		QtyIssued:              req.Qty,
		PreviousBalance:        item.LastBalance,
		Balance:                item.Balance,
	}, nil
}

func (u *WarehouseDeliveryUseCase) CreatePackingList(ctx context.Context, userID int32, req model.CreatePackingListRequest) (*model.PackingListResponse, error) {
	if len(req.Items) == 0 {
		return nil, ErrWarehouseValidation
	}
	for _, item := range req.Items {
		if len(item.Sizes) == 0 || item.BoxNoEnd < item.BoxNoStart {
			return nil, ErrWarehouseValidation
		}
	}

	tx, err := u.dbPool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to start transaction", ErrWarehouseServiceUnavailable)
	}
	defer func() {
		rollbackErr := tx.Rollback(ctx)
		if rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			err = rollbackErr
		}
	}()

	qtx := entity.New(tx)

	idSuratJalan := pgtype.Int4{Valid: false}
	if req.IDSuratJalanInternal != nil {
		idSuratJalan = pgtype.Int4{Int32: *req.IDSuratJalanInternal, Valid: true}
	}

	header, err := qtx.CreatePackingList(ctx, entity.CreatePackingListParams{
		TotalGarmentPerBox:   req.TotalGarmentPerBox,
		TotalReject:          req.TotalReject,
		IDWo:                 req.IDWO,
		IDSuratJalanInternal: idSuratJalan,
	})
	if err != nil {
		return nil, mapWarehouseDBError(err)
	}

	items := make([]model.PackingListItemResponse, 0, len(req.Items))
	for _, itemReq := range req.Items {
		item, itemErr := qtx.CreatePackingListItem(ctx, entity.CreatePackingListItemParams{
			IDPackingList: header.IDPackingList,
			Color:         itemReq.Color,
			QtyBox:        itemReq.QtyBox,
			QtyPerBox:     itemReq.QtyPerBox,
			BoxNoStart:    itemReq.BoxNoStart,
			BoxNoEnd:      itemReq.BoxNoEnd,
			Note:          itemReq.Note,
		})
		if itemErr != nil {
			return nil, mapWarehouseDBError(itemErr)
		}

		sizes := make([]model.PackingListItemSizeResponse, 0, len(itemReq.Sizes))
		for _, sizeReq := range itemReq.Sizes {
			size, sizeErr := qtx.CreatePackingListItemSize(ctx, entity.CreatePackingListItemSizeParams{
				Qty:               sizeReq.Qty,
				IDPackingListItem: item.IDPackingListItem,
			})
			if sizeErr != nil {
				return nil, mapWarehouseDBError(sizeErr)
			}

			sizes = append(sizes, model.PackingListItemSizeResponse{
				ID:        size.IDPackingListItemSize,
				Qty:       size.Qty,
				CreatedAt: size.CreatedAt.Time.Format(time.RFC3339),
			})
		}

		items = append(items, model.PackingListItemResponse{
			ID:         item.IDPackingListItem,
			Color:      item.Color,
			QtyBox:     item.QtyBox,
			QtyPerBox:  item.QtyPerBox,
			BoxNoStart: item.BoxNoStart,
			BoxNoEnd:   item.BoxNoEnd,
			Note:       item.Note,
			CreatedAt:  item.CreatedAt.Time.Format(time.RFC3339),
			Sizes:      sizes,
		})
	}

	// Initialize approval workflow
	if err = initializeApprovalWorkflow(ctx, qtx, "PACKING_LIST", header.IDPackingList, userID); err != nil {
		return nil, fmt.Errorf("failed to initialize approval workflow: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("%w: failed to commit transaction", ErrWarehouseServiceUnavailable)
	}

	var suratJalanPtr *int32
	if header.IDSuratJalanInternal.Valid {
		value := header.IDSuratJalanInternal.Int32
		suratJalanPtr = &value
	}

	return &model.PackingListResponse{
		ID:                   header.IDPackingList,
		TotalGarmentPerBox:   header.TotalGarmentPerBox,
		TotalReject:          header.TotalReject,
		IDWO:                 header.IDWo,
		IDSuratJalanInternal: suratJalanPtr,
		CreatedAt:            header.CreatedAt.Time.Format(time.RFC3339),
		Items:                items,
	}, nil
}

func (u *WarehouseDeliveryUseCase) CreateSuratJalan(ctx context.Context, suratJalanType string, req *model.CreateSuratJalanClientRequest) (*model.SuratJalanResponse, error) {
	switch normalizeSuratJalanType(suratJalanType) {
	case "internal":
		item, err := u.repo.CreateSuratJalanInternal(ctx)
		if err != nil {
			return nil, mapWarehouseDBError(err)
		}
		return &model.SuratJalanResponse{
			Type:         "internal",
			IDSuratJalan: item.IDSuratJalanInternal,
			CreatedAt:    item.CreatedAt.Time.Format(time.RFC3339),
		}, nil
	case "client":
		if req == nil || validateDate(req.Tanggal) != nil {
			return nil, ErrWarehouseValidation
		}
		item, err := u.repo.CreateSuratJalanClient(ctx, entity.CreateSuratJalanClientParams{
			Tanggal:            mustDate(req.Tanggal),
			Qty:                req.Qty,
			Keterangan:         req.Keterangan,
			IDMaterialListItem: req.IDMaterialList,
		})
		if err != nil {
			return nil, mapWarehouseDBError(err)
		}
		return &model.SuratJalanResponse{
			Type:           "client",
			IDSuratJalan:   item.IDSuratJalanClient,
			Tanggal:        item.Tanggal.Time.Format("2006-01-02"),
			Qty:            item.Qty,
			Keterangan:     item.Keterangan,
			IDMaterialList: item.IDMaterialList,
			CreatedAt:      item.CreatedAt.Time.Format(time.RFC3339),
		}, nil
	default:
		return nil, ErrSuratJalanTypeUnsupported
	}
}

func (u *WarehouseDeliveryUseCase) ListPackingLists(ctx context.Context, filter model.TransactionListFilter) (*model.PackingListListResponse, error) {
	page, limit, offset, search, sortBy, sortDesc := normalizeListFilter(filter.ListQueryFilter, "id_packing_list", true, packingListSortColumns)
	rows, err := u.repo.ListPackingLists(ctx, entity.ListPackingListsParams{
		SearchTerm: search,
		IDMitra:    nullableInt32Param(filter.IDMitra),
		SortBy:     sortBy,
		SortDesc:   sortDesc,
		PageLimit:  limit,
		PageOffset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: failed to list packing lists", ErrWarehouseServiceUnavailable)
	}

	items := make([]model.PackingListListItem, 0, len(rows))
	total := int64(0)
	for _, row := range rows {
		total = row.TotalCount
		var suratJalanPtr *int32
		if row.IDSuratJalanInternal.Valid {
			value := row.IDSuratJalanInternal.Int32
			suratJalanPtr = &value
		}

		items = append(items, model.PackingListListItem{
			ID:                   row.IDPackingList,
			TotalGarmentPerBox:   row.TotalGarmentPerBox,
			TotalReject:          row.TotalReject,
			IDWO:                 row.IDWo,
			IDSuratJalanInternal: suratJalanPtr,
			Buyer:                row.Buyer,
			Model:                row.Model,
			CreatedAt:            row.CreatedAt.Time.Format(time.RFC3339),
		})
	}

	return &model.PackingListListResponse{
		Items:      items,
		Pagination: buildPagination(total, page, limit),
	}, nil
}

func (u *WarehouseDeliveryUseCase) GetPackingListDetail(ctx context.Context, id int32, idMitra *int32) (*model.PackingListDetailResponse, error) {
	header, err := u.repo.GetPackingListDetail(ctx, entity.GetPackingListDetailParams{
		IDPackingList: id,
		IDMitra:       nullableInt32Param(idMitra),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWarehouseNotFound
		}
		return nil, fmt.Errorf("%w: failed to get packing list", ErrWarehouseServiceUnavailable)
	}

	itemRows, err := u.repo.ListPackingListItemsByPackingListID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get packing list items", ErrWarehouseServiceUnavailable)
	}
	sizeRows, err := u.repo.ListPackingListItemSizesByPackingListID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get packing list item sizes", ErrWarehouseServiceUnavailable)
	}

	sizeMap := make(map[int32][]model.PackingListItemSizeResponse)
	for _, row := range sizeRows {
		sizeMap[row.IDPackingListItem] = append(sizeMap[row.IDPackingListItem], model.PackingListItemSizeResponse{
			ID:        row.IDPackingListItemSize,
			Qty:       row.Qty,
			CreatedAt: row.CreatedAt.Time.Format(time.RFC3339),
		})
	}

	items := make([]model.PackingListItemResponse, 0, len(itemRows))
	for _, row := range itemRows {
		items = append(items, model.PackingListItemResponse{
			ID:         row.IDPackingListItem,
			Color:      row.Color,
			QtyBox:     row.QtyBox,
			QtyPerBox:  row.QtyPerBox,
			BoxNoStart: row.BoxNoStart,
			BoxNoEnd:   row.BoxNoEnd,
			Note:       row.Note,
			CreatedAt:  row.CreatedAt.Time.Format(time.RFC3339),
			Sizes:      sizeMap[row.IDPackingListItem],
		})
	}

	var suratJalanPtr *int32
	if header.IDSuratJalanInternal.Valid {
		value := header.IDSuratJalanInternal.Int32
		suratJalanPtr = &value
	}

	return &model.PackingListDetailResponse{
		ID:                   header.IDPackingList,
		TotalGarmentPerBox:   header.TotalGarmentPerBox,
		TotalReject:          header.TotalReject,
		IDWO:                 header.IDWo,
		IDSuratJalanInternal: suratJalanPtr,
		Buyer:                header.Buyer,
		Model:                header.Model,
		CreatedAt:            header.CreatedAt.Time.Format(time.RFC3339),
		Items:                items,
	}, nil
}

func (u *WarehouseDeliveryUseCase) ListSuratJalanClients(ctx context.Context, filter model.TransactionListFilter) (*model.SuratJalanClientListResponse, error) {
	page, limit, offset, search, sortBy, sortDesc := normalizeListFilter(filter.ListQueryFilter, "id_surat_jalan_client", true, suratJalanClientSortColumns)
	rows, err := u.repo.ListSuratJalanClients(ctx, entity.ListSuratJalanClientsParams{
		SearchTerm: search,
		IDMitra:    nullableInt32Param(filter.IDMitra),
		SortBy:     sortBy,
		SortDesc:   sortDesc,
		PageLimit:  limit,
		PageOffset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: failed to list surat jalan clients", ErrWarehouseServiceUnavailable)
	}

	items := make([]model.SuratJalanClientListItem, 0, len(rows))
	total := int64(0)
	for _, row := range rows {
		total = row.TotalCount
		items = append(items, model.SuratJalanClientListItem{
			ID:                  row.IDSuratJalanClient,
			Tanggal:             row.Tanggal.Time.Format("2006-01-02"),
			Qty:                 row.Qty,
			Keterangan:          row.Keterangan,
			IDMaterialList:      row.IDMaterialList,
			MaterialDescription: row.MaterialDescription,
			IDWO:                row.IDWo,
			CreatedAt:           row.CreatedAt.Time.Format(time.RFC3339),
		})
	}

	return &model.SuratJalanClientListResponse{
		Items:      items,
		Pagination: buildPagination(total, page, limit),
	}, nil
}

func (u *WarehouseDeliveryUseCase) GetSuratJalanClientDetail(ctx context.Context, id int32, idMitra *int32) (*model.SuratJalanClientDetailResponse, error) {
	row, err := u.repo.GetSuratJalanClientDetail(ctx, entity.GetSuratJalanClientDetailParams{
		IDSuratJalanClient: id,
		IDMitra:            nullableInt32Param(idMitra),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWarehouseNotFound
		}
		return nil, fmt.Errorf("%w: failed to get surat jalan client", ErrWarehouseServiceUnavailable)
	}

	return &model.SuratJalanClientDetailResponse{
		ID:                  row.IDSuratJalanClient,
		Tanggal:             row.Tanggal.Time.Format("2006-01-02"),
		Qty:                 row.Qty,
		Keterangan:          row.Keterangan,
		IDMaterialList:      row.IDMaterialList,
		MaterialDescription: row.MaterialDescription,
		IDWO:                row.IDWo,
		CreatedAt:           row.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (u *WarehouseDeliveryUseCase) ListSuratJalanInternals(ctx context.Context, filter model.TransactionListFilter) (*model.SuratJalanInternalListResponse, error) {
	page, limit, offset, _, sortBy, sortDesc := normalizeListFilter(filter.ListQueryFilter, "id_surat_jalan_internal", true, suratJalanInternalSortColumns)
	rows, err := u.repo.ListSuratJalanInternals(ctx, entity.ListSuratJalanInternalsParams{
		SortBy:     sortBy,
		SortDesc:   sortDesc,
		PageLimit:  limit,
		PageOffset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: failed to list surat jalan internals", ErrWarehouseServiceUnavailable)
	}

	items := make([]model.SuratJalanInternalListItem, 0, len(rows))
	total := int64(0)
	for _, row := range rows {
		total = row.TotalCount
		items = append(items, model.SuratJalanInternalListItem{
			ID:        row.IDSuratJalanInternal,
			CreatedAt: row.CreatedAt.Time.Format(time.RFC3339),
		})
	}

	return &model.SuratJalanInternalListResponse{
		Items:      items,
		Pagination: buildPagination(total, page, limit),
	}, nil
}

func (u *WarehouseDeliveryUseCase) GetSuratJalanInternalDetail(ctx context.Context, id int32) (*model.SuratJalanInternalDetailResponse, error) {
	row, err := u.repo.GetSuratJalanInternalDetail(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWarehouseNotFound
		}
		return nil, fmt.Errorf("%w: failed to get surat jalan internal", ErrWarehouseServiceUnavailable)
	}

	return &model.SuratJalanInternalDetailResponse{
		ID:        row.IDSuratJalanInternal,
		CreatedAt: row.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func mapWarehouseDBError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503":
			return ErrWarehouseReferenceNotFound
		}
	}
	return fmt.Errorf("%w: %v", ErrWarehouseServiceUnavailable, err)
}

func normalizeSuratJalanType(value string) string {
	normalized := strings.TrimSpace(strings.ToLower(value))
	normalized = strings.ReplaceAll(normalized, "_", "-")
	return normalized
}
