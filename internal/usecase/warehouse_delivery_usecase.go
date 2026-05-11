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
	ErrSuratJalanTypeUnsupported   = errors.New("unsupported surat jalan type")
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
		IDMaterialList:         req.IDMaterialList,
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

func (u *WarehouseDeliveryUseCase) CreatePackingList(ctx context.Context, req model.CreatePackingListRequest) (*model.PackingListResponse, error) {
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
			Tanggal:        mustDate(req.Tanggal),
			Qty:            req.Qty,
			Keterangan:     req.Keterangan,
			IDMaterialList: req.IDMaterialList,
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
