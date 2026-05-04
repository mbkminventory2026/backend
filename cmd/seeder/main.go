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

	// 1. Seed Hak Akses (Permissions)
	err = seedHakAkses(ctx, dbPool)
	if err != nil {
		slog.Error("failed to seed hak akses", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// 2. Seed Departemen
	err = seedDepartemen(ctx, dbPool)
	if err != nil {
		slog.Error("failed to seed departemen", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// 3. Seed Super Admin User
	err = seedUser(ctx, dbPool)
	if err != nil {
		slog.Error("failed to seed user", slog.String("error", err.Error()))
		os.Exit(1)
	}

	slog.Info("seeding completed successfully")
}

func seedHakAkses(ctx context.Context, db *pgxpool.Pool) error {
	permissions := []string{
		"USER_READ", "USER_CREATE", "USER_UPDATE", "USER_DELETE",
		"ITEM_READ", "ITEM_CREATE", "ITEM_UPDATE", "ITEM_DELETE",
		"PO_READ", "PO_CREATE", "REPORT_READ",
	}

	for _, p := range permissions {
		var exists bool
		err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM HAK_AKSES WHERE NAMA_HALAMAN = $1)`, p).Scan(&exists)
		if err != nil {
			return err
		}

		if !exists {
			_, err = db.Exec(ctx, `INSERT INTO HAK_AKSES (NAMA_HALAMAN) VALUES ($1)`, p)
			if err != nil {
				return err
			}
			slog.Info("permission seeded", slog.String("name", p))
		}
	}
	return nil
}

func seedDepartemen(ctx context.Context, db *pgxpool.Pool) error {
	depts := []string{"IT", "PRODUKSI", "GUDANG", "OFFICE"}
	for _, d := range depts {
		var exists bool
		err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM DEPARTEMEN WHERE NAMA_DEPARTEMEN = $1)`, d).Scan(&exists)
		if err != nil { return err }
		if !exists {
			_, err = db.Exec(ctx, `INSERT INTO DEPARTEMEN (NAMA_DEPARTEMEN) VALUES ($1)`, d)
			if err != nil { return err }
			slog.Info("departemen seeded", slog.String("name", d))
		}
	}
	return nil
}

func seedUser(ctx context.Context, db *pgxpool.Pool) error {
	username := "super-admin"
	password := "admin123"

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	var exists bool
	err = db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM USERS WHERE username = $1)`, username).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check existing user: %w", err)
	}

	if exists {
		slog.Info("user already exists, skipping", slog.String("username", username))
		return nil
	}

	_, err = db.Exec(ctx, `
		INSERT INTO USERS (USERNAME, PASSWORD, IS_MANAGER, ID_DEPARTEMEN)
		VALUES ($1, $2, $3, (SELECT ID_DEPARTEMEN FROM DEPARTEMEN WHERE NAMA_DEPARTEMEN = 'IT' LIMIT 1))
	`, username, string(hashedPassword), true)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}

	slog.Info("user seeded", slog.String("username", username), slog.String("password", password))
	return nil
}
