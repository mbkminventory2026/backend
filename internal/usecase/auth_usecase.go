package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"permatatex-inventory/internal/entity"
	"permatatex-inventory/internal/model"
)

var (
	ErrInvalidCredentials     = errors.New("invalid username or password")
	ErrAuthServiceUnavailable = errors.New("authentication service unavailable")
)

type AuthUseCase struct {
	userRepo         entity.Querier
	turnstileUseCase *TurnstileUseCase
	jwtSecret        string
}

func NewAuthUseCase(userRepo entity.Querier, turnstileUseCase *TurnstileUseCase, jwtSecret string) *AuthUseCase {
	return &AuthUseCase{
		userRepo:         userRepo,
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

	if strings.TrimSpace(u.jwtSecret) == "" {
		return nil, ErrAuthServiceUnavailable
	}

	if !user.Password.Valid {
		return nil, ErrInvalidCredentials
	}

	// 3. Verify Password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password.String), []byte(req.Password))
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// 4. Generate JWT
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &jwt.MapClaims{
		"user_id": user.IDUser,
		"exp":     expirationTime.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(u.jwtSecret))
	if err != nil {
		return nil, err
	}

	return &model.LoginResponse{
		AccessToken: tokenString,
		TokenType:   "Bearer",
		ExpiresIn:   int64(expirationTime.Sub(time.Now()).Seconds()),
	}, nil
}
