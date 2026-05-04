package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

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
	defer tx.Rollback(ctx)

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

	user, err := qtx.CreateUser(ctx, entity.CreateUserParams{
		Username:     req.Username,
		Password:     string(hashedPassword),
		IsManager:    req.IsManager,
		IDDepartemen: idDept,
		IDMitra:      idMitra,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create user", ErrUserServiceUnavailable)
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

	return &model.UserResponse{
		IDUser:    user.IDUser,
		Username:  user.Username,
		IsManager: user.IsManager,
		CreatedAt: user.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (u *UserUseCase) List(ctx context.Context, filter model.ListUsersFilter) ([]model.UserResponse, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	items, err := u.repo.ListUsers(ctx, entity.ListUsersParams{
		Limit:  limit,
		Offset: filter.Offset,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: failed to list users", ErrUserServiceUnavailable)
	}

	result := make([]model.UserResponse, 0, len(items))
	for _, item := range items {
		res := model.UserResponse{
			IDUser:    item.IDUser,
			Username:  item.Username,
			IsManager: item.IsManager,
			CreatedAt: item.CreatedAt.Time.Format(time.RFC3339),
		}
		if item.NamaDepartemen.Valid {
			res.NamaDepartemen = item.NamaDepartemen.String
		}
		if item.NamaPerusahaan.Valid {
			res.NamaPerusahaan = item.NamaPerusahaan.String
		}
		result = append(result, res)
	}

	return result, nil
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

	res := &model.UserResponse{
		IDUser:      user.IDUser,
		Username:    user.Username,
		IsManager:   user.IsManager,
		CreatedAt:   user.CreatedAt.Time.Format(time.RFC3339),
		Permissions: permissions,
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
	defer tx.Rollback(ctx)
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

	updatedUser, err := qtx.UpdateUser(ctx, entity.UpdateUserParams{
		IDUser:       id,
		Username:     req.Username,
		Password:     finalPassword,
		IsManager:    req.IsManager,
		IDDepartemen: idDept,
		IDMitra:      idMitra,
	})
	if err != nil {
		return nil, err
	}

	// 6. Sync Permissions (Delete all current and insert new ones)
	_, err = tx.Exec(ctx, `DELETE FROM USER_AKSES WHERE id_user = $1`, id)
	if err != nil {
		return nil, err
	}

	for _, pID := range req.HakAksesIDs {
		err = qtx.CreateUserAkses(ctx, entity.CreateUserAksesParams{
			IDUser:     id,
			IDHakAkses: pID,
		})
		if err != nil {
			return nil, err
		}
	}

	// 7. Commit Transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &model.UserResponse{
		IDUser:    updatedUser.IDUser,
		Username:  updatedUser.Username,
		IsManager: updatedUser.IsManager,
		CreatedAt: updatedUser.CreatedAt.Time.Format(time.RFC3339),
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
