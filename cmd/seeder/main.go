package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"permatatex-inventory/internal/config"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", slog.String("error", err.Error()))
		os.Exit(1)
	}

	dbPool, err := config.NewPGXPool(cfg)
	if err != nil {
		slog.Error("failed to initialize database pool", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer dbPool.Close()

	ctx := context.Background()

	// Seed User
	err = seedUser(ctx, dbPool)
	if err != nil {
		slog.Error("failed to seed user", slog.String("error", err.Error()))
		os.Exit(1)
	}

	slog.Info("seeding completed successfully")
}

func seedUser(ctx context.Context, db *pgxpool.Pool) error {
	username := "super-admin"
	password := "admin123"

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	// Check if user already exists
	var exists bool
	err = db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM "USER" WHERE username = $1)`, username).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check existing user: %w", err)
	}

	if exists {
		slog.Info("user already exists, skipping", slog.String("username", username))
		return nil
	}

	var nextID int32
	err = db.QueryRow(ctx, `SELECT COALESCE(MAX(ID_USER), 0) + 1 FROM "USER"`).Scan(&nextID)
	if err != nil {
		return fmt.Errorf("generate next user id: %w", err)
	}

	_, err = db.Exec(ctx, `
		INSERT INTO "USER" (ID_USER, USERNAME, PASSWORD, KARYAWAN)
		VALUES ($1, $2, $3, $4)
	`, nextID, username, string(hashedPassword), true)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}

	slog.Info("user seeded", slog.String("username", username), slog.String("password", password))
	return nil
}
