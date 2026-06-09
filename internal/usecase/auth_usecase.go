package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"permatatex-inventory/internal/entity"
	"permatatex-inventory/internal/model"
	"permatatex-inventory/pkg/passwordutil"
)

var (
	ErrInvalidCredentials              = errors.New("invalid username or password")
	ErrAuthServiceUnavailable          = errors.New("authentication service unavailable")
	ErrAccountPending                  = errors.New("akun Anda sedang menunggu persetujuan admin")
	ErrAccountRejected                 = errors.New("akun Anda telah ditolak")
	ErrRoleNotFound                    = errors.New("role not found")
	ErrCurrentPasswordInvalid          = errors.New("current password is invalid")
	ErrPasswordConfirmationMismatch    = errors.New("password confirmation does not match")
	ErrPasswordResetRequestAlreadyOpen = errors.New("password reset request already pending")
	ErrPasswordResetRequestNotFound    = errors.New("password reset request not found")
)

type AuthUseCase struct {
	userRepo         entity.Querier
	dbPool           *pgxpool.Pool
	turnstileUseCase *TurnstileUseCase
	auditLog         *AuditLogUseCase
	jwtSecret        string
}

func NewAuthUseCase(userRepo entity.Querier, dbPool *pgxpool.Pool, turnstileUseCase *TurnstileUseCase, auditLog *AuditLogUseCase, jwtSecret string) *AuthUseCase {
	return &AuthUseCase{
		userRepo:         userRepo,
		dbPool:           dbPool,
		turnstileUseCase: turnstileUseCase,
		auditLog:         auditLog,
		jwtSecret:        jwtSecret,
	}
}

func (u *AuthUseCase) Login(ctx context.Context, req model.LoginRequest, remoteIP string) (*model.LoginResponse, error) {
	// 1. Verify Turnstile Token
	_, err := u.turnstileUseCase.VerifyToken(ctx, model.VerifyTurnstileRequest{
		TurnstileToken: req.TurnstileToken,
	}, remoteIP)
	if err != nil {
		return nil, fmt.Errorf("captcha verification: %w", err)
	}

	// 2. Get User from Database
	user, err := u.userRepo.GetUserByUsername(ctx, req.Username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}

		return nil, fmt.Errorf("%w: get user by username", ErrAuthServiceUnavailable)
	}

	if user.Status == "pending" {
		return nil, ErrAccountPending
	}
	if user.Status == "rejected" {
		return nil, ErrAccountRejected
	}

	if strings.TrimSpace(u.jwtSecret) == "" {
		return nil, ErrAuthServiceUnavailable
	}

	if user.Password == "" {
		return nil, ErrInvalidCredentials
	}

	// 3. Verify Password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	var idMitra *int32
	if user.IDMitra.Valid {
		val := user.IDMitra.Int32
		idMitra = &val
	}

	return u.issueLoginToken(ctx, user.IDUser, user.IDRole, user.NamaRole, idMitra, user.MustChangePassword)
}

func (u *AuthUseCase) RegisterMitra(ctx context.Context, req model.RegisterMitraRequest, remoteIP string) error {
	// 1. Verify Turnstile Token
	_, err := u.turnstileUseCase.VerifyToken(ctx, model.VerifyTurnstileRequest{
		TurnstileToken: req.TurnstileToken,
	}, remoteIP)
	if err != nil {
		return fmt.Errorf("captcha verification: %w", err)
	}

	// 2. Hash Dummy Password for pending user (cannot log in until approved)
	dummyPassword := "pending_mitra_placeholder_password_not_for_login"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(dummyPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// 3. Start Database Transaction
	tx, err := u.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			err = rollbackErr
		}
	}()

	qtx := entity.New(tx)

	// 4. Create Mitra
	var email string
	if req.Email != nil {
		email = *req.Email
	}
	var noTelp string
	if req.NoTelp != nil {
		noTelp = *req.NoTelp
	}
	var alamat string
	if req.Alamat != nil {
		alamat = *req.Alamat
	}
	var kota string
	if req.Kota != nil {
		kota = *req.Kota
	}
	var kodePos string
	if req.KodePos != nil {
		kodePos = *req.KodePos
	}

	mitra, err := qtx.CreateMitra(ctx, entity.CreateMitraParams{
		NamaPerusahaan: req.NamaPerusahaan,
		TipePerusahaan: req.TipePerusahaan,
		Email:          email,
		NoTelp:         noTelp,
		Alamat:         alamat,
		Kota:           kota,
		KodePos:        kodePos,
	})
	if err != nil {
		return fmt.Errorf("failed to create mitra: %w", err)
	}

	// 5. Create User tied to Mitra, with status 'pending'
	clientRole, err := qtx.GetRoleByName(ctx, "CLIENT")
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrRoleNotFound
		}
		return fmt.Errorf("%w: failed to get client role", ErrAuthServiceUnavailable)
	}

	idMitra := pgtype.Int4{Int32: mitra.IDMitra, Valid: true}
	idDept := pgtype.Int4{Valid: false}

	// Generate a unique temporary pending username using Mitra ID
	tempUsername := fmt.Sprintf("pending_mitra_%d", mitra.IDMitra)

	_, err = qtx.CreateUser(ctx, entity.CreateUserParams{
		Username:           tempUsername,
		Password:           string(hashedPassword),
		IDRole:             clientRole.IDRole,
		IDDepartemen:       idDept,
		IDMitra:            idMitra,
		Status:             "pending",
		MustChangePassword: false,
		CreatedBy:          pgtype.Int4{Valid: false},
		UpdatedBy:          pgtype.Int4{Valid: false},
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrUsernameAlreadyExists
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	// 6. Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func (u *AuthUseCase) ChangePassword(ctx context.Context, userID int32, req model.ChangePasswordRequest) (*model.LoginResponse, error) {
	if req.NewPassword != req.ConfirmNewPassword {
		return nil, ErrPasswordConfirmationMismatch
	}

	userDetail, err := u.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("%w: get user detail", ErrAuthServiceUnavailable)
	}

	rawUser, err := u.userRepo.GetUserByUsername(ctx, userDetail.Username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("%w: get user by username", ErrAuthServiceUnavailable)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(rawUser.Password), []byte(req.CurrentPassword)); err != nil {
		return nil, ErrCurrentPasswordInvalid
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("%w: hash new password", ErrAuthServiceUnavailable)
	}

	updatedBy := nullableInt32Param(&userID)
	affected, err := u.userRepo.UpdateUserPasswordForChange(ctx, entity.UpdateUserPasswordForChangeParams{
		IDUser:    userID,
		Password:  string(hashedPassword),
		UpdatedBy: updatedBy,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: update password", ErrAuthServiceUnavailable)
	}
	if affected == 0 {
		return nil, ErrInvalidCredentials
	}

	var idMitra *int32
	if userDetail.IDMitra.Valid {
		val := userDetail.IDMitra.Int32
		idMitra = &val
	}

	return u.issueLoginToken(ctx, userID, userDetail.IDRole, userDetail.NamaRole, idMitra, false)
}

func (u *AuthUseCase) CreateForgotPasswordRequest(ctx context.Context, req model.ForgotPasswordRequestCreateRequest) (*model.PasswordResetRequestResponse, error) {
	user, err := u.userRepo.GetUserByUsername(ctx, req.Username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("%w: get user by username", ErrAuthServiceUnavailable)
	}

	if user.Status != "active" {
		return nil, ErrUserValidation
	}

	hasPending, err := u.userRepo.HasPendingPasswordResetRequest(ctx, user.IDUser)
	if err != nil {
		return nil, fmt.Errorf("%w: check pending password reset request", ErrAuthServiceUnavailable)
	}
	if hasPending {
		return nil, ErrPasswordResetRequestAlreadyOpen
	}

	row, err := u.userRepo.CreatePasswordResetRequest(ctx, entity.CreatePasswordResetRequestParams{
		IDUser: user.IDUser,
		Reason: req.Reason,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: create password reset request", ErrAuthServiceUnavailable)
	}

	return &model.PasswordResetRequestResponse{
		IDPasswordResetRequest: row.IDPasswordResetRequest,
		IDUser:                 row.IDUser,
		Username:               user.Username,
		IDRole:                 user.IDRole,
		NamaRole:               user.NamaRole,
		Reason:                 row.Reason,
		Status:                 row.Status,
		RequestedAt:            nullableTimestampString(row.RequestedAt),
		ApprovedAt:             nullableTimestampString(row.ApprovedAt),
		RejectedAt:             nullableTimestampString(row.RejectedAt),
		CompletedAt:            nullableTimestampString(row.CompletedAt),
		RejectedReason:         row.RejectedReason,
	}, nil
}

func (u *AuthUseCase) ListForgotPasswordRequests(ctx context.Context) ([]model.PasswordResetRequestResponse, error) {
	rows, err := u.userRepo.ListPasswordResetRequests(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: list password reset requests", ErrAuthServiceUnavailable)
	}

	items := make([]model.PasswordResetRequestResponse, 0, len(rows))
	for _, row := range rows {
		item := model.PasswordResetRequestResponse{
			IDPasswordResetRequest: row.IDPasswordResetRequest,
			IDUser:                 row.IDUser,
			Username:               row.Username,
			IDRole:                 row.IDRole,
			NamaRole:               row.NamaRole,
			Reason:                 row.Reason,
			Status:                 row.Status,
			RequestedAt:            nullableTimestampString(row.RequestedAt),
			ApprovedAt:             nullableTimestampString(row.ApprovedAt),
			RejectedAt:             nullableTimestampString(row.RejectedAt),
			CompletedAt:            nullableTimestampString(row.CompletedAt),
			RejectedReason:         row.RejectedReason,
			ApprovedByUsername:     row.ApprovedByUsername.String,
			RejectedByUsername:     row.RejectedByUsername.String,
		}
		if row.ApprovedBy.Valid {
			val := row.ApprovedBy.Int32
			item.ApprovedBy = &val
		}
		if row.RejectedBy.Valid {
			val := row.RejectedBy.Int32
			item.RejectedBy = &val
		}
		items = append(items, item)
	}

	return items, nil
}

func (u *AuthUseCase) ApproveForgotPasswordRequest(ctx context.Context, requestID int32, operatorID int32) (*model.ApprovePasswordResetResponse, error) {
	beforeRequest, err := u.getPasswordResetRequestForAudit(ctx, requestID)
	if err != nil {
		return nil, err
	}

	tx, err := u.dbPool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: begin password reset approval transaction", ErrAuthServiceUnavailable)
	}
	defer func() {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			err = rollbackErr
		}
	}()

	qtx := entity.New(tx)
	temporaryPassword, err := passwordutil.GenerateTemporaryPassword(12)
	if err != nil {
		return nil, fmt.Errorf("%w: generate temporary password", ErrAuthServiceUnavailable)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(temporaryPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("%w: hash temporary password", ErrAuthServiceUnavailable)
	}

	processedBy := nullableInt32Param(&operatorID)
	requestRow, err := qtx.ApprovePasswordResetRequest(ctx, entity.ApprovePasswordResetRequestParams{
		IDPasswordResetRequest: requestID,
		ApprovedBy:             processedBy,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPasswordResetRequestNotFound
		}
		return nil, fmt.Errorf("%w: approve password reset request", ErrAuthServiceUnavailable)
	}

	affected, err := qtx.ResetUserPasswordTemporary(ctx, entity.ResetUserPasswordTemporaryParams{
		IDUser:    requestRow.IDUser,
		Password:  string(hashedPassword),
		UpdatedBy: processedBy,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: set temporary password", ErrAuthServiceUnavailable)
	}
	if affected == 0 {
		return nil, ErrUserNotFound
	}

	user, err := qtx.GetUserByID(ctx, requestRow.IDUser)
	if err != nil {
		return nil, fmt.Errorf("%w: get updated user after password reset approval", ErrAuthServiceUnavailable)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("%w: commit password reset approval", ErrAuthServiceUnavailable)
	}

	response := &model.ApprovePasswordResetResponse{
		PasswordResetRequestResponse: model.PasswordResetRequestResponse{
			IDPasswordResetRequest: requestRow.IDPasswordResetRequest,
			IDUser:                 requestRow.IDUser,
			Username:               user.Username,
			IDRole:                 user.IDRole,
			NamaRole:               user.NamaRole,
			Reason:                 requestRow.Reason,
			Status:                 requestRow.Status,
			RequestedAt:            nullableTimestampString(requestRow.RequestedAt),
			ApprovedAt:             nullableTimestampString(requestRow.ApprovedAt),
			RejectedAt:             nullableTimestampString(requestRow.RejectedAt),
			CompletedAt:            nullableTimestampString(requestRow.CompletedAt),
			RejectedReason:         requestRow.RejectedReason,
			ApprovedBy:             &operatorID,
		},
		TemporaryPassword: temporaryPassword,
	}

	u.recordPasswordResetAudit(
		ctx,
		response.PasswordResetRequestResponse,
		buildPasswordResetRequestAuditSnapshot(beforeRequest),
		buildPasswordResetRequestAuditSnapshot(&response.PasswordResetRequestResponse),
		"password reset approval",
	)

	return response, nil
}

func (u *AuthUseCase) RejectForgotPasswordRequest(ctx context.Context, requestID int32, operatorID int32, rejectedReason string) (*model.PasswordResetRequestResponse, error) {
	beforeRequest, err := u.getPasswordResetRequestForAudit(ctx, requestID)
	if err != nil {
		return nil, err
	}

	row, err := u.userRepo.RejectPasswordResetRequest(ctx, entity.RejectPasswordResetRequestParams{
		IDPasswordResetRequest: requestID,
		RejectedBy:             nullableInt32Param(&operatorID),
		RejectedReason:         rejectedReason,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPasswordResetRequestNotFound
		}
		return nil, fmt.Errorf("%w: reject password reset request", ErrAuthServiceUnavailable)
	}

	user, err := u.userRepo.GetUserByID(ctx, row.IDUser)
	if err != nil {
		return nil, fmt.Errorf("%w: get user after password reset rejection", ErrAuthServiceUnavailable)
	}

	response := &model.PasswordResetRequestResponse{
		IDPasswordResetRequest: row.IDPasswordResetRequest,
		IDUser:                 row.IDUser,
		Username:               user.Username,
		IDRole:                 user.IDRole,
		NamaRole:               user.NamaRole,
		Reason:                 row.Reason,
		Status:                 row.Status,
		RequestedAt:            nullableTimestampString(row.RequestedAt),
		ApprovedAt:             nullableTimestampString(row.ApprovedAt),
		RejectedAt:             nullableTimestampString(row.RejectedAt),
		CompletedAt:            nullableTimestampString(row.CompletedAt),
		RejectedReason:         row.RejectedReason,
		RejectedBy:             &operatorID,
	}

	u.recordPasswordResetAudit(
		ctx,
		*response,
		buildPasswordResetRequestAuditSnapshot(beforeRequest),
		buildPasswordResetRequestAuditSnapshot(response),
		"password reset rejection",
	)

	return response, nil
}

func (u *AuthUseCase) getPasswordResetRequestForAudit(ctx context.Context, requestID int32) (*model.PasswordResetRequestResponse, error) {
	row, err := u.userRepo.GetPasswordResetRequestByID(ctx, requestID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPasswordResetRequestNotFound
		}
		return nil, fmt.Errorf("%w: get password reset request detail", ErrAuthServiceUnavailable)
	}

	item := model.PasswordResetRequestResponse{
		IDPasswordResetRequest: row.IDPasswordResetRequest,
		IDUser:                 row.IDUser,
		Username:               row.Username,
		IDRole:                 row.IDRole,
		NamaRole:               row.NamaRole,
		Reason:                 row.Reason,
		Status:                 row.Status,
		RequestedAt:            nullableTimestampString(row.RequestedAt),
		ApprovedAt:             nullableTimestampString(row.ApprovedAt),
		RejectedAt:             nullableTimestampString(row.RejectedAt),
		CompletedAt:            nullableTimestampString(row.CompletedAt),
		RejectedReason:         row.RejectedReason,
		ApprovedByUsername:     row.ApprovedByUsername.String,
		RejectedByUsername:     row.RejectedByUsername.String,
	}
	if row.ApprovedBy.Valid {
		val := row.ApprovedBy.Int32
		item.ApprovedBy = &val
	}
	if row.RejectedBy.Valid {
		val := row.RejectedBy.Int32
		item.RejectedBy = &val
	}

	return &item, nil
}

func (u *AuthUseCase) recordPasswordResetAudit(ctx context.Context, request model.PasswordResetRequestResponse, beforeSnapshot, afterSnapshot map[string]any, flow string) {
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
		Action:        "UPDATE",
		Module:        "user-management",
		EntityType:    "password_reset_requests",
		EntityID:      fmt.Sprintf("%d", request.IDPasswordResetRequest),
		EntityLabel:   request.Username,
		Method:        auditCtx.Method,
		Route:         auditCtx.Route,
		BeforeData:    beforeSnapshot,
		AfterData:     afterSnapshot,
		ChangedFields: buildChangedFieldsFromSnapshots(beforeSnapshot, afterSnapshot),
	}); err != nil {
		slog.Error("failed to record password reset audit log", slog.String("flow", flow), slog.String("error", err.Error()))
	}
}

func buildPasswordResetRequestAuditSnapshot(request *model.PasswordResetRequestResponse) map[string]any {
	if request == nil {
		return nil
	}

	return map[string]any{
		"id_password_reset_request": request.IDPasswordResetRequest,
		"id_user":                   request.IDUser,
		"username":                  request.Username,
		"id_role":                   request.IDRole,
		"nama_role":                 request.NamaRole,
		"reason":                    request.Reason,
		"status":                    request.Status,
		"requested_at":              request.RequestedAt,
		"approved_at":               request.ApprovedAt,
		"rejected_at":               request.RejectedAt,
		"completed_at":              request.CompletedAt,
		"rejected_reason":           request.RejectedReason,
		"approved_by":               request.ApprovedBy,
		"approved_by_username":      request.ApprovedByUsername,
		"rejected_by":               request.RejectedBy,
		"rejected_by_username":      request.RejectedByUsername,
	}
}

func (u *AuthUseCase) issueLoginToken(ctx context.Context, userID, roleID int32, roleName string, idMitra *int32, mustChangePassword bool) (*model.LoginResponse, error) {
	permissions, err := u.userRepo.GetUserPermissions(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("%w: get user permissions", ErrAuthServiceUnavailable)
	}

	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &jwt.MapClaims{
		"user_id":              userID,
		"id_role":              roleID,
		"role_name":            roleName,
		"permissions":          permissions,
		"id_mitra":             idMitra,
		"must_change_password": mustChangePassword,
		"exp":                  expirationTime.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(u.jwtSecret))
	if err != nil {
		return nil, err
	}

	return &model.LoginResponse{
		AccessToken:        tokenString,
		TokenType:          "Bearer",
		ExpiresIn:          int64(time.Until(expirationTime).Seconds()),
		IDRole:             roleID,
		RoleName:           roleName,
		MustChangePassword: mustChangePassword,
	}, nil
}
