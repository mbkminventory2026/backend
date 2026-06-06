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
	ErrApprovalNotFound           = errors.New("approval record not found")
	ErrPreviousStepPending        = errors.New("langkah persetujuan sebelumnya belum disetujui")
	ErrUnauthorizedApproval       = errors.New("Anda tidak memiliki wewenang untuk menyetujui langkah ini")
	ErrApprovalServiceUnavailable = errors.New("layanan approval tidak tersedia saat ini")
	ErrDocumentNotPending         = errors.New("dokumen tidak sedang dalam status pending approval")
)

type ApprovalUseCase struct {
	repo   entity.Querier
	dbPool *pgxpool.Pool
}

func NewApprovalUseCase(repo entity.Querier, dbPool *pgxpool.Pool) (*ApprovalUseCase, error) {
	if repo == nil {
		return nil, errors.New("approval repository is required")
	}
	if dbPool == nil {
		return nil, errors.New("database pool is required")
	}
	return &ApprovalUseCase{
		repo:   repo,
		dbPool: dbPool,
	}, nil
}

func (u *ApprovalUseCase) getDocumentSummaryAndRequester(ctx context.Context, tableName string, docID int32) (string, string) {
	switch tableName {
	case "PR_INTERNAL":
		pr, prErr := u.repo.GetPRInternalDetail(ctx, docID)
		if prErr == nil {
			return fmt.Sprintf("%s - Projek: %s", pr.Nama, pr.Projek), pr.Nama
		}
	case "WORK_ORDER":
		wo, woErr := u.repo.GetWorkOrderDetail(ctx, entity.GetWorkOrderDetailParams{
			IDWo:    docID,
			IDMitra: pgtype.Int4{Valid: false},
		})
		if woErr == nil {
			return fmt.Sprintf("Buyer: %s - Model: %s (Qty: %d)", wo.Buyer, wo.Model, wo.Qty), "Admin Produksi"
		}
	case "PO_INTERNAL":
		po, poErr := u.repo.GetPOInternalDetail(ctx, docID)
		if poErr == nil {
			return fmt.Sprintf("%s - Supplier: %s", po.NamaPo, po.SupplierName), "Admin Keuangan"
		}
	case "MARKER_PLAN":
		mp, mpErr := u.repo.GetMarkerPlanByID(ctx, docID)
		if mpErr == nil {
			return fmt.Sprintf("Marker Plan Dokumen: %s", mp.NoDokumen), "Admin Produksi"
		}
	}
	return fmt.Sprintf("Dokumen %s #%d", tableName, docID), "System"
}

func (u *ApprovalUseCase) GetPendingApprovals(ctx context.Context, userID int32) ([]model.ApprovalPendingResponse, error) {
	rows, err := u.repo.GetTruePendingApprovalsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to fetch pending approvals", ErrApprovalServiceUnavailable)
	}

	result := make([]model.ApprovalPendingResponse, 0, len(rows))
	for _, row := range rows {
		docSummary, requestedBy := u.getDocumentSummaryAndRequester(ctx, row.NamaTabelDokumen, row.IDDokumen)
		item := model.ApprovalPendingResponse{
			IDIDDetail:       row.IDOtoritasDetail,
			IDHeader:         row.IDOtoritas,
			NamaTabelDokumen: row.NamaTabelDokumen,
			IDDokumen:        row.IDDokumen,
			TipePeran:        row.TipePeran,
			DocSummary:       docSummary,
			RequestedBy:      requestedBy,
			RequestedAt:      row.RequestedAt.Time,
		}
		result = append(result, item)
	}

	return result, nil
}

func (u *ApprovalUseCase) GetApprovalHistory(ctx context.Context, status string, table string, limit int32, offset int32) (*model.ApprovalHistoryResponse, error) {
	rows, err := u.repo.ListApprovalHistory(ctx, entity.ListApprovalHistoryParams{
		StatusFilter: status,
		TableFilter:  table,
		PageLimit:    limit,
		PageOffset:   offset,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: failed to fetch approval history", ErrApprovalServiceUnavailable)
	}

	var totalItems int64 = 0
	if len(rows) > 0 {
		totalItems = rows[0].TotalCount
	}

	items := make([]model.ApprovalHistoryListItem, 0, len(rows))
	for _, row := range rows {
		docSummary, requestedBy := u.getDocumentSummaryAndRequester(ctx, row.NamaTabelDokumen, row.IDDokumen)
		item := model.ApprovalHistoryListItem{
			IDHeader:         row.IDOtoritas,
			NamaTabelDokumen: row.NamaTabelDokumen,
			IDDokumen:        row.IDDokumen,
			StatusGlobal:     row.StatusGlobal,
			DocSummary:       docSummary,
			RequestedBy:      requestedBy,
			CreatedAt:        row.CreatedAt.Time,
		}
		items = append(items, item)
	}

	return &model.ApprovalHistoryResponse{
		Items:      items,
		TotalItems: totalItems,
	}, nil
}

func (u *ApprovalUseCase) ProcessApprovalAction(ctx context.Context, userID int32, req model.ApprovalActionRequest) error {
	// 1. Fetch step detail
	detail, err := u.repo.GetApprovalDetailByID(ctx, req.IDDetail)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrApprovalNotFound
		}
		return fmt.Errorf("%w: get approval step", ErrApprovalServiceUnavailable)
	}

	if detail.IDUser != userID {
		return ErrUnauthorizedApproval
	}

	if detail.IsActionDone {
		return nil // Already processed
	}

	// 1.5 Check if header status is pending
	header, err := u.repo.GetApprovalHeaderByID(ctx, detail.IDOtoritas)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrApprovalNotFound
		}
		return fmt.Errorf("%w: get approval header", ErrApprovalServiceUnavailable)
	}

	if header.StatusGlobal != "pending" {
		return ErrDocumentNotPending
	}

	// 1.7 Check user's functional permission for the specific document type
	var requiredPermission string
	switch header.NamaTabelDokumen {
	case "PR_INTERNAL":
		requiredPermission = "PR_INTERNAL_APPROVE"
	case "PO_INTERNAL":
		requiredPermission = "PO_INTERNAL_APPROVE"
	case "PACKING_LIST":
		requiredPermission = "PACKING_LIST_APPROVE"
	case "WORK_ORDER":
		requiredPermission = "WO_UPDATE"
	case "MARKER_PLAN":
		requiredPermission = "MARKER_PLAN_UPDATE"
	case "TIMELINE_PRODUKSI":
		requiredPermission = "TIMELINE_UPDATE"
	default:
		return fmt.Errorf("tipe dokumen %s tidak didukung", header.NamaTabelDokumen)
	}

	userPerms, err := u.repo.GetUserPermissions(ctx, userID)
	if err != nil {
		return fmt.Errorf("%w: get user permissions", ErrApprovalServiceUnavailable)
	}

	hasAccess := false
	for _, p := range userPerms {
		if p == "ALL_ACCESS" || p == requiredPermission {
			hasAccess = true
			break
		}
	}

	if !hasAccess {
		return ErrUnauthorizedApproval
	}

	// 2. Verify sequencing (prev steps must be done)
	hasPrevPending, err := u.repo.HasPreviousPendingSteps(ctx, entity.HasPreviousPendingStepsParams{
		IDOtoritas:       detail.IDOtoritas,
		IDOtoritasDetail: detail.IDOtoritasDetail,
	})
	if err != nil {
		return fmt.Errorf("%w: verify sequence", ErrApprovalServiceUnavailable)
	}
	if hasPrevPending {
		return ErrPreviousStepPending
	}

	// 3. Start Transaction
	tx, err := u.dbPool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("%w: begin transaction", ErrApprovalServiceUnavailable)
	}
	defer func() {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			err = rollbackErr
		}
	}()

	qtx := entity.New(tx)

	// 4. Update the step
	now := pgtype.Timestamptz{Time: time.Now(), Valid: true}
	_, err = qtx.UpdateApprovalStep(ctx, entity.UpdateApprovalStepParams{
		IDOtoritasDetail: detail.IDOtoritasDetail,
		IsActionDone:     true,
		WaktuAksi:        now,
		Catatan:          req.Catatan,
	})
	if err != nil {
		return fmt.Errorf("%w: update step", ErrApprovalServiceUnavailable)
	}

	// 5. Handle global status update
	if req.Action == "reject" {
		_, err = qtx.UpdateGlobalStatus(ctx, entity.UpdateGlobalStatusParams{
			IDOtoritas:   detail.IDOtoritas,
			StatusGlobal: "rejected",
		})
		if err != nil {
			return fmt.Errorf("%w: set global status rejected", ErrApprovalServiceUnavailable)
		}
	} else {
		// Action is approve
		hasPending, err := qtx.HasPendingSteps(ctx, detail.IDOtoritas)
		if err != nil {
			return fmt.Errorf("%w: check pending steps", ErrApprovalServiceUnavailable)
		}

		if !hasPending {
			// All steps are completed successfully
			_, err = qtx.UpdateGlobalStatus(ctx, entity.UpdateGlobalStatusParams{
				IDOtoritas:   detail.IDOtoritas,
				StatusGlobal: "approved",
			})
			if err != nil {
				return fmt.Errorf("%w: set global status approved", ErrApprovalServiceUnavailable)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("%w: commit transaction", ErrApprovalServiceUnavailable)
	}

	return nil
}

func (u *ApprovalUseCase) GetDocumentAuditTrail(ctx context.Context, tableName string, docID int32) (*model.DocumentAuditTrailResponse, error) {
	header, err := u.repo.GetApprovalHeaderByDoc(ctx, entity.GetApprovalHeaderByDocParams{
		NamaTabelDokumen: tableName,
		IDDokumen:        docID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &model.DocumentAuditTrailResponse{
				NamaTabel:    tableName,
				IDDokumen:    docID,
				StatusGlobal: "draft",
				Steps:        []model.AuditTrailStep{},
			}, nil
		}
		return nil, fmt.Errorf("%w: get approval header", ErrApprovalServiceUnavailable)
	}

	details, err := u.repo.ListApprovalDetailsByHeaderID(ctx, header.IDOtoritas)
	if err != nil {
		return nil, fmt.Errorf("%w: list approval steps", ErrApprovalServiceUnavailable)
	}

	steps := make([]model.AuditTrailStep, 0, len(details))
	for _, d := range details {
		step := model.AuditTrailStep{
			IDDetail:  d.IDOtoritasDetail,
			IDUser:    d.IDUser,
			TipePeran: d.TipePeran,
			Done:      d.IsActionDone,
			Catatan:   d.Catatan,
		}

		if d.WaktuAksi.Valid {
			t := d.WaktuAksi.Time
			step.WaktuAksi = &t
		}

		user, userErr := u.repo.GetUserByID(ctx, d.IDUser)
		if userErr == nil {
			step.NamaUser = user.Username
		} else {
			step.NamaUser = "User Tidak Ditemukan"
		}

		steps = append(steps, step)
	}

	return &model.DocumentAuditTrailResponse{
		IDHeader:     header.IDOtoritas,
		NamaTabel:    header.NamaTabelDokumen,
		IDDokumen:    header.IDDokumen,
		StatusGlobal: header.StatusGlobal,
		Steps:        steps,
	}, nil
}

func (u *ApprovalUseCase) InitializeApprovalWorkflow(ctx context.Context, qtx entity.Querier, tableName string, docID int32, creatorUserID int32) error {
	header, err := qtx.CreateApprovalHeader(ctx, entity.CreateApprovalHeaderParams{
		NamaTabelDokumen: tableName,
		IDDokumen:        docID,
		StatusGlobal:     "pending",
	})
	if err != nil {
		return err
	}

	fallbackID := creatorUserID
	if fallbackID <= 0 {
		mgrUsers, err := qtx.GetUsersByRoleName(ctx, "MANAGER")
		if err == nil && len(mgrUsers) > 0 {
			fallbackID = mgrUsers[0].IDUser
		} else {
			keuUsers, err := qtx.GetUsersByRoleName(ctx, "ADMIN_KEUANGAN")
			if err == nil && len(keuUsers) > 0 {
				fallbackID = keuUsers[0].IDUser
			} else {
				fallbackID = 1 // Last resort fallback
			}
		}
	}

	// Matriks Konfigurasi Alur Approval
	switch tableName {
	case "PR_INTERNAL":
		// Step 1: PEMBUAT (Creator) - marked as done immediately
		now := pgtype.Timestamptz{Time: time.Now(), Valid: true}
		_, err = qtx.CreateApprovalDetail(ctx, entity.CreateApprovalDetailParams{
			IDOtoritas:   header,
			IDUser:       creatorUserID,
			TipePeran:    "PEMBUAT",
			IsActionDone: true,
			WaktuAksi:    now,
			Catatan:      "Dokumen PR dibuat",
		})
		if err != nil {
			return err
		}

		// Step 2: PENGECEK (Admin Produksi)
		produksiID := u.getAssigneeForRole(ctx, "ADMIN_PRODUKSI", fallbackID)
		_, err = qtx.CreateApprovalDetail(ctx, entity.CreateApprovalDetailParams{
			IDOtoritas:   header,
			IDUser:       produksiID,
			TipePeran:    "PENGECEK",
			IsActionDone: false,
		})
		if err != nil {
			return err
		}

		// Step 3: PENYETUJU (Manager)
		managerID := u.getAssigneeForRole(ctx, "MANAGER", fallbackID)
		_, err = qtx.CreateApprovalDetail(ctx, entity.CreateApprovalDetailParams{
			IDOtoritas:   header,
			IDUser:       managerID,
			TipePeran:    "PENYETUJU",
			IsActionDone: false,
		})
		if err != nil {
			return err
		}

		// Step 4: RELEASE (Admin Keuangan)
		keuanganID := u.getAssigneeForRole(ctx, "ADMIN_KEUANGAN", fallbackID)
		_, err = qtx.CreateApprovalDetail(ctx, entity.CreateApprovalDetailParams{
			IDOtoritas:   header,
			IDUser:       keuanganID,
			TipePeran:    "RELEASE",
			IsActionDone: false,
		})
		if err != nil {
			return err
		}

	case "WORK_ORDER", "PO_INTERNAL", "MARKER_PLAN", "TIMELINE_PRODUKSI", "PACKING_LIST":
		// Alur standar 2-langkah: Pembuat -> Manager
		now := pgtype.Timestamptz{Time: time.Now(), Valid: true}
		_, err = qtx.CreateApprovalDetail(ctx, entity.CreateApprovalDetailParams{
			IDOtoritas:   header,
			IDUser:       creatorUserID,
			TipePeran:    "PEMBUAT",
			IsActionDone: true,
			WaktuAksi:    now,
			Catatan:      fmt.Sprintf("Dokumen %s dibuat", tableName),
		})
		if err != nil {
			return err
		}

		managerID := u.getAssigneeForRole(ctx, "MANAGER", fallbackID)
		_, err = qtx.CreateApprovalDetail(ctx, entity.CreateApprovalDetailParams{
			IDOtoritas:   header,
			IDUser:       managerID,
			TipePeran:    "PENYETUJU",
			IsActionDone: false,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (u *ApprovalUseCase) getAssigneeForRole(ctx context.Context, roleName string, fallbackID int32) int32 {
	// 1. Try target role
	users, err := u.repo.GetUsersByRoleName(ctx, roleName)
	if err == nil && len(users) > 0 {
		return users[0].IDUser
	}

	// 2. Fallback to MANAGER
	if roleName != "MANAGER" {
		mgrUsers, err := u.repo.GetUsersByRoleName(ctx, "MANAGER")
		if err == nil && len(mgrUsers) > 0 {
			return mgrUsers[0].IDUser
		}
	}

	// 3. Fallback to ADMIN_KEUANGAN
	if roleName != "ADMIN_KEUANGAN" {
		keuUsers, err := u.repo.GetUsersByRoleName(ctx, "ADMIN_KEUANGAN")
		if err == nil && len(keuUsers) > 0 {
			return keuUsers[0].IDUser
		}
	}

	// 4. Ultimate fallback to creatorUserID
	return fallbackID
}
