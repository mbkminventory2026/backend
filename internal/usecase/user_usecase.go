package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"permatatex-inventory/internal/entity"
	"permatatex-inventory/internal/model"
	"permatatex-inventory/pkg/passwordutil"
)

var (
	ErrUserValidation         = errors.New("invalid user payload")
	ErrUserNotFound           = errors.New("user not found")
	ErrUserServiceUnavailable = errors.New("user service unavailable")
	ErrCannotDeleteSuperAdmin = errors.New("Super Admin cannot be deleted")
	ErrUsernameAlreadyExists  = errors.New("username already exists")

	userSortColumns = buildSortWhitelist("created_at", "id_user", "username", "status", "id_role", "nama_role")
)

type UserUseCase struct {
	repo     entity.Querier
	dbPool   *pgxpool.Pool
	auditLog *AuditLogUseCase
}

func NewUserUseCase(repo entity.Querier, dbPool *pgxpool.Pool, auditLog *AuditLogUseCase) (*UserUseCase, error) {
	if repo == nil {
		return nil, errors.New("user repository is required")
	}
	if dbPool == nil {
		return nil, errors.New("database pool is required")
	}

	return &UserUseCase{
		repo:     repo,
		dbPool:   dbPool,
		auditLog: auditLog,
	}, nil
}

func (u *UserUseCase) Create(ctx context.Context, actorUserID *int32, req model.CreateUserRequest) (*model.UserResponse, error) {
	temporaryPassword := ""
	if req.Password != nil && *req.Password != "" {
		temporaryPassword = *req.Password
	} else {
		var err error
		temporaryPassword, err = passwordutil.GenerateTemporaryPassword(12)
		if err != nil {
			return nil, fmt.Errorf("%w: failed to generate temporary password", ErrUserServiceUnavailable)
		}
	}

	// 1. Hash Password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(temporaryPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to hash password", ErrUserServiceUnavailable)
	}

	// 2. Start Transaction
	tx, err := u.dbPool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to start transaction", ErrUserServiceUnavailable)
	}
	defer func() {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			err = rollbackErr
		}
	}()

	qtx := entity.New(tx)

	// 3. Create User
	idDept := pgtype.Int4{Valid: false}
	if req.IDDepartemen != nil {
		idDept = pgtype.Int4{Int32: *req.IDDepartemen, Valid: true}
	}

	idMitra := pgtype.Int4{Valid: false}
	if req.IDMitra != nil {
		idMitra = pgtype.Int4{Int32: *req.IDMitra, Valid: true}
	}

	status := "active"
	if req.Status != nil && *req.Status != "" {
		status = *req.Status
	}

	actorParam := nullableInt32Param(actorUserID)
	user, err := qtx.CreateUser(ctx, entity.CreateUserParams{
		Username:           req.Username,
		Password:           string(hashedPassword),
		IDRole:             req.IDRole,
		IDDepartemen:       idDept,
		IDMitra:            idMitra,
		Status:             status,
		MustChangePassword: true,
		CreatedBy:          actorParam,
		UpdatedBy:          actorParam,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrUsernameAlreadyExists
		}
		if isForeignKeyViolation(err) {
			return nil, ErrUserValidation
		}
		return nil, fmt.Errorf("%w: failed to create user", ErrUserServiceUnavailable)
	}

	role, err := qtx.GetRoleByID(ctx, user.IDRole)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserValidation
		}
		return nil, fmt.Errorf("%w: failed to get role", ErrUserServiceUnavailable)
	}

	// 4. Create User Access (Permissions)
	for _, hakAksesID := range req.HakAksesIDs {
		err := qtx.CreateUserAkses(ctx, entity.CreateUserAksesParams{
			IDUser:     user.IDUser,
			IDHakAkses: hakAksesID,
		})
		if err != nil {
			return nil, fmt.Errorf("%w: failed to assign permissions", ErrUserServiceUnavailable)
		}
	}

	// 5. Commit
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("%w: failed to commit transaction", ErrUserServiceUnavailable)
	}

	result, err := u.GetByID(ctx, user.IDUser)
	if err != nil {
		return nil, err
	}
	result.TemporaryPassword = temporaryPassword
	if result.NamaRole == "" {
		result.NamaRole = role.NamaRole
	}
	result.HakAksesIDs = req.HakAksesIDs

	u.recordCreateUserAudit(ctx, result)

	return result, nil
}

func (u *UserUseCase) List(ctx context.Context, filter model.ListUsersFilter) ([]model.UserResponse, int64, error) {
	_, limit, offset, search, sortBy, sortDesc := normalizeListFilter(filter.ListQueryFilter, "id_user", false, userSortColumns)
	items, err := u.repo.ListUsers(ctx, entity.ListUsersParams{
		SearchTerm: search,
		SortBy:     sortBy,
		SortDesc:   sortDesc,
		PageLimit:  limit,
		PageOffset: offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("%w: failed to list users", ErrUserServiceUnavailable)
	}

	total, err := u.repo.CountUsers(ctx, search)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: failed to count users", ErrUserServiceUnavailable)
	}

	result := make([]model.UserResponse, 0, len(items))
	for _, item := range items {
		res := model.UserResponse{
			IDUser:             item.IDUser,
			Username:           item.Username,
			Status:             item.Status,
			IDRole:             item.IDRole,
			NamaRole:           item.NamaRole,
			MustChangePassword: item.MustChangePassword,
			CreatedAt:          item.CreatedAt.Time.Format(time.RFC3339),
		}
		if item.IDDepartemen.Valid {
			val := item.IDDepartemen.Int32
			res.IDDepartemen = &val
		}
		if item.IDMitra.Valid {
			val := item.IDMitra.Int32
			res.IDMitra = &val
		}
		if item.NamaDepartemen.Valid {
			res.NamaDepartemen = item.NamaDepartemen.String
		}
		if item.NamaPerusahaan.Valid {
			res.NamaPerusahaan = item.NamaPerusahaan.String
		}
		result = append(result, res)
	}

	return result, total, nil
}

func (u *UserUseCase) GetByID(ctx context.Context, id int32) (*model.UserResponse, error) {
	user, err := u.repo.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("%w: failed to get user", ErrUserServiceUnavailable)
	}

	permissions, err := u.repo.GetUserPermissions(ctx, user.IDUser)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get user permissions", ErrUserServiceUnavailable)
	}

	permissionIDs, err := u.repo.GetUserPermissionIDs(ctx, user.IDUser)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get user permission IDs", ErrUserServiceUnavailable)
	}

	res := &model.UserResponse{
		IDUser:             user.IDUser,
		Username:           user.Username,
		Status:             user.Status,
		IDRole:             user.IDRole,
		NamaRole:           user.NamaRole,
		MustChangePassword: user.MustChangePassword,
		CreatedAt:          user.CreatedAt.Time.Format(time.RFC3339),
		PasswordChangedAt:  nullableTimestampString(user.PasswordChangedAt),
		Permissions:        permissions,
		HakAksesIDs:        permissionIDs,
	}
	if user.IDDepartemen.Valid {
		val := user.IDDepartemen.Int32
		res.IDDepartemen = &val
	}
	if user.IDMitra.Valid {
		val := user.IDMitra.Int32
		res.IDMitra = &val
	}
	if user.NamaDepartemen.Valid {
		res.NamaDepartemen = user.NamaDepartemen.String
	}
	if user.NamaPerusahaan.Valid {
		res.NamaPerusahaan = user.NamaPerusahaan.String
	}

	return res, nil
}

func (u *UserUseCase) Update(ctx context.Context, id int32, actorUserID *int32, req model.UpdateUserRequest) (*model.UserResponse, error) {
	// 1. Fetch current user to check existence and get existing password
	userForUpdate, err := u.repo.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	// 2. Fetch raw user to get the current hashed password
	rawUser, err := u.repo.GetUserByUsername(ctx, userForUpdate.Username)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrUsernameAlreadyExists
		}
		return nil, err
	}
	finalPassword := rawUser.Password
	mustChangePassword := userForUpdate.MustChangePassword
	passwordChangedAt := userForUpdate.PasswordChangedAt

	// 3. Update password if provided
	if req.Password != nil && *req.Password != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		finalPassword = string(hashed)
		mustChangePassword = true
		passwordChangedAt = pgtype.Timestamptz{Valid: false}
	}

	// 4. Start Transaction
	tx, err := u.dbPool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			err = rollbackErr
		}
	}()
	qtx := entity.New(tx)

	// 5. Update User record
	idDept := pgtype.Int4{Valid: false}
	if req.IDDepartemen != nil {
		idDept = pgtype.Int4{Int32: *req.IDDepartemen, Valid: true}
	}

	idMitra := pgtype.Int4{Valid: false}
	if req.IDMitra != nil {
		idMitra = pgtype.Int4{Int32: *req.IDMitra, Valid: true}
	}

	status := userForUpdate.Status
	if req.Status != nil && *req.Status != "" {
		status = *req.Status
	}

	updatedUser, err := qtx.UpdateUser(ctx, entity.UpdateUserParams{
		IDUser:             id,
		Username:           req.Username,
		Password:           finalPassword,
		IDRole:             userForUpdate.IDRole, // Use current role, ignore changes
		IDDepartemen:       idDept,
		IDMitra:            idMitra,
		Status:             status,
		MustChangePassword: mustChangePassword,
		PasswordChangedAt:  passwordChangedAt,
		UpdatedBy:          nullableInt32Param(actorUserID),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrUsernameAlreadyExists
		}
		if isForeignKeyViolation(err) {
			return nil, ErrUserValidation
		}
		return nil, err
	}

	role, err := qtx.GetRoleByID(ctx, updatedUser.IDRole)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserValidation
		}
		return nil, err
	}

	// 6. Get Existing Override Permissions
	permissionIDs, err := qtx.GetUserPermissionIDs(ctx, id)
	if err != nil {
		return nil, err
	}

	// 7. Commit Transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	result, err := u.GetByID(ctx, updatedUser.IDUser)
	if err != nil {
		return nil, err
	}
	if result.NamaRole == "" {
		result.NamaRole = role.NamaRole
	}
	if result.PasswordChangedAt == "" {
		result.PasswordChangedAt = nullableTimestampString(passwordChangedAt)
	}
	if len(result.HakAksesIDs) == 0 {
		result.HakAksesIDs = permissionIDs
	}

	beforeSnapshot := buildUserAuditSnapshot(&model.UserResponse{
		IDUser:             userForUpdate.IDUser,
		Username:           userForUpdate.Username,
		Status:             userForUpdate.Status,
		IDRole:             userForUpdate.IDRole,
		NamaRole:           userForUpdate.NamaRole,
		MustChangePassword: userForUpdate.MustChangePassword,
		IDDepartemen:       nullableInt32Pointer(userForUpdate.IDDepartemen),
		IDMitra:            nullableInt32Pointer(userForUpdate.IDMitra),
		NamaDepartemen:     nullableTextString(userForUpdate.NamaDepartemen),
		NamaPerusahaan:     nullableTextString(userForUpdate.NamaPerusahaan),
		CreatedAt:          userForUpdate.CreatedAt.Time.Format(time.RFC3339),
		PasswordChangedAt:  nullableTimestampString(userForUpdate.PasswordChangedAt),
		HakAksesIDs:        permissionIDs,
	})
	afterSnapshot := buildUserAuditSnapshot(result)
	u.recordUpdateUserAudit(ctx, result, beforeSnapshot, afterSnapshot)

	return result, nil
}

func (u *UserUseCase) Delete(ctx context.Context, idUser int32) error {
	// Critical Validation: Protect Super Admin
	if idUser == 1 {
		return ErrCannotDeleteSuperAdmin
	}

	existing, err := u.GetByID(ctx, idUser)
	if err != nil {
		return err
	}

	affected, err := u.repo.DeleteUser(ctx, idUser)
	if err != nil {
		return fmt.Errorf("%w: failed to delete user", ErrUserServiceUnavailable)
	}
	if affected == 0 {
		return ErrUserNotFound
	}

	u.recordDeleteUserAudit(ctx, existing)

	return nil
}

func (u *UserUseCase) AssignRole(ctx context.Context, idUser int32, idRole int32) (*model.UserResponse, error) {
	beforeUser, err := u.GetByID(ctx, idUser)
	if err != nil {
		return nil, err
	}

	_, err = u.repo.UpdateUserRole(ctx, entity.UpdateUserRoleParams{
		IDUser: idUser,
		IDRole: idRole,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		if isForeignKeyViolation(err) {
			return nil, ErrUserValidation
		}
		return nil, fmt.Errorf("%w: failed to assign role", ErrUserServiceUnavailable)
	}

	result, err := u.GetByID(ctx, idUser)
	if err != nil {
		return nil, err
	}

	u.recordAssignRoleAudit(ctx, result, buildUserAuditSnapshot(beforeUser), buildUserAuditSnapshot(result))

	return result, nil
}

func (u *UserUseCase) ReplacePermissions(ctx context.Context, idUser int32, hakAksesIDs []int32) (*model.UserResponse, error) {
	_, err := u.repo.GetUserByID(ctx, idUser)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("%w: failed to get user", ErrUserServiceUnavailable)
	}

	tx, err := u.dbPool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to start transaction", ErrUserServiceUnavailable)
	}
	defer func() {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			err = rollbackErr
		}
	}()

	qtx := entity.New(tx)
	if _, err := qtx.DeleteUserAksesByUserID(ctx, idUser); err != nil {
		return nil, fmt.Errorf("%w: failed to clear user permissions", ErrUserServiceUnavailable)
	}

	for _, hakAksesID := range hakAksesIDs {
		err = qtx.CreateUserAkses(ctx, entity.CreateUserAksesParams{
			IDUser:     idUser,
			IDHakAkses: hakAksesID,
		})
		if err != nil {
			if isForeignKeyViolation(err) {
				return nil, ErrUserValidation
			}
			return nil, fmt.Errorf("%w: failed to assign user permissions", ErrUserServiceUnavailable)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("%w: failed to commit transaction", ErrUserServiceUnavailable)
	}

	return u.GetByID(ctx, idUser)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func (u *UserUseCase) recordCreateUserAudit(ctx context.Context, user *model.UserResponse) {
	if u.auditLog == nil || user == nil {
		return
	}

	auditCtx, ok := GetAuditLogContext(ctx)
	if !ok {
		return
	}

	if err := u.auditLog.Record(ctx, model.AuditLogRecordRequest{
		ActorUserID: auditCtx.ActorUserID,
		ActorRole:   auditCtx.ActorRole,
		Action:      "CREATE",
		Module:      "user-management",
		EntityType:  "users",
		EntityID:    fmt.Sprintf("%d", user.IDUser),
		EntityLabel: user.Username,
		Method:      auditCtx.Method,
		Route:       auditCtx.Route,
		AfterData:   buildUserAuditSnapshot(user),
	}); err != nil {
		slog.Error("failed to record user create audit log", slog.String("error", err.Error()))
	}
}

func (u *UserUseCase) recordUpdateUserAudit(ctx context.Context, user *model.UserResponse, beforeSnapshot, afterSnapshot map[string]any) {
	if u.auditLog == nil || user == nil {
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
		EntityType:    "users",
		EntityID:      fmt.Sprintf("%d", user.IDUser),
		EntityLabel:   user.Username,
		Method:        auditCtx.Method,
		Route:         auditCtx.Route,
		BeforeData:    beforeSnapshot,
		AfterData:     afterSnapshot,
		ChangedFields: buildChangedFieldsFromSnapshots(beforeSnapshot, afterSnapshot),
	}); err != nil {
		slog.Error("failed to record user update audit log", slog.String("error", err.Error()))
	}
}

func (u *UserUseCase) recordDeleteUserAudit(ctx context.Context, user *model.UserResponse) {
	if u.auditLog == nil || user == nil {
		return
	}

	auditCtx, ok := GetAuditLogContext(ctx)
	if !ok {
		return
	}

	if err := u.auditLog.Record(ctx, model.AuditLogRecordRequest{
		ActorUserID: auditCtx.ActorUserID,
		ActorRole:   auditCtx.ActorRole,
		Action:      "DELETE",
		Module:      "user-management",
		EntityType:  "users",
		EntityID:    fmt.Sprintf("%d", user.IDUser),
		EntityLabel: user.Username,
		Method:      auditCtx.Method,
		Route:       auditCtx.Route,
		BeforeData:  buildUserAuditSnapshot(user),
	}); err != nil {
		slog.Error("failed to record user delete audit log", slog.String("error", err.Error()))
	}
}

func buildUserAuditSnapshot(user *model.UserResponse) map[string]any {
	if user == nil {
		return nil
	}

	snapshot := map[string]any{
		"id_user":              user.IDUser,
		"username":             user.Username,
		"status":               user.Status,
		"id_role":              user.IDRole,
		"nama_role":            user.NamaRole,
		"must_change_password": user.MustChangePassword,
		"nama_departemen":      user.NamaDepartemen,
		"nama_perusahaan":      user.NamaPerusahaan,
		"password_changed_at":  user.PasswordChangedAt,
		"hak_akses_ids":        user.HakAksesIDs,
	}

	if user.IDDepartemen != nil {
		snapshot["id_departemen"] = *user.IDDepartemen
	} else {
		snapshot["id_departemen"] = nil
	}

	if user.IDMitra != nil {
		snapshot["id_mitra"] = *user.IDMitra
	} else {
		snapshot["id_mitra"] = nil
	}

	return snapshot
}

func nullableInt32Pointer(value pgtype.Int4) *int32 {
	if !value.Valid {
		return nil
	}

	val := value.Int32
	return &val
}

func nullableTextString(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}

	return value.String
}

func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

func (u *UserUseCase) Approve(ctx context.Context, id int32, newUsername string) (*model.UserResponse, error) {
	beforeUser, err := u.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 1. Generate random temporary password
	temporaryPassword, err := passwordutil.GenerateTemporaryPassword(12)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to generate password", ErrUserServiceUnavailable)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(temporaryPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to hash password", ErrUserServiceUnavailable)
	}

	tx, err := u.dbPool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			err = rollbackErr
		}
	}()

	qtx := entity.New(tx)

	// Check if user exists
	user, err := qtx.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	// Update user with newUsername, hashedPassword, active status, must_change_password
	updatedUser, err := qtx.UpdateUser(ctx, entity.UpdateUserParams{
		IDUser:             id,
		Username:           newUsername,
		Password:           string(hashedPassword),
		IDRole:             user.IDRole,
		IDDepartemen:       user.IDDepartemen,
		IDMitra:            user.IDMitra,
		Status:             "active",
		MustChangePassword: true,
		PasswordChangedAt:  pgtype.Timestamptz{Valid: false},
		UpdatedBy:          pgtype.Int4{Valid: false},
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrUsernameAlreadyExists
		}
		return nil, err
	}

	role, err := qtx.GetRoleByID(ctx, updatedUser.IDRole)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	var resDept *int32
	if updatedUser.IDDepartemen.Valid {
		val := updatedUser.IDDepartemen.Int32
		resDept = &val
	}

	var resMitra *int32
	if updatedUser.IDMitra.Valid {
		val := updatedUser.IDMitra.Int32
		resMitra = &val
	}

	result := &model.UserResponse{
		IDUser:             updatedUser.IDUser,
		Username:           updatedUser.Username,
		Status:             updatedUser.Status,
		IDRole:             updatedUser.IDRole,
		NamaRole:           role.NamaRole,
		MustChangePassword: updatedUser.MustChangePassword,
		IDDepartemen:       resDept,
		IDMitra:            resMitra,
		TemporaryPassword:  temporaryPassword,
		CreatedAt:          updatedUser.CreatedAt.Time.Format(time.RFC3339),
	}

	u.recordApproveUserAudit(ctx, result, buildUserAuditSnapshot(beforeUser), buildUserAuditSnapshot(result))

	return result, nil
}

func (u *UserUseCase) Reject(ctx context.Context, id int32) error {
	beforeUser, err := u.GetByID(ctx, id)
	if err != nil {
		return err
	}

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

	_, err = qtx.UpdateUserStatus(ctx, entity.UpdateUserStatusParams{
		IDUser: id,
		Status: "rejected",
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserNotFound
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	afterSnapshot := buildUserAuditSnapshot(&model.UserResponse{
		IDUser:             beforeUser.IDUser,
		Username:           beforeUser.Username,
		Status:             "rejected",
		IDRole:             beforeUser.IDRole,
		NamaRole:           beforeUser.NamaRole,
		MustChangePassword: beforeUser.MustChangePassword,
		IDDepartemen:       beforeUser.IDDepartemen,
		IDMitra:            beforeUser.IDMitra,
		NamaDepartemen:     beforeUser.NamaDepartemen,
		NamaPerusahaan:     beforeUser.NamaPerusahaan,
		CreatedAt:          beforeUser.CreatedAt,
		PasswordChangedAt:  beforeUser.PasswordChangedAt,
		HakAksesIDs:        beforeUser.HakAksesIDs,
	})
	u.recordRejectUserAudit(ctx, beforeUser, buildUserAuditSnapshot(beforeUser), afterSnapshot)

	return nil
}

func (u *UserUseCase) recordApproveUserAudit(ctx context.Context, user *model.UserResponse, beforeSnapshot, afterSnapshot map[string]any) {
	if u.auditLog == nil || user == nil {
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
		EntityType:    "users",
		EntityID:      fmt.Sprintf("%d", user.IDUser),
		EntityLabel:   user.Username,
		Method:        auditCtx.Method,
		Route:         auditCtx.Route,
		BeforeData:    beforeSnapshot,
		AfterData:     afterSnapshot,
		ChangedFields: buildChangedFieldsFromSnapshots(beforeSnapshot, afterSnapshot),
	}); err != nil {
		slog.Error("failed to record user approval audit log", slog.String("error", err.Error()))
	}
}

func (u *UserUseCase) recordRejectUserAudit(ctx context.Context, user *model.UserResponse, beforeSnapshot, afterSnapshot map[string]any) {
	if u.auditLog == nil || user == nil {
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
		EntityType:    "users",
		EntityID:      fmt.Sprintf("%d", user.IDUser),
		EntityLabel:   user.Username,
		Method:        auditCtx.Method,
		Route:         auditCtx.Route,
		BeforeData:    beforeSnapshot,
		AfterData:     afterSnapshot,
		ChangedFields: buildChangedFieldsFromSnapshots(beforeSnapshot, afterSnapshot),
	}); err != nil {
		slog.Error("failed to record user rejection audit log", slog.String("error", err.Error()))
	}
}

func (u *UserUseCase) recordAssignRoleAudit(ctx context.Context, user *model.UserResponse, beforeSnapshot, afterSnapshot map[string]any) {
	if u.auditLog == nil || user == nil {
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
		EntityType:    "users",
		EntityID:      fmt.Sprintf("%d", user.IDUser),
		EntityLabel:   user.Username,
		Method:        auditCtx.Method,
		Route:         auditCtx.Route,
		BeforeData:    beforeSnapshot,
		AfterData:     afterSnapshot,
		ChangedFields: buildChangedFieldsFromSnapshots(beforeSnapshot, afterSnapshot),
	}); err != nil {
		slog.Error("failed to record user role assignment audit log", slog.String("error", err.Error()))
	}
}
