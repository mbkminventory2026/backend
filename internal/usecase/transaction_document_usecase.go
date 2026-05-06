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
	ErrTransactionValidation         = errors.New("invalid transaction payload")
	ErrTransactionReferenceNotFound  = errors.New("related data not found")
	ErrTransactionServiceUnavailable = errors.New("transaction service unavailable")
	ErrPOClientAlreadyExists         = errors.New("po client number already exists")
)

type TransactionDocumentUseCase struct {
	repo   entity.Querier
	dbPool *pgxpool.Pool
}

func NewTransactionDocumentUseCase(repo entity.Querier, dbPool *pgxpool.Pool) (*TransactionDocumentUseCase, error) {
	if repo == nil {
		return nil, errors.New("transaction repository is required")
	}
	if dbPool == nil {
		return nil, errors.New("database pool is required")
	}

	return &TransactionDocumentUseCase{
		repo:   repo,
		dbPool: dbPool,
	}, nil
}

func (u *TransactionDocumentUseCase) CreatePOClient(ctx context.Context, req model.CreatePOClientRequest) (*model.POClientResponse, error) {
	if len(req.Items) == 0 || len(req.PenanggungJawab) == 0 {
		return nil, ErrTransactionValidation
	}
	if err := validateDate(req.Tanggal); err != nil {
		return nil, err
	}
	if err := validateDate(req.Delivery); err != nil {
		return nil, err
	}

	tx, err := u.dbPool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to start transaction", ErrTransactionServiceUnavailable)
	}
	defer func() {
		rollbackErr := tx.Rollback(ctx)
		if rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			err = rollbackErr
		}
	}()

	qtx := entity.New(tx)

	header, err := qtx.CreatePOClient(ctx, entity.CreatePOClientParams{
		PoNumber:    req.PONumber,
		Tanggal:     mustDate(req.Tanggal),
		Season:      req.Season,
		Delivery:    mustDate(req.Delivery),
		PaymentTerm: req.PaymentTerm,
		File:        req.File,
		IDMitra:     req.IDMitra,
	})
	if err != nil {
		return nil, mapTransactionDBError(err)
	}

	items := make([]model.POClientItemResponse, 0, len(req.Items))
	for _, itemReq := range req.Items {
		item, itemErr := qtx.CreatePOClientItem(ctx, entity.CreatePOClientItemParams{
			IDPoClient:  header.IDPoClient,
			Style:       itemReq.Style,
			Colour:      itemReq.Colour,
			Description: itemReq.Description,
			Qty:         itemReq.Qty,
			Price:       mustNumeric(itemReq.Price),
		})
		if itemErr != nil {
			return nil, mapTransactionDBError(itemErr)
		}

		items = append(items, model.POClientItemResponse{
			ID:          item.IDPoClientItem,
			Style:       item.Style,
			Colour:      item.Colour,
			Description: item.Description,
			Qty:         item.Qty,
			Price:       numericToFloat64(item.Price),
			CreatedAt:   item.CreatedAt.Time.Format(time.RFC3339),
		})
	}

	pics := make([]model.PenanggungJawabResponse, 0, len(req.PenanggungJawab))
	for _, picReq := range req.PenanggungJawab {
		pic, picErr := qtx.CreatePenanggungJawab(ctx, entity.CreatePenanggungJawabParams{
			Nama:       picReq.Nama,
			NoTelp:     picReq.NoTelp,
			Email:      picReq.Email,
			IDPoClient: header.IDPoClient,
		})
		if picErr != nil {
			return nil, mapTransactionDBError(picErr)
		}

		pics = append(pics, model.PenanggungJawabResponse{
			ID:        pic.IDPenanggungJawab,
			Nama:      pic.Nama,
			NoTelp:    pic.NoTelp,
			Email:     pic.Email,
			CreatedAt: pic.CreatedAt.Time.Format(time.RFC3339),
		})
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("%w: failed to commit transaction", ErrTransactionServiceUnavailable)
	}

	return &model.POClientResponse{
		ID:              header.IDPoClient,
		PONumber:        header.PoNumber,
		Tanggal:         header.Tanggal.Time.Format("2006-01-02"),
		Season:          header.Season,
		Delivery:        header.Delivery.Time.Format("2006-01-02"),
		PaymentTerm:     header.PaymentTerm,
		File:            header.File,
		IDMitra:         header.IDMitra,
		CreatedAt:       header.CreatedAt.Time.Format(time.RFC3339),
		Items:           items,
		PenanggungJawab: pics,
	}, nil
}

func (u *TransactionDocumentUseCase) CreatePRInternal(ctx context.Context, req model.CreatePRInternalRequest) (*model.PRInternalResponse, error) {
	if len(req.Items) == 0 {
		return nil, ErrTransactionValidation
	}
	if err := validateDate(req.Tanggal); err != nil {
		return nil, err
	}

	tx, err := u.dbPool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to start transaction", ErrTransactionServiceUnavailable)
	}
	defer func() {
		rollbackErr := tx.Rollback(ctx)
		if rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			err = rollbackErr
		}
	}()

	qtx := entity.New(tx)

	header, err := qtx.CreatePRInternal(ctx, entity.CreatePRInternalParams{
		Tanggal:       mustDate(req.Tanggal),
		Nama:          req.Nama,
		Departemen:    req.Departemen,
		VendorName:    req.VendorName,
		VendorAddress: req.VendorAddress,
		VendorTelp:    req.VendorTelp,
		Projek:        req.Projek,
		IDWo:          req.IDWO,
		IDUser:        req.IDUser,
	})
	if err != nil {
		return nil, mapTransactionDBError(err)
	}

	items := make([]model.PRInternalItemResponse, 0, len(req.Items))
	for _, itemReq := range req.Items {
		item, itemErr := qtx.CreatePRInternalItem(ctx, entity.CreatePRInternalItemParams{
			IDPrInternal: header.IDPrInternal,
			Item:         itemReq.Item,
			Description:  itemReq.Description,
			Qty:          itemReq.Qty,
			Unit:         itemReq.Unit,
			EstPrice:     mustNumeric(itemReq.EstPrice),
		})
		if itemErr != nil {
			return nil, mapTransactionDBError(itemErr)
		}

		items = append(items, model.PRInternalItemResponse{
			ID:          item.IDPrInternalItem,
			Item:        item.Item,
			Description: item.Description,
			Qty:         item.Qty,
			Unit:        item.Unit,
			EstPrice:    numericToFloat64(item.EstPrice),
			CreatedAt:   item.CreatedAt.Time.Format(time.RFC3339),
		})
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("%w: failed to commit transaction", ErrTransactionServiceUnavailable)
	}

	return &model.PRInternalResponse{
		ID:            header.IDPrInternal,
		Tanggal:       header.Tanggal.Time.Format("2006-01-02"),
		Nama:          header.Nama,
		Departemen:    header.Departemen,
		VendorName:    header.VendorName,
		VendorAddress: header.VendorAddress,
		VendorTelp:    header.VendorTelp,
		Projek:        header.Projek,
		IDWO:          header.IDWo,
		IDUser:        header.IDUser,
		CreatedAt:     header.CreatedAt.Time.Format(time.RFC3339),
		Items:         items,
	}, nil
}

func (u *TransactionDocumentUseCase) CreatePOInternal(ctx context.Context, req model.CreatePOInternalRequest) (*model.POInternalResponse, error) {
	if len(req.Items) == 0 {
		return nil, ErrTransactionValidation
	}
	if err := validateDate(req.Tanggal); err != nil {
		return nil, err
	}
	if err := validateDate(req.ShipDate); err != nil {
		return nil, err
	}

	tx, err := u.dbPool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to start transaction", ErrTransactionServiceUnavailable)
	}
	defer func() {
		rollbackErr := tx.Rollback(ctx)
		if rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			err = rollbackErr
		}
	}()

	qtx := entity.New(tx)

	header, err := qtx.CreatePOInternal(ctx, entity.CreatePOInternalParams{
		Tanggal:         mustDate(req.Tanggal),
		NamaPo:          req.NamaPO,
		SupplierName:    req.SupplierName,
		SupplierAddr:    req.SupplierAddr,
		SupplierContact: req.SupplierContact,
		SupplierEmail:   req.SupplierEmail,
		SupplierTelp:    req.SupplierTelp,
		SupplierFax:     req.SupplierFax,
		Currency:        req.Currency,
		Cpo:             req.CPO,
		Term:            req.Term,
		ShipDate:        mustDate(req.ShipDate),
		IDPrInternal:    req.IDPRInternal,
	})
	if err != nil {
		return nil, mapTransactionDBError(err)
	}

	items := make([]model.POInternalItemResponse, 0, len(req.Items))
	for _, itemReq := range req.Items {
		item, itemErr := qtx.CreatePOInternalItem(ctx, entity.CreatePOInternalItemParams{
			IDPoInternal: header.IDPoInternal,
			Item:         itemReq.Item,
			Description:  itemReq.Description,
			Qty:          itemReq.Qty,
			Unit:         itemReq.Unit,
			UnitPrice:    mustNumeric(itemReq.UnitPrice),
		})
		if itemErr != nil {
			return nil, mapTransactionDBError(itemErr)
		}

		items = append(items, model.POInternalItemResponse{
			ID:          item.IDPoInternalItem,
			Item:        item.Item,
			Description: item.Description,
			Qty:         item.Qty,
			Unit:        item.Unit,
			UnitPrice:   numericToFloat64(item.UnitPrice),
			CreatedAt:   item.CreatedAt.Time.Format(time.RFC3339),
		})
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("%w: failed to commit transaction", ErrTransactionServiceUnavailable)
	}

	return &model.POInternalResponse{
		ID:              header.IDPoInternal,
		Tanggal:         header.Tanggal.Time.Format("2006-01-02"),
		NamaPO:          header.NamaPo,
		SupplierName:    header.SupplierName,
		SupplierAddr:    header.SupplierAddr,
		SupplierContact: header.SupplierContact,
		SupplierEmail:   header.SupplierEmail,
		SupplierTelp:    header.SupplierTelp,
		SupplierFax:     header.SupplierFax,
		Currency:        header.Currency,
		CPO:             header.Cpo,
		Term:            header.Term,
		ShipDate:        header.ShipDate.Time.Format("2006-01-02"),
		IDPRInternal:    header.IDPrInternal,
		CreatedAt:       header.CreatedAt.Time.Format(time.RFC3339),
		Items:           items,
	}, nil
}

func validateDate(value string) error {
	if _, err := time.Parse("2006-01-02", value); err != nil {
		return ErrTransactionValidation
	}
	return nil
}

func mapTransactionDBError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return ErrPOClientAlreadyExists
		case "23503":
			return ErrTransactionReferenceNotFound
		}
	}
	return fmt.Errorf("%w: %v", ErrTransactionServiceUnavailable, err)
}

func mustDate(value string) pgtype.Date {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return pgtype.Date{}
	}
	return pgtype.Date{Time: parsed, Valid: true}
}

func mustNumeric(value float64) pgtype.Numeric {
	var numeric pgtype.Numeric
	if err := numeric.Scan(fmt.Sprintf("%.2f", value)); err != nil {
		return pgtype.Numeric{}
	}
	return numeric
}

func numericToFloat64(value pgtype.Numeric) float64 {
	floatVal, err := value.Float64Value()
	if err != nil || !floatVal.Valid {
		return 0
	}
	return floatVal.Float64
}
