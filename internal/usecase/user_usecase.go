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
	"golang.org/x/crypto/bcrypt"

	"permatatex-inventory/internal/entity"
	"permatatex-inventory/internal/model"
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
	repo   entity.Querier
	dbPool *pgxpool.Pool
}

func NewUserUseCase(repo entity.Querier, dbPool *pgxpool.Pool) (*UserUseCase, error) {
	if repo == nil {
		return nil, errors.New("user repository is required")
	}
	if dbPool == nil {
		return nil, errors.New("database pool is required")
	}

	return &UserUseCase{
		repo:   repo,
		dbPool: dbPool,
	}, nil
}

func (u *UserUseCase) Create(ctx context.Context, req model.CreateUserRequest) (*model.UserResponse, error) {
	// 1. Hash Password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
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

	user, err := qtx.CreateUser(ctx, entity.CreateUserParams{
		Username:     req.Username,
		Password:     string(hashedPassword),
		IDRole:       req.IDRole,
		IDDepartemen: idDept,
		IDMitra:      idMitra,
		Status:       status,
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

	var resDept *int32
	if user.IDDepartemen.Valid {
		val := user.IDDepartemen.Int32
		resDept = &val
	}

	var resMitra *int32
	if user.IDMitra.Valid {
		val := user.IDMitra.Int32
		resMitra = &val
	}

	return &model.UserResponse{
		IDUser:       user.IDUser,
		Username:     user.Username,
		Status:       user.Status,
		IDRole:       user.IDRole,
		NamaRole:     role.NamaRole,
		IDDepartemen: resDept,
		IDMitra:      resMitra,
		CreatedAt:    user.CreatedAt.Time.Format(time.RFC3339),
		HakAksesIDs:  req.HakAksesIDs,
	}, nil
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
			IDUser:    item.IDUser,
			Username:  item.Username,
			Status:    item.Status,
			IDRole:    item.IDRole,
			NamaRole:  item.NamaRole,
			CreatedAt: item.CreatedAt.Time.Format(time.RFC3339),
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
		IDUser:      user.IDUser,
		Username:    user.Username,
		Status:      user.Status,
		IDRole:      user.IDRole,
		NamaRole:    user.NamaRole,
		CreatedAt:   user.CreatedAt.Time.Format(time.RFC3339),
		Permissions: permissions,
		HakAksesIDs: permissionIDs,
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

func (u *UserUseCase) Update(ctx context.Context, id int32, req model.UpdateUserRequest) (*model.UserResponse, error) {
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

	// 3. Update password if provided
	if req.Password != nil && *req.Password != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		finalPassword = string(hashed)
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
		IDUser:       id,
		Username:     req.Username,
		Password:     finalPassword,
		IDRole:       userForUpdate.IDRole, // Use current role, ignore changes
		IDDepartemen: idDept,
		IDMitra:      idMitra,
		Status:       status,
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

	return &model.UserResponse{
		IDUser:       updatedUser.IDUser,
		Username:     updatedUser.Username,
		Status:       updatedUser.Status,
		IDRole:       updatedUser.IDRole,
		NamaRole:     role.NamaRole,
		IDDepartemen: resDept,
		IDMitra:      resMitra,
		CreatedAt:    updatedUser.CreatedAt.Time.Format(time.RFC3339),
		HakAksesIDs:  permissionIDs,
	}, nil
}

func (u *UserUseCase) Delete(ctx context.Context, idUser int32) error {
	// Critical Validation: Protect Super Admin
	if idUser == 1 {
		return ErrCannotDeleteSuperAdmin
	}

	affected, err := u.repo.DeleteUser(ctx, idUser)
	if err != nil {
		return fmt.Errorf("%w: failed to delete user", ErrUserServiceUnavailable)
	}
	if affected == 0 {
		return ErrUserNotFound
	}

	return nil
}

func (u *UserUseCase) AssignRole(ctx context.Context, idUser int32, idRole int32) (*model.UserResponse, error) {
	_, err := u.repo.UpdateUserRole(ctx, entity.UpdateUserRoleParams{
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

	return u.GetByID(ctx, idUser)
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

func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

func (u *UserUseCase) Approve(ctx context.Context, id int32) (*model.UserResponse, error) {
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
	updatedUser, err := qtx.UpdateUserStatus(ctx, entity.UpdateUserStatusParams{
		IDUser: id,
		Status: "active",
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &model.UserResponse{
		IDUser:   updatedUser.IDUser,
		Username: updatedUser.Username,
		Status:   updatedUser.Status,
	}, nil
}

func (u *UserUseCase) Reject(ctx context.Context, id int32) error {
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

	return nil
}
