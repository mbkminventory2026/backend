package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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
	ErrSuratJalanExceedsMLIQty     = errors.New("surat jalan qty would exceed material list item qty")
	ErrReceivedExceedsSuratJalan   = errors.New("received qty would exceed total surat jalan qty")

	packingListSortColumns        = buildSortWhitelist("created_at", "id_packing_list", "total_garment_per_box", "total_reject", "buyer", "model")
	suratJalanClientSortColumns   = buildSortWhitelist("created_at", "id_surat_jalan_client", "tanggal", "qty", "keterangan", "material_description", "id_wo")
	suratJalanInternalSortColumns = buildSortWhitelist("created_at", "id_surat_jalan_internal")
)

type WarehouseDeliveryUseCase struct {
	repo     entity.Querier
	dbPool   *pgxpool.Pool
	auditLog *AuditLogUseCase
}

func NewWarehouseDeliveryUseCase(repo entity.Querier, dbPool *pgxpool.Pool, auditLog *AuditLogUseCase) (*WarehouseDeliveryUseCase, error) {
	if repo == nil {
		return nil, errors.New("warehouse repository is required")
	}
	if dbPool == nil {
		return nil, errors.New("database pool is required")
	}

	return &WarehouseDeliveryUseCase{
		repo:     repo,
		dbPool:   dbPool,
		auditLog: auditLog,
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
		IDMaterialListItem:     req.IDMaterialListItem,
		IDRekonsiliasiMaterial: req.IDRekonsiliasiMaterial,
	})
	if err != nil {
		return nil, mapWarehouseDBError(err)
	}

	result := &model.ReceiveInventoryResponse{
		IDReceived:                   item.IDReceived,
		Tanggal:                      item.Tanggal.Time.Format("2006-01-02"),
		Qty:                          item.Qty,
		Keterangan:                   item.Keterangan,
		IDMaterialListItem:           item.IDMaterialList,
		IDRekonsiliasiMaterialTerima: item.IDRekonsiliasiMaterialTerima,
		IDRekonsiliasiMaterial:       item.IDRekonsiliasiMaterial,
		ActualKirim:                  item.ActualKirim,
		Balance:                      item.Balance,
		CreatedAt:                    item.CreatedAt.Time.Format(time.RFC3339),
	}

	u.recordWarehouseAudit(
		ctx,
		"CREATE",
		"warehouse-delivery",
		"inventory_receipts",
		fmt.Sprintf("%d", result.IDReceived),
		fmt.Sprintf("Inventory Receipt #%d", result.IDReceived),
		nil,
		buildReceiveInventoryAuditSnapshot(result),
	)

	return result, nil
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

	result := &model.IssueInventoryResponse{
		IDRekonsiliasiMaterial: item.IDRekonsiliasiMaterial,
		QtyIssued:              req.Qty,
		PreviousBalance:        item.LastBalance,
		Balance:                item.Balance,
	}

	u.recordWarehouseAudit(
		ctx,
		"UPDATE",
		"warehouse-delivery",
		"rekonsiliasi_materials",
		fmt.Sprintf("%d", result.IDRekonsiliasiMaterial),
		fmt.Sprintf("Rekonsiliasi Material #%d", result.IDRekonsiliasiMaterial),
		buildIssueInventoryBeforeSnapshot(current.Balance),
		buildIssueInventoryAuditSnapshot(result),
	)

	return result, nil
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
				IDWoShellSize:     sizeReq.IDWOShellSize,
			})
			if sizeErr != nil {
				return nil, mapWarehouseDBError(sizeErr)
			}

			sizes = append(sizes, model.PackingListItemSizeResponse{
				ID:            size.IDPackingListItemSize,
				IDWOShellSize: size.IDWoShellSize,
				IDSize:        nil,
				Size:          "",
				Qty:           size.Qty,
				CreatedAt:     size.CreatedAt.Time.Format(time.RFC3339),
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

	rejectSizes := make([]model.PackingListRejectSizeResponse, 0, len(req.RejectSizes))
	for _, rejectSizeReq := range req.RejectSizes {
		rejectSize, rejectSizeErr := qtx.CreatePackingListRejectSize(ctx, entity.CreatePackingListRejectSizeParams{
			Qty:           rejectSizeReq.Qty,
			IDPackingList: header.IDPackingList,
			IDWoShellSize: rejectSizeReq.IDWOShellSize,
		})
		if rejectSizeErr != nil {
			return nil, mapWarehouseDBError(rejectSizeErr)
		}

		rejectSizes = append(rejectSizes, model.PackingListRejectSizeResponse{
			ID:            rejectSize.IDPackingListRejectSize,
			IDWOShellSize: rejectSize.IDWoShellSize,
			IDSize:        nil,
			Size:          "",
			Qty:           rejectSize.Qty,
			CreatedAt:     rejectSize.CreatedAt.Time.Format(time.RFC3339),
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

	result := &model.PackingListResponse{
		ID:                   header.IDPackingList,
		TotalGarmentPerBox:   header.TotalGarmentPerBox,
		TotalReject:          header.TotalReject,
		IDWO:                 header.IDWo,
		IDSuratJalanInternal: suratJalanPtr,
		CreatedAt:            header.CreatedAt.Time.Format(time.RFC3339),
		Items:                items,
		RejectSizes:          rejectSizes,
	}

	u.recordWarehouseAudit(
		ctx,
		"CREATE",
		"warehouse-delivery",
		"packing_lists",
		fmt.Sprintf("%d", result.ID),
		fmt.Sprintf("Packing List #%d", result.ID),
		nil,
		buildPackingListAuditSnapshot(result),
	)

	return result, nil
}

func (u *WarehouseDeliveryUseCase) CreateSuratJalan(ctx context.Context, suratJalanType string, req *model.CreateSuratJalanClientRequest) (*model.SuratJalanResponse, error) {
	switch normalizeSuratJalanType(suratJalanType) {
	case "internal":
		item, err := u.repo.CreateSuratJalanInternal(ctx, entity.CreateSuratJalanInternalParams{
			NoDokumen: "",
			Deskripsi: "",
		})
		if err != nil {
			return nil, mapWarehouseDBError(err)
		}
		result := &model.SuratJalanResponse{
			Type:         "internal",
			IDSuratJalan: item.IDSuratJalanInternal,
			CreatedAt:    item.CreatedAt.Time.Format(time.RFC3339),
		}
		u.recordWarehouseAudit(
			ctx,
			"CREATE",
			"warehouse-delivery",
			"surat_jalan_internals",
			fmt.Sprintf("%d", result.IDSuratJalan),
			fmt.Sprintf("Surat Jalan Internal #%d", result.IDSuratJalan),
			nil,
			buildSuratJalanAuditSnapshot(result),
		)
		return result, nil
	case "client":
		if req == nil || validateDate(req.Tanggal) != nil {
			return nil, ErrWarehouseValidation
		}
		_, err := u.repo.GetMaterialListItem(ctx, req.IDMaterialListItem)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrWarehouseReferenceNotFound
			}
			return nil, fmt.Errorf("%w: failed to get material list item", ErrWarehouseServiceUnavailable)
		}
		item, err := u.repo.CreateSuratJalanClient(ctx, entity.CreateSuratJalanClientParams{
			Tanggal:            mustDate(req.Tanggal),
			Qty:                req.Qty,
			Keterangan:         req.Keterangan,
			IDMaterialListItem: req.IDMaterialListItem,
		})
		if err != nil {
			return nil, mapWarehouseDBError(err)
		}
		result := &model.SuratJalanResponse{
			Type:               "client",
			IDSuratJalan:       item.IDSuratJalanClient,
			Tanggal:            item.Tanggal.Time.Format("2006-01-02"),
			Qty:                item.Qty,
			Keterangan:         item.Keterangan,
			IDMaterialListItem: item.IDMaterialList,
			CreatedAt:          item.CreatedAt.Time.Format(time.RFC3339),
		}
		u.recordWarehouseAudit(
			ctx,
			"CREATE",
			"warehouse-delivery",
			"surat_jalan_clients",
			fmt.Sprintf("%d", result.IDSuratJalan),
			fmt.Sprintf("Surat Jalan Client #%d", result.IDSuratJalan),
			nil,
			buildSuratJalanAuditSnapshot(result),
		)
		return result, nil
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
		idSize := row.IDSize
		sizeMap[row.IDPackingListItem] = append(sizeMap[row.IDPackingListItem], model.PackingListItemSizeResponse{
			ID:            row.IDPackingListItemSize,
			IDWOShellSize: row.IDWoShellSize,
			IDSize:        &idSize,
			Size:          row.Size,
			Qty:           row.Qty,
			CreatedAt:     row.CreatedAt.Time.Format(time.RFC3339),
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

	rejectRows, err := u.repo.ListPackingListRejectSizesByPackingListID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get packing list reject sizes", ErrWarehouseServiceUnavailable)
	}

	rejectSizes := make([]model.PackingListRejectSizeResponse, 0, len(rejectRows))
	for _, row := range rejectRows {
		idSize := row.IDSize
		rejectSizes = append(rejectSizes, model.PackingListRejectSizeResponse{
			ID:            row.IDPackingListRejectSize,
			IDWOShellSize: row.IDWoShellSize,
			IDSize:        &idSize,
			Size:          row.Size,
			Qty:           row.Qty,
			CreatedAt:     row.CreatedAt.Time.Format(time.RFC3339),
		})
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
		RejectSizes:          rejectSizes,
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
			IDMaterialListItem:  row.IDMaterialList,
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
		IDMaterialListItem:  row.IDMaterialList,
		MaterialDescription: row.MaterialDescription,
		IDWO:                row.IDWo,
		CreatedAt:           row.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (u *WarehouseDeliveryUseCase) ListSuratJalanInternals(ctx context.Context, filter model.TransactionListFilter) (*model.SuratJalanInternalListResponse, error) {
	return u.ListSuratJalanInternalsEnriched(ctx, filter)
}

func (u *WarehouseDeliveryUseCase) GetSuratJalanInternalDetail(ctx context.Context, id int32) (*model.SuratJalanInternalDetailResponse, error) {
	return u.GetSuratJalanInternalWithData(ctx, id)
}

func (u *WarehouseDeliveryUseCase) CreateSimpleReceived(ctx context.Context, req model.CreateSimpleReceivedRequest) (*model.SimpleReceivedResponse, error) {
	if err := validateDate(req.Tanggal); err != nil {
		return nil, ErrWarehouseValidation
	}
	mli, err := u.repo.GetMaterialListItem(ctx, req.IDMaterialListItem)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWarehouseReferenceNotFound
		}
		return nil, fmt.Errorf("%w: failed to get material list item", ErrWarehouseServiceUnavailable)
	}
	if mli.QtySuratJalan == 0 {
		return nil, ErrReceivedExceedsSuratJalan
	}
	if mli.QtyReceived+req.Qty > mli.QtySuratJalan {
		return nil, ErrReceivedExceedsSuratJalan
	}
	row, err := u.repo.CreateReceivedSimple(ctx, entity.CreateReceivedSimpleParams{
		Tanggal:            mustDate(req.Tanggal),
		Qty:                req.Qty,
		Keterangan:         req.Keterangan,
		IDMaterialListItem: req.IDMaterialListItem,
	})
	if err != nil {
		return nil, mapWarehouseDBError(err)
	}
	result := &model.SimpleReceivedResponse{
		IDReceived:         row.IDReceived,
		Tanggal:            row.Tanggal.Time.Format("2006-01-02"),
		Qty:                row.Qty,
		Keterangan:         row.Keterangan,
		IDMaterialListItem: row.IDMaterialListItem,
		CreatedAt:          row.CreatedAt.Time.Format(time.RFC3339),
	}

	u.recordWarehouseAudit(
		ctx,
		"CREATE",
		"warehouse-delivery",
		"received",
		fmt.Sprintf("%d", result.IDReceived),
		fmt.Sprintf("Received #%d", result.IDReceived),
		nil,
		buildSimpleReceivedAuditSnapshot(result),
	)

	return result, nil
}

func (u *WarehouseDeliveryUseCase) ListReceived(ctx context.Context, search string, limit, offset int32) (*model.SimpleReceivedListResponse, error) {
	rows, err := u.repo.ListReceived(ctx, entity.ListReceivedParams{
		Search: search,
		Lim:    limit,
		Off:    offset,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: failed to list received", ErrWarehouseServiceUnavailable)
	}
	items := make([]model.SimpleReceivedListItem, 0, len(rows))
	total := int64(0)
	for _, row := range rows {
		total = row.TotalCount
		items = append(items, model.SimpleReceivedListItem{
			IDReceived:          row.IDReceived,
			Tanggal:             row.Tanggal.Time.Format("2006-01-02"),
			Qty:                 row.Qty,
			Keterangan:          row.Keterangan,
			IDMaterialListItem:  row.IDMaterialListItem,
			MaterialItem:        row.MaterialItem,
			MaterialDescription: row.MaterialDescription,
			IDWO:                row.IDWo,
			CreatedAt:           row.CreatedAt.Time.Format(time.RFC3339),
		})
	}
	page := int32(1)
	if limit > 0 {
		page = offset/limit + 1
	}
	return &model.SimpleReceivedListResponse{
		Items:      items,
		Pagination: buildPagination(total, page, limit),
	}, nil
}

func (u *WarehouseDeliveryUseCase) GetReceivedByID(ctx context.Context, id int32) (*model.SimpleReceivedDetailResponse, error) {
	row, err := u.repo.GetReceivedByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWarehouseNotFound
		}
		return nil, fmt.Errorf("%w: failed to get received", ErrWarehouseServiceUnavailable)
	}
	return &model.SimpleReceivedDetailResponse{
		IDReceived:          row.IDReceived,
		Tanggal:             row.Tanggal.Time.Format("2006-01-02"),
		Qty:                 row.Qty,
		Keterangan:          row.Keterangan,
		IDMaterialListItem:  row.IDMaterialListItem,
		MaterialItem:        row.MaterialItem,
		MaterialDescription: row.MaterialDescription,
		IDWO:                row.IDWo,
		CreatedAt:           row.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (u *WarehouseDeliveryUseCase) UpdateSimpleReceived(ctx context.Context, id int32, req model.UpdateSimpleReceivedRequest) (*model.SimpleReceivedResponse, error) {
	if err := validateDate(req.Tanggal); err != nil {
		return nil, ErrWarehouseValidation
	}
	beforeItem, err := u.GetReceivedByID(ctx, id)
	if err != nil {
		return nil, err
	}
	row, err := u.repo.UpdateReceivedSimple(ctx, entity.UpdateReceivedSimpleParams{
		IDReceived: id,
		Tanggal:    mustDate(req.Tanggal),
		Qty:        req.Qty,
		Keterangan: req.Keterangan,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWarehouseNotFound
		}
		return nil, mapWarehouseDBError(err)
	}
	result := &model.SimpleReceivedResponse{
		IDReceived:         row.IDReceived,
		Tanggal:            row.Tanggal.Time.Format("2006-01-02"),
		Qty:                row.Qty,
		Keterangan:         row.Keterangan,
		IDMaterialListItem: row.IDMaterialListItem,
		CreatedAt:          row.CreatedAt.Time.Format(time.RFC3339),
	}

	u.recordWarehouseAudit(
		ctx,
		"UPDATE",
		"warehouse-delivery",
		"received",
		fmt.Sprintf("%d", result.IDReceived),
		fmt.Sprintf("Received #%d", result.IDReceived),
		buildSimpleReceivedDetailAuditSnapshot(beforeItem),
		buildSimpleReceivedAuditSnapshot(result),
	)

	return result, nil
}

func (u *WarehouseDeliveryUseCase) DeleteSimpleReceived(ctx context.Context, id int32) error {
	beforeItem, err := u.GetReceivedByID(ctx, id)
	if err != nil {
		return err
	}
	if err := u.repo.DeleteReceivedSimple(ctx, id); err != nil {
		return err
	}
	u.recordWarehouseAudit(
		ctx,
		"DELETE",
		"warehouse-delivery",
		"received",
		fmt.Sprintf("%d", id),
		fmt.Sprintf("Received #%d", id),
		buildSimpleReceivedDetailAuditSnapshot(beforeItem),
		nil,
	)
	return nil
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

func (u *WarehouseDeliveryUseCase) GetMLIHistory(ctx context.Context, idMLI int32) (*model.MLIHistoryResponse, error) {
	sjRows, err := u.repo.ListSuratJalanClientByMLI(ctx, idMLI)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to list surat jalan", ErrWarehouseServiceUnavailable)
	}
	recvRows, err := u.repo.ListReceivedByMLI(ctx, idMLI)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to list received", ErrWarehouseServiceUnavailable)
	}

	sj := make([]model.MLIHistoryEntry, 0, len(sjRows))
	for _, r := range sjRows {
		sj = append(sj, model.MLIHistoryEntry{
			ID:         r.IDSuratJalanClient,
			Tanggal:    r.Tanggal.Time.Format("2006-01-02"),
			Qty:        r.Qty,
			Keterangan: r.Keterangan,
			CreatedAt:  r.CreatedAt.Time.Format(time.RFC3339),
		})
	}
	recv := make([]model.MLIHistoryEntry, 0, len(recvRows))
	for _, r := range recvRows {
		recv = append(recv, model.MLIHistoryEntry{
			ID:         r.IDReceived,
			Tanggal:    r.Tanggal.Time.Format("2006-01-02"),
			Qty:        r.Qty,
			Keterangan: r.Keterangan,
			CreatedAt:  r.CreatedAt.Time.Format(time.RFC3339),
		})
	}
	return &model.MLIHistoryResponse{SuratJalan: sj, Received: recv}, nil
}

func (u *WarehouseDeliveryUseCase) DeleteSuratJalanClient(ctx context.Context, id int32) error {
	beforeItem, err := u.GetSuratJalanClientDetail(ctx, id, nil)
	if err != nil {
		return err
	}
	if err := u.repo.DeleteSuratJalanClient(ctx, id); err != nil {
		return err
	}
	u.recordWarehouseAudit(
		ctx,
		"DELETE",
		"warehouse-delivery",
		"surat_jalan_clients",
		fmt.Sprintf("%d", id),
		fmt.Sprintf("Surat Jalan Client #%d", id),
		buildSuratJalanClientAuditSnapshot(beforeItem),
		nil,
	)
	return nil
}

func (u *WarehouseDeliveryUseCase) CreateSuratJalanInternalWithData(ctx context.Context, req model.CreateSuratJalanInternalRequest) (*model.SuratJalanInternalCreateResponse, error) {
	if len(req.IDPackingLists) == 0 {
		return nil, ErrWarehouseValidation
	}

	// Auto-detect id_wo from the first packing list if not supplied
	idWO := req.IDWO
	if idWO <= 0 {
		plRow, err := u.repo.GetPackingListDetail(ctx, entity.GetPackingListDetailParams{
			IDPackingList: req.IDPackingLists[0],
			IDMitra:       pgtype.Int4{Valid: false},
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrWarehouseNotFound
			}
			return nil, fmt.Errorf("%w: failed to resolve WO from packing list", ErrWarehouseServiceUnavailable)
		}
		idWO = plRow.IDWo
	}

	noDokumen := req.NoDokumen
	if noDokumen == "" {
		noDokumen = fmt.Sprintf("SJ-INT/%d/WO-%d", time.Now().Year(), idWO)
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

	deskripsi := req.Deskripsi
	if deskripsi == "" {
		deskripsi = fmt.Sprintf("Surat Jalan Internal WO #%d", idWO)
	}

	sj, err := qtx.CreateSuratJalanInternal(ctx, entity.CreateSuratJalanInternalParams{
		IDWo:      pgtype.Int4{Int32: idWO, Valid: true},
		NoDokumen: noDokumen,
		Deskripsi: deskripsi,
	})
	if err != nil {
		return nil, mapWarehouseDBError(err)
	}

	for idx, item := range req.Items {
		noUrut := int32(item.No)
		if noUrut <= 0 {
			noUrut = int32(idx + 1)
		}
		if _, err := qtx.CreateSuratJalanInternalItem(ctx, entity.CreateSuratJalanInternalItemParams{
			IDSuratJalanInternal: sj.IDSuratJalanInternal,
			NoUrut:               noUrut,
			Deskripsi:            item.Deskripsi,
			Qty:                  item.Qty,
			Note:                 item.Note,
		}); err != nil {
			return nil, mapWarehouseDBError(err)
		}
	}

	for _, plID := range req.IDPackingLists {
		if err := qtx.AssignPackingListToSuratJalan(ctx, entity.AssignPackingListToSuratJalanParams{
			IDSuratJalanInternal: sj.IDSuratJalanInternal,
			IDPackingList:        plID,
		}); err != nil {
			return nil, mapWarehouseDBError(err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("%w: failed to commit transaction", ErrWarehouseServiceUnavailable)
	}

	result := &model.SuratJalanInternalCreateResponse{
		ID:        sj.IDSuratJalanInternal,
		IDWO:      idWO,
		NoDokumen: sj.NoDokumen,
		Deskripsi: sj.Deskripsi,
		CreatedAt: sj.CreatedAt.Time.Format(time.RFC3339),
	}

	u.recordWarehouseAudit(
		ctx,
		"CREATE",
		"warehouse-delivery",
		"surat_jalan_internals",
		fmt.Sprintf("%d", result.ID),
		result.NoDokumen,
		nil,
		buildSuratJalanInternalCreateAuditSnapshot(result),
	)

	return result, nil
}

func (u *WarehouseDeliveryUseCase) GetSuratJalanInternalWithData(ctx context.Context, id int32) (*model.SuratJalanInternalDetailResponse, error) {
	row, err := u.repo.GetSuratJalanInternalDetail(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWarehouseNotFound
		}
		return nil, fmt.Errorf("%w: failed to get surat jalan internal", ErrWarehouseServiceUnavailable)
	}

	plRows, err := u.repo.ListPackingListsBySuratJalanID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get packing lists", ErrWarehouseServiceUnavailable)
	}

	plItems := make([]model.SuratJalanInternalPackingListRow, 0, len(plRows))
	for _, pl := range plRows {
		plItems = append(plItems, model.SuratJalanInternalPackingListRow{
			IDPackingList:      pl.IDPackingList,
			TotalGarmentPerBox: pl.TotalGarmentPerBox,
			TotalReject:        pl.TotalReject,
			IDWO:               pl.IDWo,
			CreatedAt:          pl.CreatedAt.Time.Format(time.RFC3339),
		})
	}

	woShellRows := make([]model.SuratJalanInternalShellRow, 0)
	var idWoPtr *int32
	if row.IDWo.Valid {
		idWoVal := row.IDWo.Int32
		idWoPtr = &idWoVal
	}

	savedItems, err := u.repo.ListSuratJalanInternalItemsBySJID(ctx, id)
	if err == nil && len(savedItems) > 0 {
		for _, item := range savedItems {
			woShellRows = append(woShellRows, model.SuratJalanInternalShellRow{
				No:        int(item.NoUrut),
				Deskripsi: item.Deskripsi,
				Qty:       item.Qty,
				Note:      item.Note,
			})
		}
	} else if row.IDWo.Valid {
		idWoVal := row.IDWo.Int32
		shells, err := u.repo.ListWorkOrderShellsByWorkOrderID(ctx, idWoVal)
		if err == nil {
			shellSizes, _ := u.repo.ListWorkOrderShellSizesByWorkOrderID(ctx, idWoVal)
			idx := 1
			for _, shell := range shells {
				for _, sz := range shellSizes {
					if sz.IDWoShell == shell.IDWoShell {
						desc := fmt.Sprintf("%s - %s | Warna: %s | Size: %s", row.Buyer, row.Model, shell.Color, sz.Size)
						if shell.Deskripsi != "" {
							desc += " (" + shell.Deskripsi + ")"
						}
						woShellRows = append(woShellRows, model.SuratJalanInternalShellRow{
							No:        idx,
							Deskripsi: desc,
							Color:     shell.Color,
							Qty:       sz.Qty,
							Note:      fmt.Sprintf("Ratio %d", sz.Ratio),
						})
						idx++
					}
				}
			}
		}
	}

	return &model.SuratJalanInternalDetailResponse{
		ID:           row.IDSuratJalanInternal,
		IDWO:         idWoPtr,
		NoDokumen:    row.NoDokumen,
		Deskripsi:    row.Deskripsi,
		Buyer:        row.Buyer,
		Model:        row.Model,
		WOQty:        row.WoQty,
		CreatedAt:    row.CreatedAt.Time.Format(time.RFC3339),
		PackingLists: plItems,
		WOShells:     woShellRows,
	}, nil
}

func (u *WarehouseDeliveryUseCase) ListSuratJalanInternalsEnriched(ctx context.Context, filter model.TransactionListFilter) (*model.SuratJalanInternalListResponse, error) {
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
		var idWoPtr *int32
		if row.IDWo.Valid {
			v := row.IDWo.Int32
			idWoPtr = &v
		}
		items = append(items, model.SuratJalanInternalListItem{
			ID:               row.IDSuratJalanInternal,
			IDWO:             idWoPtr,
			NoDokumen:        row.NoDokumen,
			Deskripsi:        row.Deskripsi,
			Buyer:            row.Buyer,
			Model:            row.Model,
			PackingListCount: row.PackingListCount,
			CreatedAt:        row.CreatedAt.Time.Format(time.RFC3339),
		})
	}

	return &model.SuratJalanInternalListResponse{
		Items:      items,
		Pagination: buildPagination(total, page, limit),
	}, nil
}

func (u *WarehouseDeliveryUseCase) AssignPackingListToSJ(ctx context.Context, idSJ int32, idPL int32) error {
	_, err := u.repo.GetSuratJalanInternalDetail(ctx, idSJ)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrWarehouseNotFound
		}
		return fmt.Errorf("%w: failed to get surat jalan internal", ErrWarehouseServiceUnavailable)
	}
	beforeItem, err := u.GetSuratJalanInternalWithData(ctx, idSJ)
	if err != nil {
		return err
	}
	if err := u.repo.AssignPackingListToSuratJalan(ctx, entity.AssignPackingListToSuratJalanParams{
		IDSuratJalanInternal: idSJ,
		IDPackingList:        idPL,
	}); err != nil {
		return err
	}
	afterItem, err := u.GetSuratJalanInternalWithData(ctx, idSJ)
	if err != nil {
		return err
	}
	u.recordWarehouseAudit(
		ctx,
		"UPDATE",
		"warehouse-delivery",
		"surat_jalan_internals",
		fmt.Sprintf("%d", idSJ),
		beforeItem.NoDokumen,
		buildSuratJalanInternalDetailAuditSnapshot(beforeItem),
		buildSuratJalanInternalDetailAuditSnapshot(afterItem),
	)
	return nil
}

func (u *WarehouseDeliveryUseCase) UnassignPackingListFromSJ(ctx context.Context, idPL int32) error {
	packingList, err := u.repo.GetPackingListDetail(ctx, entity.GetPackingListDetailParams{
		IDPackingList: idPL,
		IDMitra:       pgtype.Int4{Valid: false},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrWarehouseNotFound
		}
		return fmt.Errorf("%w: failed to get packing list", ErrWarehouseServiceUnavailable)
	}
	if !packingList.IDSuratJalanInternal.Valid {
		return ErrWarehouseNotFound
	}

	idSJ := packingList.IDSuratJalanInternal.Int32
	beforeItem, err := u.GetSuratJalanInternalWithData(ctx, idSJ)
	if err != nil {
		return err
	}
	if err := u.repo.UnassignPackingListFromSuratJalan(ctx, idPL); err != nil {
		return err
	}
	afterItem, err := u.GetSuratJalanInternalWithData(ctx, idSJ)
	if err != nil {
		return err
	}
	u.recordWarehouseAudit(
		ctx,
		"UPDATE",
		"warehouse-delivery",
		"surat_jalan_internals",
		fmt.Sprintf("%d", idSJ),
		beforeItem.NoDokumen,
		buildSuratJalanInternalDetailAuditSnapshot(beforeItem),
		buildSuratJalanInternalDetailAuditSnapshot(afterItem),
	)
	return nil
}

func (u *WarehouseDeliveryUseCase) recordWarehouseAudit(
	ctx context.Context,
	action string,
	module string,
	entityType string,
	entityID string,
	entityLabel string,
	beforeSnapshot map[string]any,
	afterSnapshot map[string]any,
) {
	if u.auditLog == nil {
		return
	}

	auditCtx, ok := GetAuditLogContext(ctx)
	if !ok {
		return
	}

	if err := u.auditLog.Record(ctx, model.AuditLogRecordRequest{
		ActorUserID:   auditCtx.ActorUserID,
		ActorRole:     auditCtx.ActorRole,
		Action:        action,
		Module:        module,
		EntityType:    entityType,
		EntityID:      entityID,
		EntityLabel:   entityLabel,
		Method:        auditCtx.Method,
		Route:         auditCtx.Route,
		BeforeData:    beforeSnapshot,
		AfterData:     afterSnapshot,
		ChangedFields: buildChangedFieldsFromSnapshots(beforeSnapshot, afterSnapshot),
	}); err != nil {
		slog.Error(
			"failed to record warehouse audit log",
			slog.String("action", action),
			slog.String("entity_type", entityType),
			slog.String("error", err.Error()),
		)
	}
}

func buildReceiveInventoryAuditSnapshot(item *model.ReceiveInventoryResponse) map[string]any {
	if item == nil {
		return nil
	}
	return map[string]any{
		"id_received":                     item.IDReceived,
		"tanggal":                         item.Tanggal,
		"qty":                             item.Qty,
		"keterangan":                      item.Keterangan,
		"id_material_list_item":           item.IDMaterialListItem,
		"id_rekonsiliasi_material":        item.IDRekonsiliasiMaterial,
		"id_rekonsiliasi_material_terima": item.IDRekonsiliasiMaterialTerima,
		"actual_kirim":                    item.ActualKirim,
		"balance":                         item.Balance,
	}
}

func buildIssueInventoryBeforeSnapshot(balance int32) map[string]any {
	return map[string]any{
		"balance": balance,
	}
}

func buildIssueInventoryAuditSnapshot(item *model.IssueInventoryResponse) map[string]any {
	if item == nil {
		return nil
	}
	return map[string]any{
		"id_rekonsiliasi_material": item.IDRekonsiliasiMaterial,
		"qty_issued":               item.QtyIssued,
		"previous_balance":         item.PreviousBalance,
		"balance":                  item.Balance,
	}
}

func buildPackingListAuditSnapshot(item *model.PackingListResponse) map[string]any {
	if item == nil {
		return nil
	}

	items := make([]map[string]any, 0, len(item.Items))
	for _, row := range item.Items {
		sizes := make([]map[string]any, 0, len(row.Sizes))
		for _, size := range row.Sizes {
			sizes = append(sizes, map[string]any{
				"id_wo_shell_size": size.IDWOShellSize,
				"size":             size.Size,
				"qty":              size.Qty,
			})
		}

		items = append(items, map[string]any{
			"id_packing_list_item": row.ID,
			"color":                row.Color,
			"qty_box":              row.QtyBox,
			"qty_per_box":          row.QtyPerBox,
			"box_no_start":         row.BoxNoStart,
			"box_no_end":           row.BoxNoEnd,
			"note":                 row.Note,
			"sizes":                sizes,
		})
	}

	return map[string]any{
		"id_packing_list":         item.ID,
		"total_garment_per_box":   item.TotalGarmentPerBox,
		"total_reject":            item.TotalReject,
		"id_wo":                   item.IDWO,
		"id_surat_jalan_internal": item.IDSuratJalanInternal,
		"items":                   items,
	}
}

func buildSuratJalanAuditSnapshot(item *model.SuratJalanResponse) map[string]any {
	if item == nil {
		return nil
	}
	return map[string]any{
		"type":                  item.Type,
		"id_surat_jalan":        item.IDSuratJalan,
		"tanggal":               item.Tanggal,
		"qty":                   item.Qty,
		"keterangan":            item.Keterangan,
		"id_material_list_item": item.IDMaterialListItem,
		"created_at":            item.CreatedAt,
	}
}

func buildSimpleReceivedAuditSnapshot(item *model.SimpleReceivedResponse) map[string]any {
	if item == nil {
		return nil
	}
	return map[string]any{
		"id_received":           item.IDReceived,
		"tanggal":               item.Tanggal,
		"qty":                   item.Qty,
		"keterangan":            item.Keterangan,
		"id_material_list_item": item.IDMaterialListItem,
	}
}

func buildSimpleReceivedDetailAuditSnapshot(item *model.SimpleReceivedDetailResponse) map[string]any {
	if item == nil {
		return nil
	}
	return map[string]any{
		"id_received":           item.IDReceived,
		"tanggal":               item.Tanggal,
		"qty":                   item.Qty,
		"keterangan":            item.Keterangan,
		"id_material_list_item": item.IDMaterialListItem,
		"material_item":         item.MaterialItem,
		"material_description":  item.MaterialDescription,
		"id_wo":                 item.IDWO,
	}
}

func buildSuratJalanClientAuditSnapshot(item *model.SuratJalanClientDetailResponse) map[string]any {
	if item == nil {
		return nil
	}
	return map[string]any{
		"id_surat_jalan_client": item.ID,
		"tanggal":               item.Tanggal,
		"qty":                   item.Qty,
		"keterangan":            item.Keterangan,
		"id_material_list_item": item.IDMaterialListItem,
		"material_description":  item.MaterialDescription,
		"id_wo":                 item.IDWO,
	}
}

func buildSuratJalanInternalCreateAuditSnapshot(item *model.SuratJalanInternalCreateResponse) map[string]any {
	if item == nil {
		return nil
	}
	return map[string]any{
		"id_surat_jalan_internal": item.ID,
		"id_wo":                   item.IDWO,
		"no_dokumen":              item.NoDokumen,
		"deskripsi":               item.Deskripsi,
		"created_at":              item.CreatedAt,
	}
}

func buildSuratJalanInternalDetailAuditSnapshot(item *model.SuratJalanInternalDetailResponse) map[string]any {
	if item == nil {
		return nil
	}

	packingLists := make([]map[string]any, 0, len(item.PackingLists))
	for _, row := range item.PackingLists {
		packingLists = append(packingLists, map[string]any{
			"id_packing_list":       row.IDPackingList,
			"total_garment_per_box": row.TotalGarmentPerBox,
			"total_reject":          row.TotalReject,
			"id_wo":                 row.IDWO,
			"created_at":            row.CreatedAt,
		})
	}

	items := make([]map[string]any, 0, len(item.WOShells))
	for _, row := range item.WOShells {
		items = append(items, map[string]any{
			"no":        row.No,
			"deskripsi": row.Deskripsi,
			"color":     row.Color,
			"qty":       row.Qty,
			"note":      row.Note,
		})
	}

	return map[string]any{
		"id_surat_jalan_internal": item.ID,
		"id_wo":                   item.IDWO,
		"no_dokumen":              item.NoDokumen,
		"deskripsi":               item.Deskripsi,
		"buyer":                   item.Buyer,
		"model":                   item.Model,
		"wo_qty":                  item.WOQty,
		"packing_lists":           packingLists,
		"items":                   items,
	}
}
