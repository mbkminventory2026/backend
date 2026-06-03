package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"unicode"

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
	err = seedRoles(ctx, dbPool)
	if err != nil {
		slog.Error("failed to seed roles", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// 4. Seed Role Hak Akses
	err = seedRoleHakAkses(ctx, dbPool)
	if err != nil {
		slog.Error("failed to seed role hak akses", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// 5. Seed Departemen
	err = seedDepartemen(ctx, dbPool)
	if err != nil {
		slog.Error("failed to seed departemen", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// 6. Seed Mitra (5 Mitra, ID 1 = Permatatex)
	err = seedMitra(ctx, dbPool)
	if err != nil {
		slog.Error("failed to seed mitra", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// 7. Seed Jenis Barang (5 Jenis)
	err = seedJenisBarang(ctx, dbPool)
	if err != nil {
		slog.Error("failed to seed jenis barang", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// 8. Seed Barang (5 Barang)
	err = seedBarang(ctx, dbPool)
	if err != nil {
		slog.Error("failed to seed barang", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// 9. Seed Super Admin User
	err = seedUser(ctx, dbPool)
	if err != nil {
		slog.Error("failed to seed user", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// 10. Sync Sequences
	err = syncSequences(ctx, dbPool)
	if err != nil {
		slog.Error("failed to sync sequences", slog.String("error", err.Error()))
		os.Exit(1)
	}

	slog.Info("seeding completed successfully")
}

func syncSequences(ctx context.Context, db *pgxpool.Pool) error {
	queries := []struct {
		SeqName   string
		TableName string
		ColName   string
	}{
		{"company_id_company_seq", "company", "id_company"},
		{"mitra_id_mitra_seq", "mitra", "id_mitra"},
		{"hak_akses_id_hak_akses_seq", "hak_akses", "id_hak_akses"},
		{"roles_id_role_seq", "roles", "id_role"},
		{"departemen_id_departemen_seq", "departemen", "id_departemen"},
		{"jenis_barang_id_jenis_barang_seq", "jenis_barang", "id_jenis_barang"},
		{"barang_id_barang_seq", "barang", "id_barang"},
		{"users_id_user_seq", "users", "id_user"},
	}

	for _, q := range queries {
		query := fmt.Sprintf("SELECT setval('%s', COALESCE((SELECT MAX(%s) FROM %s), 1))", q.SeqName, q.ColName, q.TableName)
		_, err := db.Exec(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to sync sequence %s: %w", q.SeqName, err)
		}
		slog.Info("sequence synchronized", slog.String("sequence", q.SeqName))
	}
	return nil
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
	permissionCodes := []string{
		"USER_READ", "USER_CREATE", "USER_UPDATE", "USER_DELETE", "USER_APPROVE",
		"ROLE_READ", "ROLE_CREATE", "ROLE_UPDATE", "ROLE_DELETE", "USER_ROLE_ASSIGN",
		"PERMISSION_READ", "PERMISSION_CREATE", "PERMISSION_UPDATE", "PERMISSION_DELETE",
		"MASTER_BARANG_READ", "MASTER_BARANG_CREATE", "MASTER_BARANG_UPDATE", "MASTER_BARANG_DELETE",
		"MASTER_MITRA_READ", "MASTER_MITRA_CREATE", "MASTER_MITRA_UPDATE", "MASTER_MITRA_DELETE",
		"MASTER_JENIS_BARANG_READ", "MASTER_JENIS_BARANG_CREATE", "MASTER_JENIS_BARANG_UPDATE", "MASTER_JENIS_BARANG_DELETE",
		"MASTER_COMPANY_READ", "MASTER_COMPANY_CREATE", "MASTER_COMPANY_UPDATE", "MASTER_COMPANY_DELETE",
		"MASTER_DEPARTEMEN_READ", "MASTER_DEPARTEMEN_CREATE", "MASTER_DEPARTEMEN_UPDATE", "MASTER_DEPARTEMEN_DELETE",
		"PO_CLIENT_READ", "PO_CLIENT_CREATE", "PO_CLIENT_UPDATE",
		"PR_INTERNAL_READ", "PR_INTERNAL_CREATE", "PR_INTERNAL_UPDATE", "PR_INTERNAL_APPROVE",
		"PO_INTERNAL_READ", "PO_INTERNAL_CREATE", "PO_INTERNAL_UPDATE", "PO_INTERNAL_APPROVE",
		"WO_READ", "WO_CREATE", "WO_UPDATE", "WO_CLOSE",
		"PRODUCTION_SUMMARY_READ",
		"PRODUCTION_REPORT_READ", "PRODUCTION_REPORT_CREATE", "PRODUCTION_REPORT_UPDATE",
		"TIMELINE_READ", "TIMELINE_CREATE", "TIMELINE_UPDATE",
		"MARKER_PLAN_READ", "MARKER_PLAN_CREATE", "MARKER_PLAN_UPDATE",
		"CUTTING_PLAN_READ", "CUTTING_PLAN_CREATE", "CUTTING_PLAN_UPDATE",
		"INVENTORY_RECEIVE", "INVENTORY_ISSUE",
		"PACKING_LIST_READ", "PACKING_LIST_CREATE", "PACKING_LIST_UPDATE", "PACKING_LIST_APPROVE",
		"SURAT_JALAN_CLIENT_READ", "SURAT_JALAN_INTERNAL_READ", "SURAT_JALAN_CREATE", "SURAT_JALAN_UPDATE",
		"REPORT_READ", "LOG_READ", "DASHBOARD_READ",
		"ALL_ACCESS",
	}

	for _, code := range permissionCodes {
		name, description, domain, action := inferPermissionMetadata(code)
		var exists bool
		err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM HAK_AKSES WHERE KODE_PERMISSION = $1)`, code).Scan(&exists)
		if err != nil {
			return err
		}
		if !exists {
			_, err = db.Exec(ctx, `
				INSERT INTO HAK_AKSES (KODE_PERMISSION, NAMA_HALAMAN, DESKRIPSI, DOMAIN_PERMISSION, AKSI_PERMISSION)
				VALUES ($1, $2, $3, $4, $5)
			`, code, name, description, domain, action)
			if err != nil {
				return err
			}
		} else {
			_, err = db.Exec(ctx, `
				UPDATE HAK_AKSES
				SET NAMA_HALAMAN = $2,
					DESKRIPSI = $3,
					DOMAIN_PERMISSION = $4,
					AKSI_PERMISSION = $5
				WHERE KODE_PERMISSION = $1
			`, code, name, description, domain, action)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func inferPermissionMetadata(code string) (name string, description string, domain string, action string) {
	if code == "ALL_ACCESS" {
		return "All Access", "Emergency full access", "system", "access"
	}

	parts := strings.Split(code, "_")
	action = strings.ToLower(parts[len(parts)-1])
	domain = strings.ToLower(strings.Join(parts[:len(parts)-1], "_"))

	words := make([]string, 0, len(parts))
	for _, part := range parts {
		words = append(words, capitalize(strings.ToLower(part)))
	}
	name = strings.Join(words, " ")
	description = "Allows " + strings.ToLower(strings.ReplaceAll(name, " ", " "))
	return name, description, domain, action
}

func capitalize(s string) string {
	if s == "" {
		return ""
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func seedRoles(ctx context.Context, db *pgxpool.Pool) error {
	roles := []string{
		"SUPER_ADMIN",
		"OPERATOR",
		"ADMIN_KEUANGAN",
		"ADMIN_PRODUKSI",
		"ADMIN_GUDANG",
		"MANAGER",
		"CLIENT",
	}

	for _, role := range roles {
		var exists bool
		err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM ROLES WHERE NAMA_ROLE = $1)`, role).Scan(&exists)
		if err != nil {
			return err
		}
		if !exists {
			_, err = db.Exec(ctx, `INSERT INTO ROLES (NAMA_ROLE) VALUES ($1)`, role)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func seedRoleHakAkses(ctx context.Context, db *pgxpool.Pool) error {
	rolePermissions := map[string][]string{
		"SUPER_ADMIN": {"ALL_ACCESS"},
		"OPERATOR": {
			"USER_READ", "USER_CREATE", "USER_UPDATE", "USER_DELETE", "USER_APPROVE",
			"ROLE_READ", "ROLE_CREATE", "ROLE_UPDATE", "ROLE_DELETE", "USER_ROLE_ASSIGN",
			"PERMISSION_READ", "PERMISSION_CREATE", "PERMISSION_UPDATE", "PERMISSION_DELETE",
			"LOG_READ",
		},
		"ADMIN_KEUANGAN": {
			"MASTER_MITRA_READ", "MASTER_BARANG_READ", "MASTER_JENIS_BARANG_READ", "MASTER_COMPANY_READ",
			"PO_CLIENT_READ",
			"PR_INTERNAL_READ", "PR_INTERNAL_APPROVE",
			"PO_INTERNAL_READ", "PO_INTERNAL_CREATE", "PO_INTERNAL_UPDATE", "PO_INTERNAL_APPROVE",
			"REPORT_READ",
		},
		"ADMIN_PRODUKSI": {
			"MASTER_BARANG_READ", "MASTER_MITRA_READ", "MASTER_JENIS_BARANG_READ", "MASTER_COMPANY_READ", "MASTER_DEPARTEMEN_READ",
			"PO_CLIENT_READ", "PO_CLIENT_CREATE", "PO_CLIENT_UPDATE",
			"WO_READ", "WO_CREATE", "WO_UPDATE", "WO_CLOSE",
			"PRODUCTION_SUMMARY_READ",
			"PRODUCTION_REPORT_READ", "PRODUCTION_REPORT_CREATE", "PRODUCTION_REPORT_UPDATE",
			"TIMELINE_READ", "TIMELINE_CREATE", "TIMELINE_UPDATE",
			"MARKER_PLAN_READ", "MARKER_PLAN_CREATE", "MARKER_PLAN_UPDATE",
			"CUTTING_PLAN_READ", "CUTTING_PLAN_CREATE", "CUTTING_PLAN_UPDATE",
			"PACKING_LIST_READ", "PACKING_LIST_CREATE", "PACKING_LIST_UPDATE",
			"SURAT_JALAN_CLIENT_READ", "SURAT_JALAN_INTERNAL_READ", "SURAT_JALAN_CREATE", "SURAT_JALAN_UPDATE",
			"REPORT_READ",
		},
		"ADMIN_GUDANG": {
			"MASTER_BARANG_READ", "MASTER_MITRA_READ", "MASTER_JENIS_BARANG_READ",
			"PR_INTERNAL_READ", "PR_INTERNAL_CREATE", "PR_INTERNAL_UPDATE",
			"INVENTORY_RECEIVE", "INVENTORY_ISSUE",
			"SURAT_JALAN_CLIENT_READ", "SURAT_JALAN_INTERNAL_READ",
			"REPORT_READ",
		},
		"MANAGER": {
			"USER_READ",
			"MASTER_BARANG_READ", "MASTER_MITRA_READ", "MASTER_JENIS_BARANG_READ", "MASTER_COMPANY_READ", "MASTER_DEPARTEMEN_READ",
			"PO_CLIENT_READ", "PR_INTERNAL_READ", "PO_INTERNAL_READ",
			"WO_READ", "PRODUCTION_SUMMARY_READ", "PRODUCTION_REPORT_READ",
			"TIMELINE_READ", "MARKER_PLAN_READ", "CUTTING_PLAN_READ",
			"PACKING_LIST_READ", "SURAT_JALAN_CLIENT_READ", "SURAT_JALAN_INTERNAL_READ",
			"REPORT_READ", "LOG_READ", "DASHBOARD_READ",
			"PR_INTERNAL_APPROVE", "PO_INTERNAL_APPROVE", "PACKING_LIST_APPROVE",
		},
		"CLIENT": {
			"PO_CLIENT_READ", "WO_READ", "PRODUCTION_SUMMARY_READ", "PACKING_LIST_READ", "SURAT_JALAN_CLIENT_READ",
		},
	}

	for roleName, permissions := range rolePermissions {
		for _, permissionCode := range permissions {
			_, err := db.Exec(ctx, `
				INSERT INTO ROLE_HAK_AKSES (ID_ROLE, ID_HAK_AKSES)
				SELECT r.ID_ROLE, h.ID_HAK_AKSES
				FROM ROLES r
				JOIN HAK_AKSES h ON h.KODE_PERMISSION = $2
				WHERE r.NAMA_ROLE = $1
				ON CONFLICT (ID_ROLE, ID_HAK_AKSES) DO NOTHING
			`, roleName, permissionCode)
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
			INSERT INTO USERS (USERNAME, PASSWORD, ID_ROLE, ID_DEPARTEMEN, STATUS)
			VALUES (
				$1,
				$2,
				(SELECT ID_ROLE FROM ROLES WHERE NAMA_ROLE = 'SUPER_ADMIN' LIMIT 1),
				(SELECT ID_DEPARTEMEN FROM DEPARTEMEN WHERE NAMA_DEPARTEMEN = 'IT' LIMIT 1),
				'active'
			)
		`, username, string(hashedPassword))
		if err != nil {
			return err
		}
		slog.Info("user seeded", slog.String("username", username))
	} else {
		_, err = db.Exec(ctx, `
			UPDATE USERS
			SET ID_ROLE = (SELECT ID_ROLE FROM ROLES WHERE NAMA_ROLE = 'SUPER_ADMIN' LIMIT 1)
			WHERE USERNAME = $1
		`, username)
		if err != nil {
			return err
		}
	}
	return nil
}
