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
	ErrWorkOrderValidation         = errors.New("invalid work order payload")
	ErrWorkOrderReferenceNotFound  = errors.New("related data not found")
	ErrWorkOrderServiceUnavailable = errors.New("work order service unavailable")
	ErrWorkOrderNotFound           = errors.New("work order not found")
	ErrReportDivisionUnsupported   = errors.New("unsupported report division")
	ErrWorkOrderAlreadyClosed      = errors.New("work order is already closed")
	ErrWorkOrderNotClosedByClient  = errors.New("work order must be marked as close by client first")
	ErrWorkOrderNotOpen            = errors.New("cannot submit return: work order is not open")
	ErrReturnAlreadySubmitted      = errors.New("return already submitted for this work order")

	ErrPOClientItemAlreadyAssigned = errors.New("work order sudah dibuat untuk PO client item ini")

	workOrderSortColumns         = buildSortWhitelist("created_at", "id_wo", "buyer", "model", "qty", "status", "po_number", "po_client_item_style")
	productionSummarySortColumns = buildSortWhitelist("last_updated", "id_wo_shell_size", "model_name", "size", "target_qty", "cutting_qty", "sewing_qty", "qc_pass_qty", "packing_qty", "shipped_qty")
)

type WorkOrderProductionUseCase struct {
	repo   entity.Querier
	dbPool *pgxpool.Pool
}

func NewWorkOrderProductionUseCase(repo entity.Querier, dbPool *pgxpool.Pool) (*WorkOrderProductionUseCase, error) {
	if repo == nil {
		return nil, errors.New("work order repository is required")
	}
	if dbPool == nil {
		return nil, errors.New("database pool is required")
	}

	return &WorkOrderProductionUseCase{
		repo:   repo,
		dbPool: dbPool,
	}, nil
}

func (u *WorkOrderProductionUseCase) CreateWorkOrder(ctx context.Context, userID int32, req model.CreateWorkOrderRequest) (*model.WorkOrderResponse, error) {
	if len(req.Shells) == 0 || len(req.Trims) == 0 {
		return nil, ErrWorkOrderValidation
	}
	if err := validateDate(req.Delivery); err != nil {
		return nil, ErrWorkOrderValidation
	}
	for _, shell := range req.Shells {
		if len(shell.Sizes) == 0 {
			return nil, ErrWorkOrderValidation
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

	header, err := qtx.CreateWorkOrder(ctx, entity.CreateWorkOrderParams{
		Buyer:          req.Buyer,
		Model:          req.Model,
		Qty:            req.Qty,
		FobCmt:         req.FOBCMT,
		Delivery:       mustDate(req.Delivery),
		IDPoClientItem: req.IDPOClientItem,
	})
	if err != nil {
		return nil, mapWorkOrderDBError(err)
	}

	// Struct bantuan untuk melacak ID yang dihasilkan database selama runtime transaksi
	type ShellTrack struct {
		ID        int32
		Deskripsi string
		Color     string
	}
	type TrimTrack struct {
		ID    int32
		Item  string
		Color string
	}

	recordedShells := make([]ShellTrack, 0, len(req.Shells))
	shells := make([]model.WorkOrderShellResponse, 0, len(req.Shells))
	for _, shellReq := range req.Shells {
		shell, shellErr := qtx.CreateWorkOrderShell(ctx, entity.CreateWorkOrderShellParams{
			Deskripsi:    shellReq.Deskripsi,
			Cons:         mustNumeric(shellReq.Cons),
			Color:        shellReq.Color,
			Allow:        shellReq.Allow,
			Berat1Yd:     mustNumeric(shellReq.Berat1Yd),
			IDWo:         header.IDWo,
			ProvidedBy:   shellReq.ProvidedBy,
			MaterialType: shellReq.MaterialType,
		})
		if shellErr != nil {
			return nil, mapWorkOrderDBError(shellErr)
		}

		// Simpan informasi kain untuk pencocokan material garmen nanti
		recordedShells = append(recordedShells, ShellTrack{
			ID:        shell.IDWoShell,
			Deskripsi: shell.Deskripsi,
			Color:     shell.Color,
		})

		sizes := make([]model.WorkOrderShellSizeResponse, 0, len(shellReq.Sizes))
		for _, sizeReq := range shellReq.Sizes {
			size, sizeErr := qtx.CreateWorkOrderShellSize(ctx, entity.CreateWorkOrderShellSizeParams{
				Size:      sizeReq.Size,
				Qty:       sizeReq.Qty,
				Ratio:     sizeReq.Ratio,
				IDWoShell: shell.IDWoShell,
			})
			if sizeErr != nil {
				return nil, mapWorkOrderDBError(sizeErr)
			}

			sizes = append(sizes, model.WorkOrderShellSizeResponse{
				ID:        size.IDWoShellSize,
				Size:      size.Size,
				Qty:       size.Qty,
				Ratio:     size.Ratio,
				CreatedAt: size.CreatedAt.Time.Format(time.RFC3339),
			})
		}

		shells = append(shells, model.WorkOrderShellResponse{
			ID:           shell.IDWoShell,
			Deskripsi:    shell.Deskripsi,
			Cons:         numericToFloat64(shell.Cons),
			Color:        shell.Color,
			Allow:        shell.Allow,
			Berat1Yd:     numericToFloat64(shell.Berat1Yd),
			CreatedAt:    shell.CreatedAt.Time.Format(time.RFC3339),
			ProvidedBy:   shell.ProvidedBy,
			MaterialType: shell.MaterialType,
			Sizes:        sizes,
		})
	}

	recordedTrims := make([]TrimTrack, 0, len(req.Trims))
	trims := make([]model.WorkOrderTrimResponse, 0, len(req.Trims))
	for _, trimReq := range req.Trims {
		trim, trimErr := qtx.CreateWorkOrderTrim(ctx, entity.CreateWorkOrderTrimParams{
			Item:        trimReq.Item,
			Description: trimReq.Description,
			Color:       trimReq.Color,
			Code:        trimReq.Code,
			Cons:        mustNumeric(trimReq.Cons),
			Qty:         trimReq.Qty,
			Uom:         trimReq.UOM,
			Position:    trimReq.Position,
			CreatedBy:   trimReq.CreatedBy,
			Allow:       trimReq.Allow,
			IDWo:        header.IDWo,
			ProvidedBy:  trimReq.ProvidedBy,
		})
		if trimErr != nil {
			return nil, mapWorkOrderDBError(trimErr)
		}

		// Simpan informasi aksesoris (trim) untuk pencocokan material garmen nanti
		recordedTrims = append(recordedTrims, TrimTrack{
			ID:    trim.IDWoTrim,
			Item:  trim.Item,
			Color: trim.Color,
		})

		trims = append(trims, model.WorkOrderTrimResponse{
			ID:          trim.IDWoTrim,
			Item:        trim.Item,
			Description: trim.Description,
			Color:       trim.Color,
			Code:        trim.Code,
			Cons:        numericToFloat64(trim.Cons),
			Qty:         trim.Qty,
			UOM:         trim.Uom,
			Position:    trim.Position,
			CreatedBy:   trim.CreatedBy,
			Allow:       trim.Allow,
			CreatedAt:   trim.CreatedAt.Time.Format(time.RFC3339),
			ProvidedBy:  trim.ProvidedBy,
		})
	}

	materials := make([]model.MaterialListResponse, 0, len(req.MaterialLists))
	for _, materialReq := range req.MaterialLists {
		var idWoShell pgtype.Int4
		var idWoTrim pgtype.Int4

		// [DEBUG LOG] Pantau data apa yang sedang dibaca backend
		fmt.Printf("[WO-DEBUG] Memproses Material: Desc='%s' | Color='%s'\n", materialReq.Description, materialReq.Color)

		// 1. Logika Pengait Otomatis ke WORK_ORDER_SHELL (Kain)
		for _, s := range recordedShells {
			// Skenario A: Warna match DAN teks deskripsi mirip
			textMatch := strings.Contains(strings.ToLower(materialReq.Description), strings.ToLower(s.Deskripsi)) ||
				strings.Contains(strings.ToLower(s.Deskripsi), strings.ToLower(materialReq.Description))

			if strings.EqualFold(materialReq.Color, s.Color) && (textMatch || materialReq.Description == "") {
				idWoShell = pgtype.Int4{Int32: s.ID, Valid: true}
				fmt.Printf("   -> 🎉 MATCH FOUND ke Shell ID: %d (Deskripsi: %s)\n", s.ID, s.Deskripsi)
				break
			}
		}

		// 2. Logika Pengait Otomatis ke WORK_ORDER_TRIM (Aksesoris)
		if !idWoShell.Valid {
			for _, t := range recordedTrims {
				textMatch := strings.Contains(strings.ToLower(materialReq.Description), strings.ToLower(t.Item)) ||
					strings.Contains(strings.ToLower(t.Item), strings.ToLower(materialReq.Description))

				if strings.EqualFold(materialReq.Color, t.Color) && (textMatch || materialReq.Description == "") {
					idWoTrim = pgtype.Int4{Int32: t.ID, Valid: true}
					fmt.Printf("   -> 🎉 MATCH FOUND ke Trim ID: %d (Item: %s)\n", t.ID, t.Item)
					break
				}
			}
		}

		// Skenario Fallback: Jika teks tidak ada yang mirip, tapi warna sama persis dengan kain tunggal
		if !idWoShell.Valid && !idWoTrim.Valid && len(recordedShells) == 1 {
			if strings.EqualFold(materialReq.Color, recordedShells[0].Color) {
				idWoShell = pgtype.Int4{Int32: recordedShells[0].ID, Valid: true}
				fmt.Printf("   -> ⚠️ FALLBACK MATCH (Hanya 1 Shell): Terhubung ke Shell ID %d karena kesamaan warna\n", recordedShells[0].ID)
			}
		}

		// Jika tetap tidak ketemu match sama sekali
		if !idWoShell.Valid && !idWoTrim.Valid {
			fmt.Println("   -> ❌ NO MATCH FOUND: Kolom akan bernilai NULL di database.")
		}

		// 3. Eksekusi simpan ke database dengan parameter ter-update
		material, materialErr := qtx.CreateMaterialList(ctx, entity.CreateMaterialListParams{
			Description: materialReq.Description,
			Size:        materialReq.Size,
			Color:       materialReq.Color,
			Uom:         materialReq.UOM,
			IDWo:        header.IDWo,
			IDWoShell:   idWoShell, // Hasil pencarian dinamis
			IDWoTrim:    idWoTrim,  // Hasil pencarian dinamis
		})
		if materialErr != nil {
			return nil, mapWorkOrderDBError(materialErr)
		}

		materials = append(materials, model.MaterialListResponse{
			ID:          material.IDMaterialList,
			Description: material.Description,
			Size:        material.Size,
			Color:       material.Color,
			UOM:         material.Uom,
			CreatedAt:   material.CreatedAt.Time.Format(time.RFC3339),
		})
	}

	err = initializeApprovalWorkflow(ctx, qtx, "WORK_ORDER", header.IDWo, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize approval workflow: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("%w: failed to commit transaction", ErrWorkOrderServiceUnavailable)
	}

	return &model.WorkOrderResponse{
		ID:             header.IDWo,
		Buyer:          header.Buyer,
		Model:          header.Model,
		Qty:            header.Qty,
		FOBCMT:         header.FobCmt,
		Delivery:       header.Delivery.Time.Format("2006-01-02"),
		IDPOClientItem: header.IDPoClientItem,
		Status:         "open",
		CreatedAt:      header.CreatedAt.Time.Format(time.RFC3339),
		Shells:         shells,
		Trims:          trims,
		MaterialLists:  materials,
	}, nil
}

func (u *WorkOrderProductionUseCase) CloseWorkOrder(ctx context.Context, id int32, closerUserID int32) (*model.WorkOrderStatusResponse, error) {
	// First run auto-close to catch any auto-closable work orders
	if err := u.repo.AutoCloseWorkOrders(ctx); err != nil {
		return nil, fmt.Errorf("%w: failed to run auto-close", ErrWorkOrderServiceUnavailable)
	}

	current, err := u.repo.GetWorkOrderDetail(ctx, entity.GetWorkOrderDetailParams{
		IDWo:    id,
		IDMitra: nullableInt32Param(nil),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWorkOrderNotFound
		}
		return nil, fmt.Errorf("%w: failed to get work order", ErrWorkOrderServiceUnavailable)
	}
	if current.Status == "closed" {
		return nil, ErrWorkOrderAlreadyClosed
	}
	if current.Status != "client_closed" {
		return nil, ErrWorkOrderNotClosedByClient
	}

	updated, err := u.repo.CloseWorkOrder(ctx, entity.CloseWorkOrderParams{
		IDWo:           id,
		ClosedByUserID: pgtype.Int4{Int32: closerUserID, Valid: true},
	})
	if err != nil {
		return nil, mapWorkOrderDBError(err)
	}

	return &model.WorkOrderStatusResponse{
		ID:             updated.IDWo,
		Status:         updated.Status,
		ClosedByUserID: nullableInt32Ptr(updated.ClosedByUserID),
		ClosedAt:       nullableTimestampString(updated.ClosedAt),
	}, nil
}

func (u *WorkOrderProductionUseCase) CreateFactoryReport(ctx context.Context, division string, req model.CreateFactoryReportRequest) (*model.FactoryReportResponse, error) {
	if err := validateDate(req.Tanggal); err != nil {
		return nil, ErrWorkOrderValidation
	}

	normalized := normalizeDivision(division)
	switch normalized {
	case "cutting":
		item, err := u.repo.CreateReportCutting(ctx, entity.CreateReportCuttingParams{
			Tanggal:       mustDate(req.Tanggal),
			Qty:           req.Qty,
			IDWoShellSize: req.IDWOShellSize,
		})
		if err != nil {
			return nil, mapWorkOrderDBError(err)
		}
		return &model.FactoryReportResponse{
			Division:      "cutting",
			ReportID:      item.IDReportCutting,
			Tanggal:       item.Tanggal.Time.Format("2006-01-02"),
			Qty:           item.Qty,
			IDWOShellSize: item.IDWoShellSize,
			CreatedAt:     item.CreatedAt.Time.Format(time.RFC3339),
		}, nil
	case "sewing":
		item, err := u.repo.CreateReportSewing(ctx, entity.CreateReportSewingParams{
			Tanggal:       mustDate(req.Tanggal),
			Qty:           req.Qty,
			IDWoShellSize: req.IDWOShellSize,
		})
		if err != nil {
			return nil, mapWorkOrderDBError(err)
		}
		return &model.FactoryReportResponse{
			Division:      "sewing",
			ReportID:      item.IDReportSewing,
			Tanggal:       item.Tanggal.Time.Format("2006-01-02"),
			Qty:           item.Qty,
			IDWOShellSize: item.IDWoShellSize,
			CreatedAt:     item.CreatedAt.Time.Format(time.RFC3339),
		}, nil
	case "qc-finish":
		item, err := u.repo.CreateReportQCFinish(ctx, entity.CreateReportQCFinishParams{
			Tanggal:       mustDate(req.Tanggal),
			Qty:           req.Qty,
			IDWoShellSize: req.IDWOShellSize,
		})
		if err != nil {
			return nil, mapWorkOrderDBError(err)
		}
		return &model.FactoryReportResponse{
			Division:      "qc-finish",
			ReportID:      item.IDReportQcFinishing,
			Tanggal:       item.Tanggal.Time.Format("2006-01-02"),
			Qty:           item.Qty,
			IDWOShellSize: item.IDWoShellSize,
			CreatedAt:     item.CreatedAt.Time.Format(time.RFC3339),
		}, nil
	case "packing":
		item, err := u.repo.CreateReportPacking(ctx, entity.CreateReportPackingParams{
			Tanggal:       mustDate(req.Tanggal),
			Qty:           req.Qty,
			IDWoShellSize: req.IDWOShellSize,
		})
		if err != nil {
			return nil, mapWorkOrderDBError(err)
		}
		return &model.FactoryReportResponse{
			Division:      "packing",
			ReportID:      item.IDReportPacking,
			Tanggal:       item.Tanggal.Time.Format("2006-01-02"),
			Qty:           item.Qty,
			IDWOShellSize: item.IDWoShellSize,
			CreatedAt:     item.CreatedAt.Time.Format(time.RFC3339),
		}, nil
	case "pengiriman":
		item, err := u.repo.CreateReportPengiriman(ctx, entity.CreateReportPengirimanParams{
			ReportDate:    mustDate(req.Tanggal),
			Qty:           req.Qty,
			IDWoShellSize: req.IDWOShellSize,
		})
		if err != nil {
			return nil, mapWorkOrderDBError(err)
		}
		return &model.FactoryReportResponse{
			Division:      "pengiriman",
			ReportID:      item.IDReportPengiriman,
			Tanggal:       item.ReportDate.Time.Format("2006-01-02"),
			Qty:           item.Qty,
			IDWOShellSize: item.IDWoShellSize,
			CreatedAt:     item.CreatedAt.Time.Format(time.RFC3339),
		}, nil
	default:
		return nil, ErrReportDivisionUnsupported
	}
}

func (u *WorkOrderProductionUseCase) ListWorkOrders(ctx context.Context, filter model.TransactionListFilter) (*model.WorkOrderListResponse, error) {
	// Automatically close any work orders with no returns after 2 months
	if err := u.repo.AutoCloseWorkOrders(ctx); err != nil {
		return nil, fmt.Errorf("%w: failed to run auto-close", ErrWorkOrderServiceUnavailable)
	}

	page, limit, offset, search, sortBy, sortDesc := normalizeListFilter(filter.ListQueryFilter, "id_wo", true, workOrderSortColumns)
	rows, err := u.repo.ListWorkOrders(ctx, entity.ListWorkOrdersParams{
		SearchTerm: search,
		IDMitra:    nullableInt32Param(filter.IDMitra),
		SortBy:     sortBy,
		SortDesc:   sortDesc,
		PageLimit:  limit,
		PageOffset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: failed to list work orders", ErrWorkOrderServiceUnavailable)
	}

	items := make([]model.WorkOrderListItem, 0, len(rows))
	total := int64(0)
	for _, row := range rows {
		total = row.TotalCount
		var returFilePtr *string
		if row.ReturFile != "" {
			val := row.ReturFile
			returFilePtr = &val
		}

		items = append(items, model.WorkOrderListItem{
			ID:                row.IDWo,
			Buyer:             row.Buyer,
			Model:             row.Model,
			Qty:               row.Qty,
			FOBCMT:            row.FobCmt,
			Delivery:          row.Delivery.Time.Format("2006-01-02"),
			IDPOClientItem:    row.IDPoClientItem,
			Status:            row.Status,
			ClosedAt:          nullableTimestampString(row.ClosedAt),
			PONumber:          row.PoNumber,
			POClientItemStyle: row.PoClientItemStyle,
			CreatedAt:         row.CreatedAt.Time.Format(time.RFC3339),
			HasRetur:          row.HasRetur,
			IDPOClient:        row.IDPoClient,
			ReturFile:         returFilePtr,
		})
	}

	return &model.WorkOrderListResponse{
		Items:      items,
		Pagination: buildPagination(total, page, limit),
	}, nil
}

func (u *WorkOrderProductionUseCase) ListProductionSummary(ctx context.Context, filter model.ProductionSummaryFilter) (*model.ProductionSummaryListResponse, error) {
	if filter.IDWO < 0 || filter.IDWOShellSize < 0 {
		return nil, ErrWorkOrderValidation
	}

	page, limit, offset, search, sortBy, sortDesc := normalizeListFilter(filter.ListQueryFilter, "last_updated", true, productionSummarySortColumns)

	rows, err := u.repo.ListProductionSummary(ctx, entity.ListProductionSummaryParams{
		IDWo:          filter.IDWO,
		IDWoShellSize: filter.IDWOShellSize,
		IDMitra:       nullableInt32Param(filter.IDMitra),
		SearchTerm:    search,
		SortBy:        sortBy,
		SortDesc:      sortDesc,
		PageOffset:    offset,
		PageLimit:     limit,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: failed to list production summary", ErrWorkOrderServiceUnavailable)
	}

	items := make([]model.ProductionAggregateResponse, 0, len(rows))
	total := int64(0)
	for _, row := range rows {
		total = row.TotalCount
		items = append(items, model.ProductionAggregateResponse{
			IDWOShellSize: row.IDWoShellSize,
			ModelName:     row.ModelName,
			Size:          row.Size,
			TargetQty:     row.TargetQty,
			Production: model.ProductionStats{
				Cutting: row.CuttingQty,
				Sewing:  row.SewingQty,
				QCPass:  row.QcPassQty,
				Packing: row.PackingQty,
				Shipped: row.ShippedQty,
			},
			LastUpdated: nullableTimestampString(row.LastUpdated),
			Status:      deriveProductionStatus(row.TargetQty, row.CuttingQty, row.SewingQty, row.QcPassQty, row.PackingQty, row.ShippedQty),
		})
	}

	return &model.ProductionSummaryListResponse{
		Items:      items,
		Pagination: buildPagination(total, page, limit),
	}, nil
}

func (u *WorkOrderProductionUseCase) GetWorkOrderDetail(ctx context.Context, id int32, idMitra *int32) (*model.WorkOrderDetailResponse, error) {
	// Automatically close any work orders with no returns after 2 months
	if err := u.repo.AutoCloseWorkOrders(ctx); err != nil {
		return nil, fmt.Errorf("%w: failed to run auto-close", ErrWorkOrderServiceUnavailable)
	}

	header, err := u.repo.GetWorkOrderDetail(ctx, entity.GetWorkOrderDetailParams{
		IDWo:    id,
		IDMitra: nullableInt32Param(idMitra),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWorkOrderNotFound
		}
		return nil, fmt.Errorf("%w: failed to get work order", ErrWorkOrderServiceUnavailable)
	}

	shellRows, err := u.repo.ListWorkOrderShellsByWorkOrderID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get work order shells", ErrWorkOrderServiceUnavailable)
	}
	sizeRows, err := u.repo.ListWorkOrderShellSizesByWorkOrderID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get work order shell sizes", ErrWorkOrderServiceUnavailable)
	}
	trimRows, err := u.repo.ListWorkOrderTrimsByWorkOrderID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get work order trims", ErrWorkOrderServiceUnavailable)
	}
	materialRows, err := u.repo.ListMaterialListsByWorkOrderID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get material lists", ErrWorkOrderServiceUnavailable)
	}

	sizeMap := make(map[int32][]model.WorkOrderShellSizeResponse)
	for _, row := range sizeRows {
		sizeMap[row.IDWoShell] = append(sizeMap[row.IDWoShell], model.WorkOrderShellSizeResponse{
			ID:        row.IDWoShellSize,
			Size:      row.Size,
			Qty:       row.Qty,
			Ratio:     row.Ratio,
			CreatedAt: row.CreatedAt.Time.Format(time.RFC3339),
		})
	}

	shells := make([]model.WorkOrderShellResponse, 0, len(shellRows))
	for _, row := range shellRows {
		shells = append(shells, model.WorkOrderShellResponse{
			ID:           row.IDWoShell,
			Deskripsi:    row.Deskripsi,
			Cons:         numericToFloat64(row.Cons),
			Color:        row.Color,
			Allow:        row.Allow,
			Berat1Yd:     numericToFloat64(row.Berat1Yd),
			CreatedAt:    row.CreatedAt.Time.Format(time.RFC3339),
			ProvidedBy:   row.ProvidedBy,
			MaterialType: row.MaterialType,
			Sizes:        sizeMap[row.IDWoShell],
		})
	}

	trims := make([]model.WorkOrderTrimResponse, 0, len(trimRows))
	for _, row := range trimRows {
		trims = append(trims, model.WorkOrderTrimResponse{
			ID:          row.IDWoTrim,
			Item:        row.Item,
			Description: row.Description,
			Color:       row.Color,
			Code:        row.Code,
			Cons:        numericToFloat64(row.Cons),
			Qty:         row.Qty,
			UOM:         row.Uom,
			Position:    row.Position,
			CreatedBy:   row.CreatedBy,
			Allow:       row.Allow,
			CreatedAt:   row.CreatedAt.Time.Format(time.RFC3339),
			ProvidedBy:  row.ProvidedBy,
		})
	}

	materials := make([]model.MaterialListResponse, 0, len(materialRows))
	for _, row := range materialRows {
		materials = append(materials, model.MaterialListResponse{
			ID:          row.IDMaterialList,
			Description: row.Description,
			Size:        row.Size,
			Color:       row.Color,
			UOM:         row.Uom,
			CreatedAt:   row.CreatedAt.Time.Format(time.RFC3339),
		})
	}

	var returResponse *model.ReturClientResponse
	returRow, err := u.repo.GetReturClientByWorkOrderID(ctx, id)
	if err == nil {
		returResponse = &model.ReturClientResponse{
			IDReturClient: returRow.IDReturClient,
			IDWo:          returRow.IDWo,
			File:          returRow.File,
			Deskripsi:     returRow.Deskripsi,
			CreatedAt:     returRow.CreatedAt.Time.Format(time.RFC3339),
		}
	}

	return &model.WorkOrderDetailResponse{
		ID:                header.IDWo,
		Buyer:             header.Buyer,
		Model:             header.Model,
		Qty:               header.Qty,
		FOBCMT:            header.FobCmt,
		Delivery:          header.Delivery.Time.Format("2006-01-02"),
		IDPOClientItem:    header.IDPoClientItem,
		Status:            header.Status,
		ClosedByUserID:    nullableInt32Ptr(header.ClosedByUserID),
		ClosedAt:          nullableTimestampString(header.ClosedAt),
		PONumber:          header.PoNumber,
		POClientItemStyle: header.PoClientItemStyle,
		CreatedAt:         header.CreatedAt.Time.Format(time.RFC3339),
		Shells:            shells,
		Trims:             trims,
		MaterialLists:     materials,
		Retur:             returResponse,
	}, nil
}

func mapWorkOrderDBError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503":
			return ErrWorkOrderReferenceNotFound
		case "23505":
			if strings.Contains(pgErr.ConstraintName, "unique_id_po_client_item") {
				return ErrPOClientItemAlreadyAssigned
			}
			return ErrPOClientItemAlreadyAssigned
		}
	}
	return fmt.Errorf("%w: %v", ErrWorkOrderServiceUnavailable, err)
}

func normalizeDivision(value string) string {
	normalized := strings.TrimSpace(strings.ToLower(value))
	normalized = strings.ReplaceAll(normalized, "_", "-")
	return normalized
}

func deriveProductionStatus(targetQty int32, cutting int32, sewing int32, qcPass int32, packing int32, shipped int32) string {
	switch {
	case targetQty > 0 && shipped >= targetQty:
		return "Completed"
	case shipped > 0:
		return "Shipping Stage"
	case packing > 0:
		return "Packing Stage"
	case qcPass > 0:
		return "QC Stage"
	case sewing > 0:
		return "Sewing Stage"
	case cutting > 0:
		return "Cutting Stage"
	default:
		return "Not Started"
	}
}

func (u *WorkOrderProductionUseCase) GetWorkOrderShellTotalQty(ctx context.Context, idWoShell int32) (*model.WorkOrderShellTotalQtyResponse, error) {
	totalQty, err := u.repo.WorkOrderShellTotalQty(ctx, idWoShell)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get work order shell total qty", ErrWorkOrderServiceUnavailable)
	}

	return &model.WorkOrderShellTotalQtyResponse{
		IDWoShell: idWoShell,
		TotalQty:  totalQty,
	}, nil
}

func (u *WorkOrderProductionUseCase) CreateReturClient(ctx context.Context, idWo int32, file string, deskripsi string, idMitra *int32) (*model.ReturClientResponse, error) {
	// Automatically close other orders
	if err := u.repo.AutoCloseWorkOrders(ctx); err != nil {
		return nil, fmt.Errorf("%w: failed to run auto-close", ErrWorkOrderServiceUnavailable)
	}

	// Fetch WO and verify it exists and is open
	wo, err := u.repo.GetWorkOrderDetail(ctx, entity.GetWorkOrderDetailParams{
		IDWo:    idWo,
		IDMitra: nullableInt32Param(idMitra),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWorkOrderNotFound
		}
		return nil, fmt.Errorf("%w: failed to fetch work order", ErrWorkOrderServiceUnavailable)
	}

	fmt.Printf("[RETUR-DEBUG] ID WO: %d, Status Saat Ini: '%s'\n", idWo, wo.Status)

	if wo.Status != "open" && wo.Status != "pending" {
		return nil, ErrWorkOrderNotOpen
	}

	// Verify that a return has not been submitted yet
	_, err = u.repo.GetReturClientByWorkOrderID(ctx, idWo)
	if err == nil {
		return nil, ErrReturnAlreadySubmitted
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("%w: failed to verify existing return", ErrWorkOrderServiceUnavailable)
	}

	// Create return record
	retur, err := u.repo.CreateReturClient(ctx, entity.CreateReturClientParams{
		IDWo:      idWo,
		File:      file,
		Deskripsi: deskripsi,
	})
	if err != nil {
		return nil, mapWorkOrderDBError(err)
	}

	return &model.ReturClientResponse{
		IDReturClient: retur.IDReturClient,
		IDWo:          retur.IDWo,
		File:          retur.File,
		Deskripsi:     retur.Deskripsi,
		CreatedAt:     retur.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (u *WorkOrderProductionUseCase) ClientCloseWorkOrder(ctx context.Context, idWo int32, idMitra *int32) (*model.WorkOrderStatusResponse, error) {
	// Automatically close other orders
	if err := u.repo.AutoCloseWorkOrders(ctx); err != nil {
		return nil, fmt.Errorf("%w: failed to run auto-close", ErrWorkOrderServiceUnavailable)
	}

	// Fetch WO and verify it exists and is open
	wo, err := u.repo.GetWorkOrderDetail(ctx, entity.GetWorkOrderDetailParams{
		IDWo:    idWo,
		IDMitra: nullableInt32Param(idMitra),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWorkOrderNotFound
		}
		return nil, fmt.Errorf("%w: failed to fetch work order", ErrWorkOrderServiceUnavailable)
	}

	if wo.Status == "closed" {
		return nil, ErrWorkOrderAlreadyClosed
	}

	// If it is already client_closed, return immediately with success
	if wo.Status == "client_closed" {
		return &model.WorkOrderStatusResponse{
			ID:     wo.IDWo,
			Status: "client_closed",
		}, nil
	}

	// Update status to client_closed
	updated, err := u.repo.ClientCloseWorkOrder(ctx, idWo)
	if err != nil {
		return nil, mapWorkOrderDBError(err)
	}

	return &model.WorkOrderStatusResponse{
		ID:     updated.IDWo,
		Status: updated.Status,
	}, nil
}

func (u *WorkOrderProductionUseCase) GetDailyReportsByWorkOrder(ctx context.Context, idWo int32) (*model.DailyReportListResponse, error) {
	rows, err := u.repo.GetDailyReportsByWorkOrder(ctx, idWo)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to fetch daily reports", ErrWorkOrderServiceUnavailable)
	}

	items := make([]model.DailyReportListItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, model.DailyReportListItem{
			Division:      row.Division,
			Tanggal:       row.Tanggal.Time.Format("2006-01-02"),
			Qty:           row.Qty,
			IDWOShellSize: row.IDWoShellSize,
		})
	}

	return &model.DailyReportListResponse{
		Items: items,
	}, nil
}

func (u *WorkOrderProductionUseCase) ListReturClients(ctx context.Context, filter model.TransactionListFilter) (*model.ReturClientListResponse, error) {
	page, limit, offset, search, _, _ := normalizeListFilter(filter.ListQueryFilter, "", false, nil)

	rows, err := u.repo.ListReturClients(ctx, entity.ListReturClientsParams{
		IDMitra:    nullableInt32Param(filter.IDMitra),
		Search:     search,
		PageLimit:  limit,
		PageOffset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: failed to list retur clients", ErrWorkOrderServiceUnavailable)
	}

	items := make([]model.ReturClientListItem, 0, len(rows))
	total := int64(0)
	for _, row := range rows {
		total = row.TotalCount
		items = append(items, model.ReturClientListItem{
			IDReturClient: row.IDReturClient,
			IDWo:          row.IDWo,
			File:          row.File,
			Deskripsi:     row.Deskripsi,
			CreatedAt:     row.CreatedAt.Time.Format(time.RFC3339),
			Buyer:         row.Buyer,
			Model:         row.Model,
			WoQty:         row.WoQty,
			PoNumber:      row.PoNumber,
			IDMitra:       row.IDMitra,
			MitraName:     row.MitraName,
			IDPOClient:    row.IDPoClient,
		})
	}

	return &model.ReturClientListResponse{
		Items:      items,
		Pagination: buildPagination(total, page, limit),
	}, nil
}
