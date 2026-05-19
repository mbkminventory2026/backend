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

func (u *WorkOrderProductionUseCase) CreateWorkOrder(ctx context.Context, req model.CreateWorkOrderRequest) (*model.WorkOrderResponse, error) {
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

	shells := make([]model.WorkOrderShellResponse, 0, len(req.Shells))
	for _, shellReq := range req.Shells {
		shell, shellErr := qtx.CreateWorkOrderShell(ctx, entity.CreateWorkOrderShellParams{
			Fabric:   shellReq.Fabric,
			Cons:     mustNumeric(shellReq.Cons),
			Color:    shellReq.Color,
			Allow:    shellReq.Allow,
			Berat1Yd: mustNumeric(shellReq.Berat1Yd),
			IDWo:     header.IDWo,
		})
		if shellErr != nil {
			return nil, mapWorkOrderDBError(shellErr)
		}

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
			ID:        shell.IDWoShell,
			Fabric:    shell.Fabric,
			Cons:      numericToFloat64(shell.Cons),
			Color:     shell.Color,
			Allow:     shell.Allow,
			Berat1Yd:  numericToFloat64(shell.Berat1Yd),
			CreatedAt: shell.CreatedAt.Time.Format(time.RFC3339),
			Sizes:     sizes,
		})
	}

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
		})
		if trimErr != nil {
			return nil, mapWorkOrderDBError(trimErr)
		}

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
		})
	}

	materials := make([]model.MaterialListResponse, 0, len(req.MaterialLists))
	for _, materialReq := range req.MaterialLists {
		material, materialErr := qtx.CreateMaterialList(ctx, entity.CreateMaterialListParams{
			Description: materialReq.Description,
			Size:        materialReq.Size,
			Color:       materialReq.Color,
			Uom:         materialReq.UOM,
			IDWo:        header.IDWo,
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
	current, err := u.repo.GetWorkOrderDetail(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWorkOrderNotFound
		}
		return nil, fmt.Errorf("%w: failed to get work order", ErrWorkOrderServiceUnavailable)
	}
	if current.Status == "closed" {
		return nil, ErrWorkOrderAlreadyClosed
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
	page, limit, offset := normalizePagination(filter)
	rows, err := u.repo.ListWorkOrders(ctx, entity.ListWorkOrdersParams{
		SearchTerm: filter.Search,
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

	page, limit, offset := normalizePagination(model.TransactionListFilter{
		Page:   filter.Page,
		Limit:  filter.Limit,
		Search: filter.Search,
	})

	rows, err := u.repo.ListProductionSummary(ctx, entity.ListProductionSummaryParams{
		IDWo:          filter.IDWO,
		IDWoShellSize: filter.IDWOShellSize,
		SearchTerm:    filter.Search,
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

func (u *WorkOrderProductionUseCase) GetWorkOrderDetail(ctx context.Context, id int32) (*model.WorkOrderDetailResponse, error) {
	header, err := u.repo.GetWorkOrderDetail(ctx, id)
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
			ID:        row.IDWoShell,
			Fabric:    row.Fabric,
			Cons:      numericToFloat64(row.Cons),
			Color:     row.Color,
			Allow:     row.Allow,
			Berat1Yd:  numericToFloat64(row.Berat1Yd),
			CreatedAt: row.CreatedAt.Time.Format(time.RFC3339),
			Sizes:     sizeMap[row.IDWoShell],
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
	}, nil
}

func mapWorkOrderDBError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503":
			return ErrWorkOrderReferenceNotFound
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
