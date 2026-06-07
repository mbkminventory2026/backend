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
	ErrTransactionNotFound           = errors.New("transaction not found")
	ErrPOClientAlreadyExists         = errors.New("po client number already exists")
	ErrPOClientLockedForUpdate       = errors.New("po client cannot be updated because it is already used by work orders")
	ErrPRInternalAlreadyApproved     = errors.New("pr internal is already approved")

	poClientSortColumns   = buildSortWhitelist("created_at", "id_po_client", "po_number", "tanggal", "season", "delivery", "mitra_name")
	prInternalSortColumns = buildSortWhitelist("created_at", "id_pr_internal", "tanggal", "nama", "departemen", "vendor_name", "projek", "status")
	poInternalSortColumns = buildSortWhitelist("created_at", "id_po_internal", "tanggal", "nama_po", "supplier_name", "currency", "cpo", "ship_date")
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

func (u *TransactionDocumentUseCase) UpdatePOClient(ctx context.Context, id int32, req model.CreatePOClientRequest) (*model.POClientResponse, error) {
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

	workOrderCount, err := qtx.CountWorkOrdersByPOClientID(ctx, id)
	if err != nil {
		return nil, mapTransactionDBError(err)
	}
	if workOrderCount > 0 {
		return nil, ErrPOClientLockedForUpdate
	}

	header, err := qtx.UpdatePOClient(ctx, entity.UpdatePOClientParams{
		IDPoClient:  id,
		PoNumber:    req.PONumber,
		Tanggal:     mustDate(req.Tanggal),
		Season:      req.Season,
		Delivery:    mustDate(req.Delivery),
		PaymentTerm: req.PaymentTerm,
		File:        req.File,
		IDMitra:     req.IDMitra,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTransactionNotFound
		}
		return nil, mapTransactionDBError(err)
	}

	if err := qtx.DeletePOClientItemsByPOClientID(ctx, id); err != nil {
		return nil, mapTransactionDBError(err)
	}
	if err := qtx.DeletePenanggungJawabByPOClientID(ctx, id); err != nil {
		return nil, mapTransactionDBError(err)
	}

	items := make([]model.POClientItemResponse, 0, len(req.Items))
	for _, itemReq := range req.Items {
		item, itemErr := qtx.CreatePOClientItem(ctx, entity.CreatePOClientItemParams{
			IDPoClient:  id,
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
			IDPoClient: id,
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

func (u *TransactionDocumentUseCase) CreatePRInternal(ctx context.Context, actorUserID int32, req model.CreatePRInternalRequest) (*model.PRInternalResponse, error) {
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

	user, err := qtx.GetUserByID(ctx, actorUserID)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to fetch user for PR internal", ErrTransactionServiceUnavailable)
	}

	dept := "produksi"
	if user.Username != "super-admin" && user.NamaRole != "super-admin" {
		if user.NamaDepartemen.Valid && user.NamaDepartemen.String != "" {
			dept = user.NamaDepartemen.String
		}
	}

	header, err := qtx.CreatePRInternal(ctx, entity.CreatePRInternalParams{
		Tanggal:       mustDate(req.Tanggal),
		Nama:          req.Nama,
		Departemen:    dept,
		VendorName:    req.VendorName,
		VendorAddress: req.VendorAddress,
		VendorTelp:    req.VendorTelp,
		Projek:        req.Projek,
		IDWo:          req.IDWO,
		IDUser:        actorUserID,
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

	err = initializeApprovalWorkflow(ctx, qtx, "PR_INTERNAL", header.IDPrInternal, actorUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize approval workflow: %w", err)
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
		Status:        "draft",
		CreatedAt:     header.CreatedAt.Time.Format(time.RFC3339),
		Items:         items,
	}, nil
}

func (u *TransactionDocumentUseCase) ApprovePRInternal(ctx context.Context, id int32, approverUserID int32) (*model.PRInternalStatusResponse, error) {
	current, err := u.repo.GetPRInternalDetail(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTransactionNotFound
		}
		return nil, fmt.Errorf("%w: failed to get pr internal", ErrTransactionServiceUnavailable)
	}
	if current.Status == "approved" {
		return nil, ErrPRInternalAlreadyApproved
	}

	updated, err := u.repo.ApprovePRInternal(ctx, entity.ApprovePRInternalParams{
		IDPrInternal:     id,
		ApprovedByUserID: pgtype.Int4{Int32: approverUserID, Valid: true},
	})
	if err != nil {
		return nil, mapTransactionDBError(err)
	}

	return &model.PRInternalStatusResponse{
		ID:           updated.IDPrInternal,
		Status:       updated.Status,
		ApprovedByID: nullableInt32Ptr(updated.ApprovedByUserID),
		ApprovedAt:   nullableTimestampString(updated.ApprovedAt),
	}, nil
}

func (u *TransactionDocumentUseCase) CreatePOInternal(ctx context.Context, userID int32, req model.CreatePOInternalRequest) (*model.POInternalResponse, error) {
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

	err = initializeApprovalWorkflow(ctx, qtx, "PO_INTERNAL", header.IDPoInternal, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize approval workflow: %w", err)
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

func (u *TransactionDocumentUseCase) ListPOClients(ctx context.Context, filter model.TransactionListFilter) (*model.POClientListResponse, error) {
	page, limit, offset, search, sortBy, sortDesc := normalizeListFilter(filter.ListQueryFilter, "id_po_client", true, poClientSortColumns)

	var idMitraVal pgtype.Int4
	if filter.IDMitra != nil {
		idMitraVal = pgtype.Int4{Int32: *filter.IDMitra, Valid: true}
	}

	rows, err := u.repo.ListPOClients(ctx, entity.ListPOClientsParams{
		SearchTerm: search,
		IDMitra:    idMitraVal,
		SortBy:     sortBy,
		SortDesc:   sortDesc,
		PageLimit:  limit,
		PageOffset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: failed to list po clients", ErrTransactionServiceUnavailable)
	}

	items := make([]model.POClientListItem, 0, len(rows))
	total := int64(0)
	for _, row := range rows {
		total = row.TotalCount
		items = append(items, model.POClientListItem{
			ID:        row.IDPoClient,
			PONumber:  row.PoNumber,
			Tanggal:   row.Tanggal.Time.Format("2006-01-02"),
			Season:    row.Season,
			Delivery:  row.Delivery.Time.Format("2006-01-02"),
			IDMitra:   row.IDMitra,
			MitraName: row.MitraName,
			CreatedAt: row.CreatedAt.Time.Format(time.RFC3339),
			HasRetur:  row.HasRetur,
		})
	}

	return &model.POClientListResponse{
		Items:      items,
		Pagination: buildPagination(total, page, limit),
	}, nil
}

func (u *TransactionDocumentUseCase) GetPOClientDetail(ctx context.Context, id int32, idMitra *int32) (*model.POClientDetailResponse, error) {
	header, err := u.repo.GetPOClientDetail(ctx, entity.GetPOClientDetailParams{
		IDPoClient: id,
		IDMitra:    nullableInt32Param(idMitra),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTransactionNotFound
		}
		return nil, fmt.Errorf("%w: failed to get po client", ErrTransactionServiceUnavailable)
	}

	itemRows, err := u.repo.ListPOClientItemsByPOClientID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get po client items", ErrTransactionServiceUnavailable)
	}
	picRows, err := u.repo.ListPenanggungJawabByPOClientID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get po client contacts", ErrTransactionServiceUnavailable)
	}

	items := make([]model.POClientItemResponse, 0, len(itemRows))
	for _, row := range itemRows {
		items = append(items, model.POClientItemResponse{
			ID:          row.IDPoClientItem,
			Style:       row.Style,
			Colour:      row.Colour,
			Description: row.Description,
			Qty:         row.Qty,
			Price:       numericToFloat64(row.Price),
			CreatedAt:   row.CreatedAt.Time.Format(time.RFC3339),
			IDWo:        nullableInt32Ptr(row.IDWo),
			WoStatus:    nullableStringPtr(row.WoStatus),
			HasRetur:    row.HasRetur,
		})
	}

	pics := make([]model.PenanggungJawabResponse, 0, len(picRows))
	for _, row := range picRows {
		pics = append(pics, model.PenanggungJawabResponse{
			ID:        row.IDPenanggungJawab,
			Nama:      row.Nama,
			NoTelp:    row.NoTelp,
			Email:     row.Email,
			CreatedAt: row.CreatedAt.Time.Format(time.RFC3339),
		})
	}

	return &model.POClientDetailResponse{
		ID:              header.IDPoClient,
		PONumber:        header.PoNumber,
		Tanggal:         header.Tanggal.Time.Format("2006-01-02"),
		Season:          header.Season,
		Delivery:        header.Delivery.Time.Format("2006-01-02"),
		PaymentTerm:     header.PaymentTerm,
		File:            header.File,
		IDMitra:         header.IDMitra,
		MitraName:       header.MitraName,
		CreatedAt:       header.CreatedAt.Time.Format(time.RFC3339),
		Items:           items,
		PenanggungJawab: pics,
	}, nil
}

func (u *TransactionDocumentUseCase) ListPRInternals(ctx context.Context, filter model.TransactionListFilter) (*model.PRInternalListResponse, error) {
	page, limit, offset, search, sortBy, sortDesc := normalizeListFilter(filter.ListQueryFilter, "id_pr_internal", true, prInternalSortColumns)
	rows, err := u.repo.ListPRInternals(ctx, entity.ListPRInternalsParams{
		SearchTerm: search,
		SortBy:     sortBy,
		SortDesc:   sortDesc,
		PageLimit:  limit,
		PageOffset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: failed to list pr internals", ErrTransactionServiceUnavailable)
	}

	items := make([]model.PRInternalListItem, 0, len(rows))
	total := int64(0)
	for _, row := range rows {
		total = row.TotalCount
		items = append(items, model.PRInternalListItem{
			ID:         row.IDPrInternal,
			Tanggal:    row.Tanggal.Time.Format("2006-01-02"),
			Nama:       row.Nama,
			Departemen: row.Departemen,
			VendorName: row.VendorName,
			Projek:     row.Projek,
			IDWO:       row.IDWo,
			IDUser:     row.IDUser,
			Status:     row.Status,
			ApprovedAt: nullableTimestampString(row.ApprovedAt),
			CreatedAt:  row.CreatedAt.Time.Format(time.RFC3339),
		})
	}

	return &model.PRInternalListResponse{
		Items:      items,
		Pagination: buildPagination(total, page, limit),
	}, nil
}

func (u *TransactionDocumentUseCase) GetPRInternalDetail(ctx context.Context, id int32) (*model.PRInternalResponse, error) {
	header, err := u.repo.GetPRInternalDetail(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTransactionNotFound
		}
		return nil, fmt.Errorf("%w: failed to get pr internal", ErrTransactionServiceUnavailable)
	}
	rows, err := u.repo.ListPRInternalItemsByPRInternalID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get pr internal items", ErrTransactionServiceUnavailable)
	}

	items := make([]model.PRInternalItemResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, model.PRInternalItemResponse{
			ID:          row.IDPrInternalItem,
			Item:        row.Item,
			Description: row.Description,
			Qty:         row.Qty,
			Unit:        row.Unit,
			EstPrice:    numericToFloat64(row.EstPrice),
			CreatedAt:   row.CreatedAt.Time.Format(time.RFC3339),
		})
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
		Status:        header.Status,
		ApprovedByID:  nullableInt32Ptr(header.ApprovedByUserID),
		ApprovedAt:    nullableTimestampString(header.ApprovedAt),
		CreatedAt:     header.CreatedAt.Time.Format(time.RFC3339),
		Items:         items,
	}, nil
}

func (u *TransactionDocumentUseCase) ListPOInternals(ctx context.Context, filter model.TransactionListFilter) (*model.POInternalListResponse, error) {
	page, limit, offset, search, sortBy, sortDesc := normalizeListFilter(filter.ListQueryFilter, "id_po_internal", true, poInternalSortColumns)
	rows, err := u.repo.ListPOInternals(ctx, entity.ListPOInternalsParams{
		SearchTerm: search,
		SortBy:     sortBy,
		SortDesc:   sortDesc,
		PageLimit:  limit,
		PageOffset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: failed to list po internals", ErrTransactionServiceUnavailable)
	}

	items := make([]model.POInternalListItem, 0, len(rows))
	total := int64(0)
	for _, row := range rows {
		total = row.TotalCount
		items = append(items, model.POInternalListItem{
			ID:           row.IDPoInternal,
			Tanggal:      row.Tanggal.Time.Format("2006-01-02"),
			NamaPO:       row.NamaPo,
			SupplierName: row.SupplierName,
			Currency:     row.Currency,
			CPO:          row.Cpo,
			ShipDate:     row.ShipDate.Time.Format("2006-01-02"),
			IDPRInternal: row.IDPrInternal,
			CreatedAt:    row.CreatedAt.Time.Format(time.RFC3339),
		})
	}

	return &model.POInternalListResponse{
		Items:      items,
		Pagination: buildPagination(total, page, limit),
	}, nil
}

func (u *TransactionDocumentUseCase) GetPOInternalDetail(ctx context.Context, id int32) (*model.POInternalResponse, error) {
	header, err := u.repo.GetPOInternalDetail(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTransactionNotFound
		}
		return nil, fmt.Errorf("%w: failed to get po internal", ErrTransactionServiceUnavailable)
	}
	rows, err := u.repo.ListPOInternalItemsByPOInternalID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get po internal items", ErrTransactionServiceUnavailable)
	}

	items := make([]model.POInternalItemResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, model.POInternalItemResponse{
			ID:          row.IDPoInternalItem,
			Item:        row.Item,
			Description: row.Description,
			Qty:         row.Qty,
			Unit:        row.Unit,
			UnitPrice:   numericToFloat64(row.UnitPrice),
			CreatedAt:   row.CreatedAt.Time.Format(time.RFC3339),
		})
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
