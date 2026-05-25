package usecase

import (
	"context"
	"errors"
	"fmt"
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
)

var (
	ErrInvalidCredentials     = errors.New("invalid username or password")
	ErrAuthServiceUnavailable = errors.New("authentication service unavailable")
	ErrAccountPending         = errors.New("akun Anda sedang menunggu persetujuan admin")
	ErrAccountRejected        = errors.New("akun Anda telah ditolak")
)

type AuthUseCase struct {
	userRepo         entity.Querier
	dbPool           *pgxpool.Pool
	turnstileUseCase *TurnstileUseCase
	jwtSecret        string
}

func NewAuthUseCase(userRepo entity.Querier, dbPool *pgxpool.Pool, turnstileUseCase *TurnstileUseCase, jwtSecret string) *AuthUseCase {
	return &AuthUseCase{
		userRepo:         userRepo,
		dbPool:           dbPool,
		turnstileUseCase: turnstileUseCase,
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

	// 4. Handle RBAC (Permissions)
	var permissions []string
	if user.IsManager {
		permissions = []string{"ALL_ACCESS"}
	} else {
		permissions, err = u.userRepo.GetUserPermissions(ctx, user.IDUser)
		if err != nil {
			return nil, fmt.Errorf("%w: get user permissions", ErrAuthServiceUnavailable)
		}
	}

	var idMitra *int32
	if user.IDMitra.Valid {
		val := user.IDMitra.Int32
		idMitra = &val
	}

	// 5. Generate JWT
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &jwt.MapClaims{
		"user_id":     user.IDUser,
		"is_manager":  user.IsManager,
		"permissions": permissions,
		"id_mitra":    idMitra,
		"exp":         expirationTime.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(u.jwtSecret))
	if err != nil {
		return nil, err
	}

	return &model.LoginResponse{
		AccessToken: tokenString,
		TokenType:   "Bearer",
		ExpiresIn:   int64(time.Until(expirationTime).Seconds()),
	}, nil
}

func (u *AuthUseCase) RegisterMitra(ctx context.Context, req model.RegisterMitraRequest, remoteIP string) error {
	// 1. Verify Turnstile Token
	_, err := u.turnstileUseCase.VerifyToken(ctx, model.VerifyTurnstileRequest{
		TurnstileToken: req.TurnstileToken,
	}, remoteIP)
	if err != nil {
		return fmt.Errorf("captcha verification: %w", err)
	}

	// 2. Hash Password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
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
	idMitra := pgtype.Int4{Int32: mitra.IDMitra, Valid: true}
	idDept := pgtype.Int4{Valid: false}

	_, err = qtx.CreateUser(ctx, entity.CreateUserParams{
		Username:     req.Username,
		Password:     string(hashedPassword),
		IsManager:    false,
		IDDepartemen: idDept,
		IDMitra:      idMitra,
		Status:       "pending",
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
