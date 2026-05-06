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

	// 1. Seed Company (Satu saja)
	err = seedCompany(ctx, dbPool)
	if err != nil {
		slog.Error("failed to seed company", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// 2. Seed Hak Akses
	err = seedHakAkses(ctx, dbPool)
	if err != nil {
		slog.Error("failed to seed hak akses", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// 3. Seed Departemen
	err = seedDepartemen(ctx, dbPool)
	if err != nil {
		slog.Error("failed to seed departemen", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// 4. Seed Mitra (5 Mitra, ID 1 = Permatatex)
	err = seedMitra(ctx, dbPool)
	if err != nil {
		slog.Error("failed to seed mitra", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// 5. Seed Jenis Barang (5 Jenis)
	err = seedJenisBarang(ctx, dbPool)
	if err != nil {
		slog.Error("failed to seed jenis barang", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// 6. Seed Barang (5 Barang)
	err = seedBarang(ctx, dbPool)
	if err != nil {
		slog.Error("failed to seed barang", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// 7. Seed Super Admin User
	err = seedUser(ctx, dbPool)
	if err != nil {
		slog.Error("failed to seed user", slog.String("error", err.Error()))
		os.Exit(1)
	}

	slog.Info("seeding completed successfully")
}

func seedCompany(ctx context.Context, db *pgxpool.Pool) error {
	var exists bool
	err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM COMPANY WHERE ID_COMPANY = 1)`).Scan(&exists)
	if err != nil {
		return err
	}

	if !exists {
		_, err = db.Exec(ctx, `
			INSERT INTO COMPANY (ID_COMPANY, NAMA, ALAMAT, EMAIL, NO_TELP, ABOUT)
			VALUES (1, 'PT. Permata Anugrah Kusuma', 'Kawasan Industri Bandung', 'info@permatatex.com', '022-123456', 'Garment Manufacturing Expert')
		`)
		if err != nil {
			return err
		}
		slog.Info("company seeded: PT. Permata Anugrah Kusuma")
	}
	return nil
}

func seedMitra(ctx context.Context, db *pgxpool.Pool) error {
	mitras := []struct {
		ID   int
		Nama string
		Tipe string
	}{
		{1, "Permatatex", "INTERNAL"},
		{2, "Vendor Kain Sejahtera", "SUPPLIER"},
		{3, "Toko Benang Jaya", "SUPPLIER"},
		{4, "Garment Client Global", "CLIENT"},
		{5, "Fashion Brand Indonesia", "CLIENT"},
	}

	for _, m := range mitras {
		var exists bool
		err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM MITRA WHERE NAMA_PERUSAHAAN = $1)`, m.Nama).Scan(&exists)
		if err != nil {
			return err
		}
		if !exists {
			_, err = db.Exec(ctx, `
				INSERT INTO MITRA (ID_MITRA, NAMA_PERUSAHAAN, TIPE_PERUSAHAAN, EMAIL, NO_TELP, ALAMAT, KOTA)
				VALUES ($1, $2, $3, $4, $5, $6, $7)
			`, m.ID, m.Nama, m.Tipe, "contact@"+m.Nama+".com", "0812345678", "Jl. Industri No. "+fmt.Sprint(m.ID), "Bandung")
			if err != nil {
				return err
			}
			slog.Info("mitra seeded", slog.String("name", m.Nama))
		}
	}
	return nil
}

func seedJenisBarang(ctx context.Context, db *pgxpool.Pool) error {
	types := []struct {
		Nama string
		Kode string
	}{
		{"Bahan Baku", "RAW"},
		{"Aksesoris", "ACC"},
		{"WIP", "WIP"},
		{"Barang Jadi", "FG"},
		{"Kemasan", "PKG"},
	}
	for _, t := range types {
		var exists bool
		err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM JENIS_BARANG WHERE KODE = $1)`, t.Kode).Scan(&exists)
		if err != nil {
			return err
		}
		if !exists {
			_, err = db.Exec(ctx, `INSERT INTO JENIS_BARANG (NAMA_JENIS_BARANG, KODE) VALUES ($1, $2)`, t.Nama, t.Kode)
			if err != nil {
				return err
			}
			slog.Info("jenis barang seeded", slog.String("code", t.Kode))
		}
	}
	return nil
}

func seedBarang(ctx context.Context, db *pgxpool.Pool) error {
	items := []struct {
		Nama      string
		Kode      string
		JenisKode string
	}{
		{"Kain Cotton Combed 30s", "CTN-30S", "RAW"},
		{"Benang Jahit Putih", "BNG-WHT", "ACC"},
		{"Kancing Kemeja 12mm", "KNC-12", "ACC"},
		{"Kaus Polos Hitam XL", "TSH-BLK-XL", "FG"},
		{"Plastik Packing 25x35", "PL-2535", "PKG"},
	}

	for _, it := range items {
		var exists bool
		err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM BARANG WHERE KODE = $1)`, it.Kode).Scan(&exists)
		if err != nil {
			return err
		}
		if !exists {
			_, err = db.Exec(ctx, `
				INSERT INTO BARANG (NAMA_BARANG, KODE, ID_JENIS_BARANG, ID_MITRA)
				VALUES ($1, $2, 
					(SELECT ID_JENIS_BARANG FROM JENIS_BARANG WHERE KODE = $3 LIMIT 1),
					(SELECT ID_MITRA FROM MITRA WHERE ID_MITRA = 1 LIMIT 1)
				)
			`, it.Nama, it.Kode, it.JenisKode)
			if err != nil {
				return err
			}
			slog.Info("barang seeded", slog.String("name", it.Nama))
		}
	}
	return nil
}

func seedHakAkses(ctx context.Context, db *pgxpool.Pool) error {
	permissions := []string{
		"USER_READ", "USER_CREATE", "USER_UPDATE", "USER_DELETE",
		"ITEM_READ", "ITEM_CREATE", "ITEM_UPDATE", "ITEM_DELETE",
		"PO_READ", "PO_CREATE", "REPORT_READ", "ALL_ACCESS",
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
		}
	}
	return nil
}

func seedDepartemen(ctx context.Context, db *pgxpool.Pool) error {
	depts := []string{"IT", "PRODUKSI", "GUDANG", "OFFICE"}
	for _, d := range depts {
		var exists bool
		err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM DEPARTEMEN WHERE NAMA_DEPARTEMEN = $1)`, d).Scan(&exists)
		if err != nil {
			return err
		}
		if !exists {
			_, err = db.Exec(ctx, `INSERT INTO DEPARTEMEN (NAMA_DEPARTEMEN) VALUES ($1)`, d)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func seedUser(ctx context.Context, db *pgxpool.Pool) error {
	username := "super-admin"
	password := "admin123"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	var exists bool
	err = db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM USERS WHERE username = $1)`, username).Scan(&exists)
	if err != nil {
		return err
	}

	if !exists {
		_, err = db.Exec(ctx, `
			INSERT INTO USERS (USERNAME, PASSWORD, IS_MANAGER, ID_DEPARTEMEN)
			VALUES ($1, $2, $3, (SELECT ID_DEPARTEMEN FROM DEPARTEMEN WHERE NAMA_DEPARTEMEN = 'IT' LIMIT 1))
		`, username, string(hashedPassword), true)
		if err != nil {
			return err
		}
		slog.Info("user seeded", slog.String("username", username))
	}
	return nil
}
