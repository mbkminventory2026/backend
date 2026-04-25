package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

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

	err = seedReportPengirimanDependencies(ctx, dbPool)
	if err != nil {
		slog.Error("failed to seed report pengiriman dependencies", slog.String("error", err.Error()))
		os.Exit(1)
	}

	err = seedReportPengiriman(ctx, dbPool)
	if err != nil {
		slog.Error("failed to seed report pengiriman", slog.String("error", err.Error()))
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

func seedReportPengirimanDependencies(ctx context.Context, db *pgxpool.Pool) error {
	_, err := db.Exec(ctx, `
		INSERT INTO PO_CLIENT_ITEM (ID_PO_CLIENT_ITEM, STYLE, COLOUR, "DESC", QTY, PRICE)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (ID_PO_CLIENT_ITEM) DO NOTHING
	`, 1, "SEED-STYLE", "BLACK", "seed po client item", 100, 1.000)
	if err != nil {
		return fmt.Errorf("seed po_client_item: %w", err)
	}

	_, err = db.Exec(ctx, `
		INSERT INTO WORK_ORDER (ID_WO, BUYER, MODEL, QTY, FOB_CMT, ID_PO_Client_Item)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (ID_WO) DO NOTHING
	`, 1, "SEED-BUYER", "SEED-MODEL", 100, false, 1)
	if err != nil {
		return fmt.Errorf("seed work_order: %w", err)
	}

	_, err = db.Exec(ctx, `
		INSERT INTO WORK_ORDER_SHELL (ID_WO_SHELL, FABRIC, CONS, COLOR, ALLOW, BERAT_1_YD, ID_WO)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (ID_WO_SHELL) DO NOTHING
	`, 1, "SEED-FABRIC", 1.000, "BLACK", 0, 1.000, 1)
	if err != nil {
		return fmt.Errorf("seed work_order_shell: %w", err)
	}

	_, err = db.Exec(ctx, `
		INSERT INTO WORK_ORDER_SHELL_SIZE (ID_WO_SHELL_SIZE, SIZE, QTY, RATIO, ID_WO_SHELL)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (ID_WO_SHELL_SIZE) DO NOTHING
	`, 1, "M", 100, 1, 1)
	if err != nil {
		return fmt.Errorf("seed work_order_shell_size: %w", err)
	}

	slog.Info("report pengiriman dependencies seeded")
	return nil
}

func seedReportPengiriman(ctx context.Context, db *pgxpool.Pool) error {
	baseDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	insertedCount := 0

	for i := int32(1); i <= 10; i++ {
		quantity := int32(40 + i*5)
		reportDate := baseDate.AddDate(0, 0, int(i-1)).Format("2006-01-02")

		result, err := db.Exec(ctx, `
			INSERT INTO REPORT_PENGIRIMAN (ID_REPORT_PENGIRIMAN, "DATE", Quantity, ID_WO_SHELL_SIZE)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (ID_REPORT_PENGIRIMAN) DO NOTHING
		`, i, reportDate, quantity, 1)
		if err != nil {
			return fmt.Errorf("insert report_pengiriman id %d: %w", i, err)
		}

		if result.RowsAffected() > 0 {
			insertedCount++
		}
	}

	slog.Info("report pengiriman seeded", slog.Int("target_rows", 10), slog.Int("rows_inserted", insertedCount))
	return nil
}
