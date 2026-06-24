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

	// 0. Clean and Migrate existing seed data casing to uppercase
	err = cleanAndMigrateSeedingCasing(ctx, dbPool)
	if err != nil {
		slog.Error("failed to migrate existing seed data casing", slog.String("error", err.Error()))
		os.Exit(1)
	}

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

	// 9. Seed Master Size
	err = seedMasterSizes(ctx, dbPool)
	if err != nil {
		slog.Error("failed to seed master sizes", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Seed Master Warna
	err = seedMasterWarna(ctx, dbPool)
	if err != nil {
		slog.Error("failed to seed master warna", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// 10. Seed bootstrap users
	err = seedSystemUsers(ctx, dbPool)
	if err != nil {
		slog.Error("failed to seed system users", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// 11. Seed PO Client & Items
	err = seedPOClient(ctx, dbPool)
	if err != nil {
		slog.Error("failed to seed PO Client", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// 12. Seed Work Order
	err = seedWorkOrder(ctx, dbPool)
	if err != nil {
		slog.Error("failed to seed Work Order", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// 13. Seed Production Reports
	err = seedProductionReports(ctx, dbPool)
	if err != nil {
		slog.Error("failed to seed Production Reports", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// 14. Seed Marker Plan
	if err := seedMarkerPlan(ctx, dbPool); err != nil {
		logger.Error("Failed to seed marker plan", slog.String("error", err.Error()))
	}

	// 15. Seed Production Master
	if err := seedProductionMaster(ctx, dbPool); err != nil {
		logger.Error("Failed to seed production master", slog.String("error", err.Error()))
	}

	// 16. Seed Timeline Plan
	err = seedTimelinePlan(ctx, dbPool)
	if err != nil {
		slog.Error("failed to seed Timeline Plan", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// 17. Seed PR Internal (pending approval, for testing approve flow)
	err = seedPRInternal(ctx, dbPool)
	if err != nil {
		slog.Error("failed to seed PR Internal", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// 18. Sync Sequences
	err = syncSequences(ctx, dbPool)
	if err != nil {
		slog.Error("failed to sync sequences", slog.String("error", err.Error()))
		os.Exit(1)
	}

	slog.Info("seeding completed successfully")
}

func cleanAndMigrateSeedingCasing(ctx context.Context, db *pgxpool.Pool) error {
	_, err := db.Exec(ctx, `UPDATE PO_CLIENT_ITEM SET COLOUR = UPPER(BTRIM(COLOUR))`)
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, `UPDATE WORK_ORDER_SHELL SET COLOR = UPPER(BTRIM(COLOR))`)
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, `UPDATE WORK_ORDER_TRIM SET COLOR = UPPER(BTRIM(COLOR))`)
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, `UPDATE PACKING_LIST_ITEM SET COLOR = UPPER(BTRIM(COLOR))`)
	if err != nil {
		return err
	}
	// Also migrate any legacy size records just in case
	_, err = db.Exec(ctx, `UPDATE REKONSILIASI_MATERIAL SET SIZE = UPPER(BTRIM(SIZE))`)
	if err != nil {
		return err
	}
	return nil
}

func syncSequences(ctx context.Context, db *pgxpool.Pool) error {
	queries := []struct {
		SeqName   string
		TableName string
		ColName   string
	}{
		{"company_id_company_seq", "profil_perusahaan", "id_profil_perusahaan"},
		{"mitra_id_mitra_seq", "mitra", "id_mitra"},
		{"hak_akses_id_hak_akses_seq", "hak_akses", "id_hak_akses"},
		{"roles_id_role_seq", "roles", "id_role"},
		{"departemen_id_departemen_seq", "departemen", "id_departemen"},
		{"jenis_barang_id_jenis_barang_seq", "jenis_barang", "id_jenis_barang"},
		{"barang_id_barang_seq", "barang", "id_barang"},
		{"master_size_id_size_seq", "master_size", "id_size"},
		{"users_id_user_seq", "users", "id_user"},
		{"po_client_id_po_client_seq", "po_client", "id_po_client"},
		{"po_client_item_id_po_client_item_seq", "po_client_item", "id_po_client_item"},
		{"work_order_id_wo_seq", "work_order", "id_wo"},
		{"work_order_shell_id_wo_shell_seq", "work_order_shell", "id_wo_shell"},
		{"work_order_shell_size_id_wo_shell_size_seq", "work_order_shell_size", "id_wo_shell_size"},
		{"work_order_trim_id_wo_trim_seq", "work_order_trim", "id_wo_trim"},
		{"material_list_item_id_material_list_item_seq", "material_list_item", "id_material_list_item"},
		{"material_list_id_material_list_seq", "material_list", "id_material_list"},
		{"received_id_received_seq", "received", "id_received"},
		{"report_cutting_id_report_cutting_seq", "report_cutting", "id_report_cutting"},
		{"report_sewing_id_report_sewing_seq", "report_sewing", "id_report_sewing"},
		{"report_qc_finish_id_report_qc_finishing_seq", "report_qc_finish", "id_report_qc_finishing"},
		{"report_packing_id_report_packing_seq", "report_packing", "id_report_packing"},
		{"report_pengiriman_id_report_pengiriman_seq", "report_pengiriman", "id_report_pengiriman"},
		{"marker_plan_id_marker_plan_seq", "marker_plan", "id_marker_plan"},
		{"komponen_marker_plan_id_komponen_marker_seq", "komponen_marker_plan", "id_komponen_marker"},
		{"ratio_marker_id_ratio_marker_seq", "ratio_marker", "id_ratio_marker"},
		{"ratio_size_marker_id_ratio_size_marker_seq", "ratio_size_marker", "id_ratio_size_marker"},
		{"timeline_plan_produksi_id_timeline_seq", "timeline_plan_produksi", "id_timeline"},
		{"wo_shell_plan_id_wo_shell_plan_seq", "wo_shell_plan", "id_wo_shell_plan"},
		{"spreading_cutting_plan_id_spreading_cutting_plan_seq", "spreading_cutting_plan", "id_spreading_cutting_plan"},
		{"komponen_spreading_cutting_plan_id_komponen_spreading_seq", "komponen_spreading_cutting_plan", "id_komponen_spreading"},
		{"ratio_spreading_id_ratio_spreading_seq", "ratio_spreading", "id_ratio_spreading"},
		{"ratio_size_spreading_id_ratio_size_spreading_seq", "ratio_size_spreading", "id_ratio_size_spreading"},
		{"production_line_id_production_line_seq", "production_line", "id_production_line"},
		{"production_status_plan_id_production_status_plan_seq", "production_status_plan", "id_production_status_plan"},
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
	err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM PROFIL_PERUSAHAAN WHERE ID_PROFIL_PERUSAHAAN = 1)`).Scan(&exists)
	if err != nil {
		return err
	}

	if !exists {
		_, err = db.Exec(ctx, `
			INSERT INTO PROFIL_PERUSAHAAN (ID_PROFIL_PERUSAHAAN, NAMA, ALAMAT, EMAIL, NO_TELP, ABOUT)
			VALUES (1, 'PT. Permata Anugrah Kusuma', 'Kawasan Industri Bandung', 'info@permatatex.com', '022-123456', 'Garment Manufacturing Expert')
		`)
		if err != nil {
			return err
		}
		slog.Info("profil perusahaan seeded: PT. Permata Anugrah Kusuma")
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

func seedMasterSizes(ctx context.Context, db *pgxpool.Pool) error {
	sizes := []string{"XXS", "XS", "S", "M", "L", "XL", "XXL", "XXXL", "ALL SIZE", "FREE SIZE"}

	for _, sizeName := range sizes {
		var exists bool
		err := db.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1
				FROM MASTER_SIZE
				WHERE LOWER(BTRIM(NAMA_SIZE)) = LOWER(BTRIM($1))
			)
		`, sizeName).Scan(&exists)
		if err != nil {
			return err
		}
		if exists {
			continue
		}

		_, err = db.Exec(ctx, `INSERT INTO MASTER_SIZE (NAMA_SIZE) VALUES ($1)`, sizeName)
		if err != nil {
			return err
		}
		slog.Info("master size seeded", slog.String("name", sizeName))
	}

	return nil
}

func seedMasterWarna(ctx context.Context, db *pgxpool.Pool) error {
	colors := []struct {
		Name string
		Hex  string
	}{
		{"BLACK", "#000000"},
		{"WHITE", "#FFFFFF"},
		{"NAVY", "#000080"},
		{"MAROON", "#800000"},
		{"GREY", "#808080"},
		{"RED", "#FF0000"},
		{"BLUE", "#0000FF"},
		{"GREEN", "#008000"},
		{"YELLOW", "#FFFF00"},
	}

	for _, c := range colors {
		var exists bool
		err := db.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1
				FROM WARNA
				WHERE LOWER(BTRIM(NAMA_WARNA)) = LOWER(BTRIM($1))
			)
		`, c.Name).Scan(&exists)
		if err != nil {
			return err
		}
		if exists {
			continue
		}

		_, err = db.Exec(ctx, `INSERT INTO WARNA (NAMA_WARNA, KODE_HEX) VALUES ($1, $2)`, c.Name, c.Hex)
		if err != nil {
			return err
		}
		slog.Info("master warna seeded", slog.String("name", c.Name))
	}

	return nil
}

func seedHakAkses(ctx context.Context, db *pgxpool.Pool) error {
	permissionCodes := []string{
		"AUTH_CHANGE_PASSWORD",
		"USER_READ", "USER_CREATE", "USER_UPDATE", "USER_DELETE", "USER_APPROVE",
		"USER_TEMP_PASSWORD_CREATE",
		"ROLE_READ", "ROLE_CREATE", "ROLE_UPDATE", "ROLE_DELETE", "USER_ROLE_ASSIGN",
		"PERMISSION_READ", "PERMISSION_CREATE", "PERMISSION_UPDATE", "PERMISSION_DELETE",
		"MASTER_BARANG_READ", "MASTER_BARANG_CREATE", "MASTER_BARANG_UPDATE", "MASTER_BARANG_DELETE",
		"MASTER_WARNA_READ", "MASTER_WARNA_CREATE", "MASTER_WARNA_UPDATE", "MASTER_WARNA_DELETE",
		"MASTER_SIZE_READ", "MASTER_SIZE_CREATE", "MASTER_SIZE_UPDATE", "MASTER_SIZE_DELETE",
		"MASTER_MITRA_READ", "MASTER_MITRA_CREATE", "MASTER_MITRA_UPDATE", "MASTER_MITRA_DELETE",
		"MASTER_JENIS_BARANG_READ", "MASTER_JENIS_BARANG_CREATE", "MASTER_JENIS_BARANG_UPDATE", "MASTER_JENIS_BARANG_DELETE",
		"MASTER_PROFIL_PERUSAHAAN_READ", "MASTER_PROFIL_PERUSAHAAN_CREATE", "MASTER_PROFIL_PERUSAHAAN_UPDATE", "MASTER_PROFIL_PERUSAHAAN_DELETE",
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
		"DATA_APPROVE_CUTTING_PLAN_READ", "DATA_APPROVE_CUTTING_PLAN_CREATE",
		"INVENTORY_RECEIVE", "INVENTORY_ISSUE",
		"PACKING_LIST_READ", "PACKING_LIST_CREATE", "PACKING_LIST_UPDATE", "PACKING_LIST_APPROVE",
		"SURAT_JALAN_CLIENT_READ", "SURAT_JALAN_CLIENT_CREATE", "SURAT_JALAN_CLIENT_UPDATE", "SURAT_JALAN_CLIENT_DELETE",
		"SURAT_JALAN_INTERNAL_READ", "SURAT_JALAN_INTERNAL_CREATE", "SURAT_JALAN_INTERNAL_UPDATE", "SURAT_JALAN_INTERNAL_DELETE",
		"REKONSILIASI_READ", "REKONSILIASI_CREATE", "REKONSILIASI_UPDATE",
		"PASSWORD_RESET_REQUEST_CREATE",
		"PASSWORD_RESET_REQUEST_READ", "PASSWORD_RESET_REQUEST_APPROVE", "PASSWORD_RESET_REQUEST_REJECT",
		"REPORT_READ", "LOG_READ", "DASHBOARD_READ", "AI_ESTIMATION_READ",
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
	switch code {
	case "ALL_ACCESS":
		return "All Access", "Emergency full access", "system", "access"
	case "AUTH_CHANGE_PASSWORD":
		return "Auth Change Password", "Allows authenticated users to change their own password", "auth", "change_password"
	case "PASSWORD_RESET_REQUEST_CREATE":
		return "Password Reset Request Create", "Allows users to submit a password reset request", "password_reset_request", "create"
	case "PASSWORD_RESET_REQUEST_READ":
		return "Password Reset Request Read", "Allows admin sistem to read password reset requests", "password_reset_request", "read"
	case "PASSWORD_RESET_REQUEST_APPROVE":
		return "Password Reset Request Approve", "Allows admin sistem to approve password reset requests", "password_reset_request", "approve"
	case "PASSWORD_RESET_REQUEST_REJECT":
		return "Password Reset Request Reject", "Allows admin sistem to reject password reset requests", "password_reset_request", "reject"
	case "USER_TEMP_PASSWORD_CREATE":
		return "User Temporary Password Create", "Allows admin sistem to generate a temporary password while creating or resetting users", "user_temp_password", "create"
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
		"ADMIN_SISTEM",
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
		"ADMIN_SISTEM": {
			"AUTH_CHANGE_PASSWORD",
			"USER_READ", "USER_CREATE", "USER_UPDATE", "USER_DELETE", "USER_APPROVE",
			"USER_TEMP_PASSWORD_CREATE",
			"ROLE_READ", "ROLE_CREATE", "ROLE_UPDATE", "ROLE_DELETE", "USER_ROLE_ASSIGN",
			"PERMISSION_READ", "PERMISSION_CREATE", "PERMISSION_UPDATE", "PERMISSION_DELETE",
			"MASTER_DEPARTEMEN_READ", "MASTER_DEPARTEMEN_CREATE", "MASTER_DEPARTEMEN_UPDATE", "MASTER_DEPARTEMEN_DELETE",
			"MASTER_PROFIL_PERUSAHAAN_READ", "MASTER_PROFIL_PERUSAHAAN_CREATE", "MASTER_PROFIL_PERUSAHAAN_UPDATE", "MASTER_PROFIL_PERUSAHAAN_DELETE",
			"PASSWORD_RESET_REQUEST_CREATE",
			"PASSWORD_RESET_REQUEST_READ", "PASSWORD_RESET_REQUEST_APPROVE", "PASSWORD_RESET_REQUEST_REJECT",
			"LOG_READ", "DASHBOARD_READ",
		},
		"ADMIN_KEUANGAN": {
			"AUTH_CHANGE_PASSWORD", "PASSWORD_RESET_REQUEST_CREATE",
			"MASTER_MITRA_READ", "MASTER_MITRA_CREATE", "MASTER_MITRA_UPDATE", "MASTER_MITRA_DELETE",
			"MASTER_PROFIL_PERUSAHAAN_READ", "MASTER_PROFIL_PERUSAHAAN_CREATE", "MASTER_PROFIL_PERUSAHAAN_UPDATE", "MASTER_PROFIL_PERUSAHAAN_DELETE",
			"MASTER_BARANG_READ", "MASTER_JENIS_BARANG_READ", "MASTER_WARNA_READ", "MASTER_SIZE_READ", "MASTER_DEPARTEMEN_READ",
			"PO_CLIENT_READ", "PO_CLIENT_CREATE", "PO_CLIENT_UPDATE",
			"PR_INTERNAL_READ", "PR_INTERNAL_APPROVE",
			"PO_INTERNAL_READ", "PO_INTERNAL_CREATE", "PO_INTERNAL_UPDATE",
			"WO_READ", "WO_CLOSE", "REKONSILIASI_READ",
			"REPORT_READ", "DASHBOARD_READ",
		},
		"ADMIN_PRODUKSI": {
			"AUTH_CHANGE_PASSWORD", "PASSWORD_RESET_REQUEST_CREATE",
			"MASTER_BARANG_READ", "MASTER_BARANG_CREATE", "MASTER_BARANG_UPDATE", "MASTER_BARANG_DELETE",
			"MASTER_JENIS_BARANG_READ", "MASTER_JENIS_BARANG_CREATE", "MASTER_JENIS_BARANG_UPDATE", "MASTER_JENIS_BARANG_DELETE",
			"MASTER_WARNA_READ", "MASTER_WARNA_CREATE", "MASTER_WARNA_UPDATE", "MASTER_WARNA_DELETE",
			"MASTER_SIZE_READ", "MASTER_SIZE_CREATE", "MASTER_SIZE_UPDATE", "MASTER_SIZE_DELETE",
			"MASTER_MITRA_READ", "MASTER_PROFIL_PERUSAHAAN_READ", "MASTER_DEPARTEMEN_READ",
			"PR_INTERNAL_READ", "PR_INTERNAL_APPROVE",
			"PO_CLIENT_READ",
			"WO_READ", "WO_CREATE", "WO_UPDATE",
			"PRODUCTION_SUMMARY_READ",
			"PRODUCTION_REPORT_READ", "PRODUCTION_REPORT_CREATE", "PRODUCTION_REPORT_UPDATE",
			"TIMELINE_READ", "TIMELINE_CREATE", "TIMELINE_UPDATE",
			"MARKER_PLAN_READ", "MARKER_PLAN_CREATE", "MARKER_PLAN_UPDATE",
			"CUTTING_PLAN_READ", "CUTTING_PLAN_CREATE", "CUTTING_PLAN_UPDATE",
			"DATA_APPROVE_CUTTING_PLAN_READ", "DATA_APPROVE_CUTTING_PLAN_CREATE",
			"PACKING_LIST_READ", "PACKING_LIST_CREATE", "PACKING_LIST_UPDATE",
			"SURAT_JALAN_INTERNAL_READ", "SURAT_JALAN_INTERNAL_CREATE", "SURAT_JALAN_INTERNAL_UPDATE", "SURAT_JALAN_INTERNAL_DELETE",
			"REKONSILIASI_READ", "REKONSILIASI_CREATE", "REKONSILIASI_UPDATE",
			"REPORT_READ", "DASHBOARD_READ", "AI_ESTIMATION_READ",
		},
		"ADMIN_GUDANG": {
			"AUTH_CHANGE_PASSWORD", "PASSWORD_RESET_REQUEST_CREATE",
			"MASTER_MITRA_READ", "MASTER_BARANG_READ", "MASTER_JENIS_BARANG_READ", "MASTER_WARNA_READ", "MASTER_SIZE_READ", "MASTER_DEPARTEMEN_READ", "MASTER_PROFIL_PERUSAHAAN_READ",
			"PR_INTERNAL_READ", "PR_INTERNAL_CREATE", "PR_INTERNAL_UPDATE",
			"INVENTORY_RECEIVE", "INVENTORY_ISSUE",
			"SURAT_JALAN_CLIENT_READ", "SURAT_JALAN_CLIENT_CREATE", "SURAT_JALAN_CLIENT_UPDATE", "SURAT_JALAN_CLIENT_DELETE",
			"SURAT_JALAN_INTERNAL_READ",
			"REKONSILIASI_READ",
			"REPORT_READ", "DASHBOARD_READ",
		},
		"MANAGER": {
			"AUTH_CHANGE_PASSWORD", "PASSWORD_RESET_REQUEST_CREATE",
			"USER_READ", "USER_APPROVE",
			"MASTER_DEPARTEMEN_READ", "MASTER_DEPARTEMEN_CREATE", "MASTER_DEPARTEMEN_UPDATE", "MASTER_DEPARTEMEN_DELETE",
			"MASTER_MITRA_READ", "MASTER_BARANG_READ", "MASTER_JENIS_BARANG_READ", "MASTER_WARNA_READ", "MASTER_SIZE_READ", "MASTER_PROFIL_PERUSAHAAN_READ",
			"PO_CLIENT_READ", "PR_INTERNAL_READ", "PO_INTERNAL_READ",
			"WO_READ", "PRODUCTION_SUMMARY_READ", "PRODUCTION_REPORT_READ",
			"TIMELINE_READ", "MARKER_PLAN_READ", "CUTTING_PLAN_READ",
			"DATA_APPROVE_CUTTING_PLAN_READ", "DATA_APPROVE_CUTTING_PLAN_CREATE",
			"PACKING_LIST_READ", "SURAT_JALAN_CLIENT_READ", "SURAT_JALAN_INTERNAL_READ", "REKONSILIASI_READ",
			"REPORT_READ", "LOG_READ", "DASHBOARD_READ", "AI_ESTIMATION_READ",
			"PR_INTERNAL_APPROVE", "PO_INTERNAL_APPROVE", "PACKING_LIST_APPROVE",
		},
		"CLIENT": {
			"AUTH_CHANGE_PASSWORD", "PASSWORD_RESET_REQUEST_CREATE",
			"PO_CLIENT_READ", "WO_READ", "PRODUCTION_SUMMARY_READ", "PACKING_LIST_READ", "SURAT_JALAN_CLIENT_READ",
			"DASHBOARD_READ",
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

func seedSystemUsers(ctx context.Context, db *pgxpool.Pool) error {
	if err := seedSystemUser(ctx, db, seedSystemUserParams{
		Username:                 "super-admin",
		Password:                 "admin123",
		RoleName:                 "SUPER_ADMIN",
		DepartmentName:           "IT",
		MustChangePassword:       false,
		PreservePasswordOnUpdate: false,
	}); err != nil {
		return err
	}

	if err := seedSystemUser(ctx, db, seedSystemUserParams{
		Username:                 "admin-sistem",
		Password:                 "admin123",
		RoleName:                 "ADMIN_SISTEM",
		DepartmentName:           "IT",
		MustChangePassword:       false,
		PreservePasswordOnUpdate: false,
	}); err != nil {
		return err
	}

	if err := seedSystemUser(ctx, db, seedSystemUserParams{
		Username:                 "admin-keuangan",
		Password:                 "admin123",
		RoleName:                 "ADMIN_KEUANGAN",
		DepartmentName:           "OFFICE",
		MustChangePassword:       false,
		PreservePasswordOnUpdate: false,
	}); err != nil {
		return err
	}

	if err := seedSystemUser(ctx, db, seedSystemUserParams{
		Username:                 "admin-produksi",
		Password:                 "admin123",
		RoleName:                 "ADMIN_PRODUKSI",
		DepartmentName:           "PRODUKSI",
		MustChangePassword:       false,
		PreservePasswordOnUpdate: false,
	}); err != nil {
		return err
	}

	if err := seedSystemUser(ctx, db, seedSystemUserParams{
		Username:                 "admin-gudang",
		Password:                 "admin123",
		RoleName:                 "ADMIN_GUDANG",
		DepartmentName:           "GUDANG",
		MustChangePassword:       false,
		PreservePasswordOnUpdate: false,
	}); err != nil {
		return err
	}

	if err := seedSystemUser(ctx, db, seedSystemUserParams{
		Username:                 "manager",
		Password:                 "admin123",
		RoleName:                 "MANAGER",
		DepartmentName:           "OFFICE",
		MustChangePassword:       false,
		PreservePasswordOnUpdate: false,
	}); err != nil {
		return err
	}

	clientMitraID := int32(4)
	if err := seedSystemUser(ctx, db, seedSystemUserParams{
		Username:                 "client",
		Password:                 "admin123",
		RoleName:                 "CLIENT",
		DepartmentName:           "",
		MustChangePassword:       false,
		PreservePasswordOnUpdate: false,
		IDMitra:                  &clientMitraID,
	}); err != nil {
		return err
	}

	return nil
}

func seedTimelinePlan(ctx context.Context, db *pgxpool.Pool) error {
	var exists bool
	err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM TIMELINE_PLAN_PRODUKSI WHERE NOTES = 'Initial Master Timeline')`).Scan(&exists)
	if err != nil {
		return err
	}

	if !exists {
		// Dapatkan PO Client
		var idPoClient int32
		err = db.QueryRow(ctx, `SELECT ID_PO_CLIENT FROM PO_CLIENT WHERE PO_NUMBER = 'PO Test' LIMIT 1`).Scan(&idPoClient)
		if err != nil {
			return fmt.Errorf("failed to get PO Client for timeline seeder: %w", err)
		}

		// Insert Timeline Plan Produksi
		var idTimeline int32
		err = db.QueryRow(ctx, `
			INSERT INTO TIMELINE_PLAN_PRODUKSI (ID_PO_CLIENT, TANGGAL_DISUSUN, NOTES)
			VALUES ($1, '2026-06-07', 'Initial Master Timeline')
			RETURNING ID_TIMELINE
		`, idPoClient).Scan(&idTimeline)
		if err != nil {
			return err
		}
		slog.Info("timeline plan seeded", slog.Int("id", int(idTimeline)))

		// Dapatkan WO Shell
		var idWoShell int32
		err = db.QueryRow(ctx, `
			SELECT s.ID_WO_SHELL 
			FROM WORK_ORDER_SHELL s 
			JOIN WORK_ORDER w ON s.ID_WO = w.ID_WO 
			WHERE w.MODEL = 'WO Test 1' AND s.COLOR = 'NAVY' LIMIT 1
		`).Scan(&idWoShell)
		if err != nil {
			return fmt.Errorf("failed to get WO Shell for timeline seeder: %w", err)
		}

		// Insert WO Shell Plan
		_, err = db.Exec(ctx, `
			INSERT INTO WO_SHELL_PLAN (
				ID_TIMELINE, ID_WO_SHELL, IN_LINE, 
				TGL_GELAR_CUTTING, STATUS_GELAR_CUTTING,
				TGL_EMBROO, STATUS_EMBROO,
				TGL_LOADING_SEWING, STATUS_LOADING_SEWING,
				TGL_FINISHING_PACKING, STATUS_FINISHING_PACKING
			) VALUES (
				$1, $2, 'Line 1',
				'2026-06-10', 'PENDING',
				'2026-06-12', 'PENDING',
				'2026-06-15', 'PENDING',
				'2026-06-20', 'PENDING'
			)
		`, idTimeline, idWoShell)
		if err != nil {
			return err
		}
		slog.Info("wo shell plan seeded for timeline", slog.Int("id_timeline", int(idTimeline)))
	}

	return nil
}

type seedSystemUserParams struct {
	Username                 string
	Password                 string
	RoleName                 string
	DepartmentName           string
	MustChangePassword       bool
	PreservePasswordOnUpdate bool
	IDMitra                  *int32
}

func seedSystemUser(ctx context.Context, db *pgxpool.Pool, params seedSystemUserParams) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(params.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	var roleID int32
	err = db.QueryRow(ctx, `SELECT ID_ROLE FROM ROLES WHERE NAMA_ROLE = $1 LIMIT 1`, params.RoleName).Scan(&roleID)
	if err != nil {
		return fmt.Errorf("role %s not found: %w", params.RoleName, err)
	}

	var deptID *int32
	if params.DepartmentName != "" {
		var id int32
		err = db.QueryRow(ctx, `SELECT ID_DEPARTEMEN FROM DEPARTEMEN WHERE NAMA_DEPARTEMEN = $1 LIMIT 1`, params.DepartmentName).Scan(&id)
		if err == nil {
			deptID = &id
		}
	}

	var exists bool
	err = db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM USERS WHERE username = $1)`, params.Username).Scan(&exists)
	if err != nil {
		return err
	}

	if !exists {
		_, err = db.Exec(ctx, `
			INSERT INTO USERS (USERNAME, PASSWORD, ID_ROLE, ID_DEPARTEMEN, ID_MITRA, STATUS, MUST_CHANGE_PASSWORD)
			VALUES ($1, $2, $3, $4, $5, 'active', $6)
		`, params.Username, string(hashedPassword), roleID, deptID, params.IDMitra, params.MustChangePassword)
		if err != nil {
			return err
		}
		slog.Info("user seeded", slog.String("username", params.Username))
		return nil
	}

	if params.PreservePasswordOnUpdate {
		_, err = db.Exec(ctx, `
			UPDATE USERS
			SET ID_ROLE = $2,
				ID_DEPARTEMEN = COALESCE($3, ID_DEPARTEMEN),
				ID_MITRA = COALESCE($4, ID_MITRA),
				STATUS = 'active',
				MUST_CHANGE_PASSWORD = $5
			WHERE USERNAME = $1
		`, params.Username, roleID, deptID, params.IDMitra, params.MustChangePassword)
		return err
	}

	_, err = db.Exec(ctx, `
		UPDATE USERS
		SET PASSWORD = $2,
			ID_ROLE = $3,
			ID_DEPARTEMEN = $4,
			ID_MITRA = $5,
			STATUS = 'active',
			MUST_CHANGE_PASSWORD = $6
		WHERE USERNAME = $1
	`, params.Username, string(hashedPassword), roleID, deptID, params.IDMitra, params.MustChangePassword)
	return err
}

func seedPOClient(ctx context.Context, db *pgxpool.Pool) error {
	var exists bool
	err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM PO_CLIENT WHERE PO_NUMBER = 'PO Test')`).Scan(&exists)
	if err != nil {
		return err
	}

	if !exists {
		// Insert PO Client
		var idPoClient int32
		err = db.QueryRow(ctx, `
			INSERT INTO PO_CLIENT (PO_NUMBER, TANGGAL, SEASON, DELIVERY, PAYMENT_TERM, FILE, ID_MITRA)
			VALUES ('PO Test', '2026-06-01', 'Summer 2026', '2026-08-30', 'Net 30', 'po_test_file.pdf', 4)
			RETURNING ID_PO_CLIENT
		`).Scan(&idPoClient)
		if err != nil {
			return err
		}
		slog.Info("po client seeded: PO Test", slog.Int("id", int(idPoClient)))

		// Insert PO Client Items
		_, err = db.Exec(ctx, `
			INSERT INTO PO_CLIENT_ITEM (ID_PO_CLIENT, STYLE, COLOUR, DESCRIPTION, QTY, PRICE)
			VALUES 
			($1, 'PO Test 1', 'NAVY', 'PO Test 1 Item Description', 1000, 10.00),
			($1, 'PO Test 2', 'MAROON', 'PO Test 2 Item Description', 1000, 12.00)
		`, idPoClient)
		if err != nil {
			return err
		}
		slog.Info("po client items seeded for PO Test")
	}

	// Seed another PO without Work Orders for testing
	var existsPending bool
	err = db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM PO_CLIENT WHERE PO_NUMBER = 'PO Test Pending')`).Scan(&existsPending)
	if err != nil {
		return err
	}

	if !existsPending {
		var idPoClientPending int32
		err = db.QueryRow(ctx, `
			INSERT INTO PO_CLIENT (PO_NUMBER, TANGGAL, SEASON, DELIVERY, PAYMENT_TERM, FILE, ID_MITRA)
			VALUES ('PO Test Pending', '2026-06-01', 'Summer 2026', '2026-08-30', 'Net 30', 'po_test_pending_file.pdf', 4)
			RETURNING ID_PO_CLIENT
		`).Scan(&idPoClientPending)
		if err != nil {
			return err
		}
		slog.Info("po client seeded: PO Test Pending", slog.Int("id", int(idPoClientPending)))

		_, err = db.Exec(ctx, `
			INSERT INTO PO_CLIENT_ITEM (ID_PO_CLIENT, STYLE, COLOUR, DESCRIPTION, QTY, PRICE)
			VALUES 
			($1, 'PO Pending Item 1', 'BLACK', 'PO Pending 1 Item Description', 800, 15.00),
			($1, 'PO Pending Item 2', 'WHITE', 'PO Pending 2 Item Description', 1200, 14.50)
		`, idPoClientPending)
		if err != nil {
			return err
		}
		slog.Info("po client items seeded for PO Test Pending")
	}

	return nil
}

func seedWorkOrder(ctx context.Context, db *pgxpool.Pool) error {
	var exists bool
	err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM WORK_ORDER WHERE MODEL IN ('WO Test 1', 'WO Test 2'))`).Scan(&exists)
	if err != nil {
		return err
	}

	if !exists {
		var idItem1, idItem2 int32
		err = db.QueryRow(ctx, `SELECT ID_PO_CLIENT_ITEM FROM PO_CLIENT_ITEM WHERE STYLE = 'PO Test 1' LIMIT 1`).Scan(&idItem1)
		if err != nil {
			return fmt.Errorf("failed to find PO Item PO Test 1: %w", err)
		}
		err = db.QueryRow(ctx, `SELECT ID_PO_CLIENT_ITEM FROM PO_CLIENT_ITEM WHERE STYLE = 'PO Test 2' LIMIT 1`).Scan(&idItem2)
		if err != nil {
			return fmt.Errorf("failed to find PO Item PO Test 2: %w", err)
		}

		// WO 1: WO Test 1
		var idWo1 int32
		err = db.QueryRow(ctx, `
			INSERT INTO WORK_ORDER (BUYER, MODEL, QTY, FOB_CMT, DELIVERY, ID_PO_CLIENT_ITEM)
			VALUES ('Garment Client Global', 'WO Test 1', 1000, true, '2026-08-30', $1)
			RETURNING ID_WO
		`, idItem1).Scan(&idWo1)
		if err != nil {
			return err
		}
		slog.Info("work order 1 seeded: WO Test 1", slog.Int("id", int(idWo1)))

		// WO 1 Shells:
		// Shell 1: Fabric (Cotton Fleece, Navy)
		var idShell1 int32
		err = db.QueryRow(ctx, `
			INSERT INTO WORK_ORDER_SHELL (DESKRIPSI, CONS, COLOR, ALLOW, BERAT_1_YD, ID_WO, PROVIDED_BY, MATERIAL_TYPE)
			VALUES ('Cotton Fleece', 0.35, 'NAVY', 3, 0.22, $1, 'permata', 'fabric')
			RETURNING ID_WO_SHELL
		`, idWo1).Scan(&idShell1)
		if err != nil {
			return err
		}

		// Shell 2: Fabric (Cotton Fleece, Maroon)
		var idShell2 int32
		err = db.QueryRow(ctx, `
			INSERT INTO WORK_ORDER_SHELL (DESKRIPSI, CONS, COLOR, ALLOW, BERAT_1_YD, ID_WO, PROVIDED_BY, MATERIAL_TYPE)
			VALUES ('Cotton Fleece', 0.35, 'MAROON', 3, 0.22, $1, 'permata', 'fabric')
			RETURNING ID_WO_SHELL
		`, idWo1).Scan(&idShell2)
		if err != nil {
			return err
		}

		// Shell 3: Interlining (2016F, White)
		var idShell3 int32
		err = db.QueryRow(ctx, `
			INSERT INTO WORK_ORDER_SHELL (DESKRIPSI, CONS, COLOR, ALLOW, BERAT_1_YD, ID_WO, PROVIDED_BY, MATERIAL_TYPE)
			VALUES ('2016F', 0.1, 'WHITE', 3, 0.05, $1, 'permata', 'interlining')
			RETURNING ID_WO_SHELL
		`, idWo1).Scan(&idShell3)
		if err != nil {
			return err
		}

		// Sizes for shells
		sizes := []struct {
			size  string
			qty   int32
			ratio int32
		}{
			{"XS", 100, 1},
			{"S", 150, 2},
			{"M", 250, 3},
			{"L", 250, 3},
			{"XL", 150, 2},
			{"XXL", 100, 1},
		}

		for _, sz := range sizes {
			_, err = db.Exec(ctx, `
				INSERT INTO WORK_ORDER_SHELL_SIZE (ID_SIZE, QTY, RATIO, ID_WO_SHELL)
				VALUES (
					(SELECT ID_SIZE FROM MASTER_SIZE WHERE LOWER(BTRIM(NAMA_SIZE)) = LOWER(BTRIM($1)) LIMIT 1),
					$2, $3, $4
				), (
					(SELECT ID_SIZE FROM MASTER_SIZE WHERE LOWER(BTRIM(NAMA_SIZE)) = LOWER(BTRIM($1)) LIMIT 1),
					$2, $3, $5
				), (
					(SELECT ID_SIZE FROM MASTER_SIZE WHERE LOWER(BTRIM(NAMA_SIZE)) = LOWER(BTRIM($1)) LIMIT 1),
					$2, $3, $6
				)
			`, sz.size, sz.qty, sz.ratio, idShell1, idShell2, idShell3)
			if err != nil {
				return err
			}
		}

		// Trims for WO 1
		// 1. Thread Navy
		var idTrimThreadNavy int32
		err = db.QueryRow(ctx, `
			INSERT INTO WORK_ORDER_TRIM (ITEM, DESCRIPTION, COLOR, CODE, CONS, QTY, UOM, POSITION, CREATED_BY, ALLOW, ID_WO, PROVIDED_BY)
			VALUES ('Thread', 'Sewing Thread Navy', 'NAVY', 'TRM-THR-NAVY', 0.05, 50, 'cones', 'Seam', 'super-admin', 0, $1, 'permata')
			RETURNING ID_WO_TRIM
		`, idWo1).Scan(&idTrimThreadNavy)
		if err != nil {
			return err
		}

		// 2. Thread Maroon
		var idTrimThreadMaroon int32
		err = db.QueryRow(ctx, `
			INSERT INTO WORK_ORDER_TRIM (ITEM, DESCRIPTION, COLOR, CODE, CONS, QTY, UOM, POSITION, CREATED_BY, ALLOW, ID_WO, PROVIDED_BY)
			VALUES ('Thread', 'Sewing Thread Maroon', 'MAROON', 'TRM-THR-MAROON', 0.05, 50, 'cones', 'Seam', 'super-admin', 0, $1, 'permata')
			RETURNING ID_WO_TRIM
		`, idWo1).Scan(&idTrimThreadMaroon)
		if err != nil {
			return err
		}

		// 3. Main Size Labels for each size (XS, S, M, L, XL, XXL)
		var labelTrimIDs []int32
		for _, sz := range sizes {
			var idLabel int32
			err = db.QueryRow(ctx, `
				INSERT INTO WORK_ORDER_TRIM (ITEM, DESCRIPTION, COLOR, CODE, CONS, QTY, UOM, POSITION, CREATED_BY, ALLOW, ID_WO, PROVIDED_BY)
				VALUES ('Main Size Label', 'Main Size Label ' || $1, 'WHITE', 'TRM-LBL-' || $1, 1.0, $2 + 3, 'pcs', 'Neck', 'super-admin', 3, $3, 'permata')
				RETURNING ID_WO_TRIM
			`, sz.size, sz.qty, idWo1).Scan(&idLabel)
			if err != nil {
				return err
			}
			labelTrimIDs = append(labelTrimIDs, idLabel)
		}

		// 4. Buttons Navy
		var idTrimButtonNavy int32
		err = db.QueryRow(ctx, `
			INSERT INTO WORK_ORDER_TRIM (ITEM, DESCRIPTION, COLOR, CODE, CONS, QTY, UOM, POSITION, CREATED_BY, ALLOW, ID_WO, PROVIDED_BY)
			VALUES ('Kancing', 'Button Navy 24L', 'NAVY', 'TRM-BTN-NAVY', 10.0, 10300, 'pcs', 'Front', 'super-admin', 3, $1, 'permata')
			RETURNING ID_WO_TRIM
		`, idWo1).Scan(&idTrimButtonNavy)
		if err != nil {
			return err
		}

		// 5. Buttons Maroon
		var idTrimButtonMaroon int32
		err = db.QueryRow(ctx, `
			INSERT INTO WORK_ORDER_TRIM (ITEM, DESCRIPTION, COLOR, CODE, CONS, QTY, UOM, POSITION, CREATED_BY, ALLOW, ID_WO, PROVIDED_BY)
			VALUES ('Kancing', 'Button Maroon 24L', 'MAROON', 'TRM-BTN-MAROON', 10.0, 10300, 'pcs', 'Front', 'super-admin', 3, $1, 'permata')
			RETURNING ID_WO_TRIM
		`, idWo1).Scan(&idTrimButtonMaroon)
		if err != nil {
			return err
		}

		// Material List utama untuk WO 1 (1 grouping container + N items).
		var idMLWo1 int32
		err = db.QueryRow(ctx, `
			INSERT INTO MATERIAL_LIST (ID_WO, NAME)
			VALUES ($1, 'Material List Utama')
			RETURNING ID_MATERIAL_LIST
		`, idWo1).Scan(&idMLWo1)
		if err != nil {
			return err
		}

		// Shells Material List Items
		for i, idShell := range []int32{idShell1, idShell2, idShell3} {
			desc := fmt.Sprintf("WO Test 1 Shell %d Fabric", i+1)
			var idMLI int32
			err = db.QueryRow(ctx, `
				INSERT INTO MATERIAL_LIST_ITEM (ID_MATERIAL_LIST, ITEM, DESCRIPTION, QTY, UNIT, EST_PRICE, ID_WO_SHELL, ID_WO_TRIM)
				VALUES ($1, $2, $3, 0, 'yds', 0, $4, NULL)
				RETURNING ID_MATERIAL_LIST_ITEM
			`, idMLWo1, desc, desc, idShell).Scan(&idMLI)
			if err != nil {
				return err
			}

			// Seed RECEIVED quantity only for the first shell (Navy Fabric) to prevent migrate-down errors
			if i == 0 {
				_, err = db.Exec(ctx, `
					INSERT INTO RECEIVED (TANGGAL, QTY, KETERANGAN, ID_MATERIAL_LIST_ITEM)
					VALUES ('2026-06-05', 500, 'Penerimaan Awal', $1)
				`, idMLI)
				if err != nil {
					return err
				}
			}
		}

		// Trims Material List Items
		trimIDs := append([]int32{idTrimThreadNavy, idTrimThreadMaroon}, labelTrimIDs...)
		trimIDs = append(trimIDs, idTrimButtonNavy, idTrimButtonMaroon)

		for i, idTrim := range trimIDs {
			desc := fmt.Sprintf("WO Test 1 Trim %d", i+1)
			_, err = db.Exec(ctx, `
				INSERT INTO MATERIAL_LIST_ITEM (ID_MATERIAL_LIST, ITEM, DESCRIPTION, QTY, UNIT, EST_PRICE, ID_WO_SHELL, ID_WO_TRIM)
				VALUES ($1, $2, $3, 0, 'pcs', 0, NULL, $4)
			`, idMLWo1, desc, desc, idTrim)
			if err != nil {
				return err
			}
		}

		// WO 2: WO Test 2
		var idWo2 int32
		err = db.QueryRow(ctx, `
			INSERT INTO WORK_ORDER (BUYER, MODEL, QTY, FOB_CMT, DELIVERY, ID_PO_CLIENT_ITEM)
			VALUES ('Garment Client Global', 'WO Test 2', 1000, true, '2026-08-30', $1)
			RETURNING ID_WO
		`, idItem2).Scan(&idWo2)
		if err != nil {
			return err
		}
		slog.Info("work order 2 seeded: WO Test 2", slog.Int("id", int(idWo2)))

		// WO 2 Details: Shell
		var idShellWO2 int32
		err = db.QueryRow(ctx, `
			INSERT INTO WORK_ORDER_SHELL (DESKRIPSI, CONS, COLOR, ALLOW, BERAT_1_YD, ID_WO, PROVIDED_BY, MATERIAL_TYPE)
			VALUES ('Cotton Combed 30s', 0.35, 'MAROON', 3, 0.22, $1, 'permata', 'fabric')
			RETURNING ID_WO_SHELL
		`, idWo2).Scan(&idShellWO2)
		if err != nil {
			return err
		}

		// WO 2 Details: Sizes
		for _, sz := range sizes {
			_, err = db.Exec(ctx, `
				INSERT INTO WORK_ORDER_SHELL_SIZE (ID_SIZE, QTY, RATIO, ID_WO_SHELL)
				VALUES (
					(SELECT ID_SIZE FROM MASTER_SIZE WHERE LOWER(BTRIM(NAMA_SIZE)) = LOWER(BTRIM($1)) LIMIT 1),
					$2, $3, $4
				)
			`, sz.size, sz.qty, sz.ratio, idShellWO2)
			if err != nil {
				return err
			}
		}

		// WO 2 Details: Material List utama + item.
		var idMLWo2 int32
		err = db.QueryRow(ctx, `
			INSERT INTO MATERIAL_LIST (ID_WO, NAME)
			VALUES ($1, 'Material List Utama')
			RETURNING ID_MATERIAL_LIST
		`, idWo2).Scan(&idMLWo2)
		if err != nil {
			return err
		}

		_, err = db.Exec(ctx, `
			INSERT INTO MATERIAL_LIST_ITEM (ID_MATERIAL_LIST, ITEM, DESCRIPTION, QTY, UNIT, EST_PRICE, ID_WO_SHELL, ID_WO_TRIM)
			VALUES ($1, 'WO Test 2 Fabric', 'WO Test 2 Fabric', 0, 'yds', 0, $2, NULL)
		`, idMLWo2, idShellWO2)
		if err != nil {
			return err
		}
	}

	return nil
}

func seedProductionReports(ctx context.Context, db *pgxpool.Pool) error {
	var exists bool
	err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM REPORT_CUTTING)`).Scan(&exists)
	if err != nil {
		return err
	}

	if !exists {
		type ShellSize struct {
			ID   int32
			Size string
			Qty  int32
		}
		rows, err := db.Query(ctx, `
			SELECT woss.ID_WO_SHELL_SIZE, ms.NAMA_SIZE AS SIZE, woss.QTY
			FROM WORK_ORDER_SHELL_SIZE woss
			JOIN MASTER_SIZE ms ON ms.ID_SIZE = woss.ID_SIZE
			JOIN WORK_ORDER_SHELL wos ON wos.ID_WO_SHELL = woss.ID_WO_SHELL
			JOIN WORK_ORDER wo ON wo.ID_WO = wos.ID_WO
			WHERE wo.MODEL = 'WO Test 1' AND wos.COLOR = 'NAVY'
		`)
		if err != nil {
			return err
		}
		defer rows.Close()

		var sizes []ShellSize
		for rows.Next() {
			var sz ShellSize
			if err := rows.Scan(&sz.ID, &sz.Size, &sz.Qty); err != nil {
				return err
			}
			sizes = append(sizes, sz)
		}

		for _, sz := range sizes {
			switch sz.Size {
			case "XS":
				_, err = db.Exec(ctx, `INSERT INTO REPORT_CUTTING (TANGGAL, QTY, ID_WO_SHELL_SIZE) VALUES ('2026-06-01', 100, $1)`, sz.ID)
				if err != nil {
					return err
				}
				_, err = db.Exec(ctx, `INSERT INTO REPORT_SEWING (TANGGAL, QTY, ID_WO_SHELL_SIZE) VALUES ('2026-06-02', 80, $1)`, sz.ID)
				if err != nil {
					return err
				}
				_, err = db.Exec(ctx, `INSERT INTO REPORT_QC_FINISH (TANGGAL, QTY, ID_WO_SHELL_SIZE) VALUES ('2026-06-03', 70, $1)`, sz.ID)
				if err != nil {
					return err
				}
				_, err = db.Exec(ctx, `INSERT INTO REPORT_PACKING (TANGGAL, QTY, ID_WO_SHELL_SIZE) VALUES ('2026-06-03', 60, $1)`, sz.ID)
				if err != nil {
					return err
				}
				_, err = db.Exec(ctx, `INSERT INTO REPORT_PENGIRIMAN (REPORT_DATE, QTY, ID_WO_SHELL_SIZE) VALUES ('2026-06-03', 50, $1)`, sz.ID)
				if err != nil {
					return err
				}
			case "S":
				_, err = db.Exec(ctx, `INSERT INTO REPORT_CUTTING (TANGGAL, QTY, ID_WO_SHELL_SIZE) VALUES ('2026-06-01', 150, $1)`, sz.ID)
				if err != nil {
					return err
				}
				_, err = db.Exec(ctx, `INSERT INTO REPORT_SEWING (TANGGAL, QTY, ID_WO_SHELL_SIZE) VALUES ('2026-06-02', 130, $1)`, sz.ID)
				if err != nil {
					return err
				}
				_, err = db.Exec(ctx, `INSERT INTO REPORT_QC_FINISH (TANGGAL, QTY, ID_WO_SHELL_SIZE) VALUES ('2026-06-03', 120, $1)`, sz.ID)
				if err != nil {
					return err
				}
				_, err = db.Exec(ctx, `INSERT INTO REPORT_PACKING (TANGGAL, QTY, ID_WO_SHELL_SIZE) VALUES ('2026-06-03', 110, $1)`, sz.ID)
				if err != nil {
					return err
				}
				_, err = db.Exec(ctx, `INSERT INTO REPORT_PENGIRIMAN (REPORT_DATE, QTY, ID_WO_SHELL_SIZE) VALUES ('2026-06-03', 100, $1)`, sz.ID)
				if err != nil {
					return err
				}
			case "M":
				_, err = db.Exec(ctx, `INSERT INTO REPORT_CUTTING (TANGGAL, QTY, ID_WO_SHELL_SIZE) VALUES ('2026-06-01', 250, $1)`, sz.ID)
				if err != nil {
					return err
				}
				_, err = db.Exec(ctx, `INSERT INTO REPORT_SEWING (TANGGAL, QTY, ID_WO_SHELL_SIZE) VALUES ('2026-06-02', 220, $1)`, sz.ID)
				if err != nil {
					return err
				}
				_, err = db.Exec(ctx, `INSERT INTO REPORT_QC_FINISH (TANGGAL, QTY, ID_WO_SHELL_SIZE) VALUES ('2026-06-03', 200, $1)`, sz.ID)
				if err != nil {
					return err
				}
				_, err = db.Exec(ctx, `INSERT INTO REPORT_PACKING (TANGGAL, QTY, ID_WO_SHELL_SIZE) VALUES ('2026-06-03', 180, $1)`, sz.ID)
				if err != nil {
					return err
				}
				_, err = db.Exec(ctx, `INSERT INTO REPORT_PENGIRIMAN (REPORT_DATE, QTY, ID_WO_SHELL_SIZE) VALUES ('2026-06-03', 150, $1)`, sz.ID)
				if err != nil {
					return err
				}
			case "L":
				_, err = db.Exec(ctx, `INSERT INTO REPORT_CUTTING (TANGGAL, QTY, ID_WO_SHELL_SIZE) VALUES ('2026-06-02', 250, $1)`, sz.ID)
				if err != nil {
					return err
				}
				_, err = db.Exec(ctx, `INSERT INTO REPORT_SEWING (TANGGAL, QTY, ID_WO_SHELL_SIZE) VALUES ('2026-06-03', 200, $1)`, sz.ID)
				if err != nil {
					return err
				}
			case "XL":
				_, err = db.Exec(ctx, `INSERT INTO REPORT_CUTTING (TANGGAL, QTY, ID_WO_SHELL_SIZE) VALUES ('2026-06-03', 150, $1)`, sz.ID)
				if err != nil {
					return err
				}
			case "XXL":
				_, err = db.Exec(ctx, `INSERT INTO REPORT_CUTTING (TANGGAL, QTY, ID_WO_SHELL_SIZE) VALUES ('2026-06-03', 100, $1)`, sz.ID)
				if err != nil {
					return err
				}
			}
		}
		slog.Info("production reports seeded successfully for WO Test 1")
	}

	return nil
}

func seedMarkerPlan(ctx context.Context, db *pgxpool.Pool) error {
	_, err := db.Exec(ctx, `DELETE FROM MARKER_PLAN WHERE NO_DOKUMEN = 'MP-2026-001'`)
	if err != nil {
		return fmt.Errorf("failed to clean up existing marker plan seed: %w", err)
	}

	// 1. Get Shell ID of WO 1 (Navy Fabric)
	var idShell int32
	err = db.QueryRow(ctx, `
			SELECT wos.ID_WO_SHELL 
			FROM WORK_ORDER_SHELL wos
			JOIN WORK_ORDER wo ON wo.ID_WO = wos.ID_WO
			WHERE wo.MODEL = 'WO Test 1' AND wos.COLOR = 'NAVY' LIMIT 1
		`).Scan(&idShell)
	if err != nil {
		return fmt.Errorf("failed to find WO Shell for Navy Fabric WO Test 1: %w", err)
	}

	// Get Shell ID of WO 1 (White Interlining)
	var idShellInterlining int32
	err = db.QueryRow(ctx, `
			SELECT wos.ID_WO_SHELL 
			FROM WORK_ORDER_SHELL wos
			JOIN WORK_ORDER wo ON wo.ID_WO = wos.ID_WO
			WHERE wo.MODEL = 'WO Test 1' AND wos.MATERIAL_TYPE = 'interlining' LIMIT 1
		`).Scan(&idShellInterlining)
	if err != nil {
		return fmt.Errorf("failed to find WO Shell for Interlining WO Test 1: %w", err)
	}

	// 2. Insert Marker Plan
	var idMarkerPlan int32
	err = db.QueryRow(ctx, `
			INSERT INTO MARKER_PLAN (NO_DOKUMEN, TANGGAL_EFEKTIF, ID_WO_SHELL)
			VALUES ('MP-2026-001', '2026-06-02', $1)
			RETURNING ID_MARKER_PLAN
		`, idShell).Scan(&idMarkerPlan)
	if err != nil {
		return err
	}
	slog.Info("marker plan seeded: MP-2026-001", slog.Int("id", int(idMarkerPlan)))

	// 3. Get Shell size IDs of WO 1 (Navy Fabric: XS to XXL)
	rows, err := db.Query(ctx, `
			SELECT woss.ID_WO_SHELL_SIZE, ms.NAMA_SIZE AS SIZE
			FROM WORK_ORDER_SHELL_SIZE woss
			JOIN MASTER_SIZE ms ON ms.ID_SIZE = woss.ID_SIZE
			WHERE woss.ID_WO_SHELL = $1
		`, idShell)
	if err != nil {
		return err
	}

	type SizeInfo struct {
		ID   int32
		Size string
	}
	var sizes []SizeInfo
	for rows.Next() {
		var sz SizeInfo
		if err := rows.Scan(&sz.ID, &sz.Size); err != nil {
			rows.Close()
			return err
		}
		sizes = append(sizes, sz)
	}
	rows.Close()

	// Get Shell size IDs of WO 1 (White Interlining: XS to XXL)
	rowsInterlining, err := db.Query(ctx, `
			SELECT woss.ID_WO_SHELL_SIZE, ms.NAMA_SIZE AS SIZE
			FROM WORK_ORDER_SHELL_SIZE woss
			JOIN MASTER_SIZE ms ON ms.ID_SIZE = woss.ID_SIZE
			WHERE woss.ID_WO_SHELL = $1
		`, idShellInterlining)
	if err != nil {
		return err
	}

	var sizesInterlining []SizeInfo
	for rowsInterlining.Next() {
		var sz SizeInfo
		if err := rowsInterlining.Scan(&sz.ID, &sz.Size); err != nil {
			rowsInterlining.Close()
			return err
		}
		sizesInterlining = append(sizesInterlining, sz)
	}
	rowsInterlining.Close()

	// 4. Insert Komponen 1: Cotton Fleece Navy
	var idKomponen1 int32
	err = db.QueryRow(ctx, `
			INSERT INTO KOMPONEN_MARKER_PLAN (ID_MARKER_PLAN, NAMA_KOMPONEN)
			VALUES ($1, 'Cotton Fleece Navy')
			RETURNING ID_KOMPONEN_MARKER
		`, idMarkerPlan).Scan(&idKomponen1)
	if err != nil {
		return err
	}

	// Ratio 1 (Cut Pertama) for Komponen 1
	var idRatio1Komponen1 int32
	err = db.QueryRow(ctx, `
			INSERT INTO RATIO_MARKER (
				ID_KOMPONEN_MARKER, ID_WO_SHELL, CONS, PLAN_SPREADING_GELARAN,
				PANJANG_MARKER, EFFICIENCY_MARKER, ALLOWANCE, CONS_BUYER,
				PLOT, LEBAR_KAIN, PANJANG_MARKER_UNIT, KET
			) VALUES ($1, $2, 0.350, 50, 17.500, 85.50, 3.00, 0.360, 1, 1.450, 'yard', 'Cut Pertama')
			RETURNING ID_RATIO_MARKER
		`, idKomponen1, idShell).Scan(&idRatio1Komponen1)
	if err != nil {
		return err
	}

	for _, sz := range sizes {
		var ratioPlan int32
		switch sz.Size {
		case "XS":
			ratioPlan = 1
		case "S":
			ratioPlan = 2
		case "M":
			ratioPlan = 3
		case "L":
			ratioPlan = 3
		case "XL":
			ratioPlan = 2
		case "XXL":
			ratioPlan = 1
		default:
			ratioPlan = 1
		}
		_, err = db.Exec(ctx, `
				INSERT INTO RATIO_SIZE_MARKER (ID_RATIO_MARKER, ID_WO_SHELL_SIZE, RATIO_PLAN)
				VALUES ($1, $2, $3)
			`, idRatio1Komponen1, sz.ID, ratioPlan)
		if err != nil {
			return err
		}
	}

	// Ratio 2 (Cut Sisa) for Komponen 1
	var idRatio2Komponen1 int32
	err = db.QueryRow(ctx, `
			INSERT INTO RATIO_MARKER (
				ID_KOMPONEN_MARKER, ID_WO_SHELL, CONS, PLAN_SPREADING_GELARAN,
				PANJANG_MARKER, EFFICIENCY_MARKER, ALLOWANCE, CONS_BUYER,
				PLOT, LEBAR_KAIN, PANJANG_MARKER_UNIT, KET
			) VALUES ($1, $2, 0.350, 50, 17.500, 85.50, 3.00, 0.360, 2, 1.450, 'yard', 'Cut Sisa')
			RETURNING ID_RATIO_MARKER
		`, idKomponen1, idShell).Scan(&idRatio2Komponen1)
	if err != nil {
		return err
	}

	for _, sz := range sizes {
		var ratioPlan int32
		switch sz.Size {
		case "XS":
			ratioPlan = 1
		case "S":
			ratioPlan = 1
		case "M":
			ratioPlan = 2
		case "L":
			ratioPlan = 2
		case "XL":
			ratioPlan = 1
		case "XXL":
			ratioPlan = 1
		default:
			ratioPlan = 1
		}
		_, err = db.Exec(ctx, `
				INSERT INTO RATIO_SIZE_MARKER (ID_RATIO_MARKER, ID_WO_SHELL_SIZE, RATIO_PLAN)
				VALUES ($1, $2, $3)
			`, idRatio2Komponen1, sz.ID, ratioPlan)
		if err != nil {
			return err
		}
	}

	// 5. Insert Komponen 2: Interlining Plakat
	var idKomponen2 int32
	err = db.QueryRow(ctx, `
			INSERT INTO KOMPONEN_MARKER_PLAN (ID_MARKER_PLAN, NAMA_KOMPONEN)
			VALUES ($1, 'Interlining Plakat')
			RETURNING ID_KOMPONEN_MARKER
		`, idMarkerPlan).Scan(&idKomponen2)
	if err != nil {
		return err
	}

	// Ratio 1 for Komponen 2
	var idRatio1Komponen2 int32
	err = db.QueryRow(ctx, `
			INSERT INTO RATIO_MARKER (
				ID_KOMPONEN_MARKER, ID_WO_SHELL, CONS, PLAN_SPREADING_GELARAN,
				PANJANG_MARKER, EFFICIENCY_MARKER, ALLOWANCE, CONS_BUYER,
				PLOT, LEBAR_KAIN, PANJANG_MARKER_UNIT, KET
			) VALUES ($1, $2, 0.100, 50, 5.000, 90.00, 3.00, 0.110, 1, 1.100, 'yard', 'Interlining Plakat')
			RETURNING ID_RATIO_MARKER
		`, idKomponen2, idShellInterlining).Scan(&idRatio1Komponen2)
	if err != nil {
		return err
	}

	for _, sz := range sizesInterlining {
		var ratioPlan int32
		switch sz.Size {
		case "XS":
			ratioPlan = 2
		case "S":
			ratioPlan = 3
		case "M":
			ratioPlan = 5
		case "L":
			ratioPlan = 5
		case "XL":
			ratioPlan = 3
		case "XXL":
			ratioPlan = 2
		default:
			ratioPlan = 1
		}
		_, err = db.Exec(ctx, `
				INSERT INTO RATIO_SIZE_MARKER (ID_RATIO_MARKER, ID_WO_SHELL_SIZE, RATIO_PLAN)
				VALUES ($1, $2, $3)
			`, idRatio1Komponen2, sz.ID, ratioPlan)
		if err != nil {
			return err
		}
	}
	slog.Info("ratio size markers seeded for MP-2026-001")

	return nil
}

func seedProductionMaster(ctx context.Context, db *pgxpool.Pool) error {
	lines := []string{"Line 1", "Line 2", "Line 3"}
	for _, l := range lines {
		var exists bool
		err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM PRODUCTION_LINE WHERE NAME = $1)`, l).Scan(&exists)
		if err != nil {
			return err
		}
		if !exists {
			_, err = db.Exec(ctx, `INSERT INTO PRODUCTION_LINE (name) VALUES ($1)`, l)
			if err != nil {
				return err
			}
			slog.Info("production line seeded", slog.String("name", l))
		}
	}

	statuses := []string{"Done", "Plan Start", "Plan Finish", "Proses"}
	for _, s := range statuses {
		var exists bool
		err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM PRODUCTION_STATUS_PLAN WHERE NAME = $1)`, s).Scan(&exists)
		if err != nil {
			return err
		}
		if !exists {
			_, err = db.Exec(ctx, `INSERT INTO PRODUCTION_STATUS_PLAN (name) VALUES ($1)`, s)
			if err != nil {
				return err
			}
			slog.Info("production status plan seeded", slog.String("name", s))
		}
	}
	return nil
}

// seedPRInternal seeds PR_INTERNAL documents that are NOT yet approved, so the
// sequential approval flow can be exercised end-to-end from the frontend.
//
// Approval flow mirrors initializeApprovalWorkflow() for PR_INTERNAL:
//
//	Step 1 PEMBUAT   (creator)        -> done
//	Step 2 PENGECEK  (ADMIN_PRODUKSI) -> pending
//	Step 3 PENYETUJU (MANAGER)        -> pending
//	Step 4 RELEASE   (ADMIN_KEUANGAN) -> pending
//
// Header STATUS_GLOBAL stays 'pending' until all three roles approve in order.
func seedPRInternal(ctx context.Context, db *pgxpool.Pool) error {
	// Resolve FK dependencies.
	var creatorID int32
	err := db.QueryRow(ctx, `SELECT ID_USER FROM USERS WHERE username = 'super-admin' LIMIT 1`).Scan(&creatorID)
	if err != nil {
		return fmt.Errorf("failed to find creator user super-admin: %w", err)
	}

	var idWo int32
	err = db.QueryRow(ctx, `SELECT ID_WO FROM WORK_ORDER WHERE MODEL = 'WO Test 1' LIMIT 1`).Scan(&idWo)
	if err != nil {
		return fmt.Errorf("failed to find WO Test 1 for PR seeding: %w", err)
	}

	// Resolve the three approver user ids by role.
	roleUserID := func(roleName string) (int32, error) {
		var id int32
		qErr := db.QueryRow(ctx, `
			SELECT u.ID_USER
			FROM USERS u
			JOIN ROLES r ON u.ID_ROLE = r.ID_ROLE
			WHERE r.NAMA_ROLE = $1 AND u.status = 'active'
			ORDER BY u.ID_USER ASC
			LIMIT 1
		`, roleName).Scan(&id)
		return id, qErr
	}

	produksiID, err := roleUserID("ADMIN_PRODUKSI")
	if err != nil {
		return fmt.Errorf("failed to find ADMIN_PRODUKSI user: %w", err)
	}
	managerID, err := roleUserID("MANAGER")
	if err != nil {
		return fmt.Errorf("failed to find MANAGER user: %w", err)
	}
	keuanganID, err := roleUserID("ADMIN_KEUANGAN")
	if err != nil {
		return fmt.Errorf("failed to find ADMIN_KEUANGAN user: %w", err)
	}

	type prItem struct {
		item, description, unit string
		qty                     int32
		estPrice                float64
	}
	prs := []struct {
		nama, departemen, vendorName, vendorAddr, vendorTelp, projek string
		items                                                        []prItem
	}{
		{
			nama: "PR Test 1", departemen: "PRODUKSI",
			vendorName: "PT Sumber Kain Jaya", vendorAddr: "Jl. Industri No. 1, Bandung", vendorTelp: "022-1112233",
			projek: "PR Approval Test",
			items: []prItem{
				{item: "Cotton Fleece", description: "Fabric Navy 280gsm", unit: "yard", qty: 500, estPrice: 25000},
				{item: "Sewing Thread", description: "Thread Navy cones", unit: "cones", qty: 50, estPrice: 15000},
			},
		},
		{
			nama: "PR Test 2", departemen: "PRODUKSI",
			vendorName: "CV Benang Mas", vendorAddr: "Jl. Tekstil No. 9, Solo", vendorTelp: "0271-445566",
			projek: "PR Approval Test",
			items: []prItem{
				{item: "Zipper", description: "Metal Zipper 20cm", unit: "pcs", qty: 1000, estPrice: 3500},
			},
		},
		{
			nama: "PR Test 3", departemen: "GUDANG",
			vendorName: "PT Aksesoris Garmen", vendorAddr: "Jl. Pelabuhan No. 4, Semarang", vendorTelp: "024-778899",
			projek: "PR Approval Test",
			items: []prItem{
				{item: "Button", description: "Plastic Button 4-hole", unit: "gross", qty: 200, estPrice: 12000},
				{item: "Care Label", description: "Woven Care Label", unit: "pcs", qty: 2000, estPrice: 500},
			},
		},
	}

	for _, pr := range prs {
		var exists bool
		err = db.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM PR_INTERNAL WHERE NAMA = $1 AND PROJEK = $2)`,
			pr.nama, pr.projek,
		).Scan(&exists)
		if err != nil {
			return err
		}
		if exists {
			continue
		}

		// Insert PR header.
		var idPR int32
		err = db.QueryRow(ctx, `
			INSERT INTO PR_INTERNAL (TANGGAL, NAMA, DEPARTEMEN, VENDOR_NAME, VENDOR_ADDRESS, VENDOR_TELP, PROJEK, ID_WO, ID_USER)
			VALUES (CURRENT_DATE, $1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING ID_PR_INTERNAL
		`, pr.nama, pr.departemen, pr.vendorName, pr.vendorAddr, pr.vendorTelp, pr.projek, idWo, creatorID).Scan(&idPR)
		if err != nil {
			return err
		}

		// Insert PR items.
		for _, it := range pr.items {
			_, err = db.Exec(ctx, `
				INSERT INTO PR_INTERNAL_ITEM (ID_PR_INTERNAL, ITEM, DESCRIPTION, QTY, UNIT, EST_PRICE)
				VALUES ($1, $2, $3, $4, $5, $6)
			`, idPR, it.item, it.description, it.qty, it.unit, it.estPrice)
			if err != nil {
				return err
			}
		}

		// Approval header (pending).
		var idOtoritas int32
		err = db.QueryRow(ctx, `
			INSERT INTO OTORITAS_DOKUMEN (NAMA_TABEL_DOKUMEN, ID_DOKUMEN, STATUS_GLOBAL)
			VALUES ('PR_INTERNAL', $1, 'pending')
			RETURNING ID_OTORITAS
		`, idPR).Scan(&idOtoritas)
		if err != nil {
			return err
		}

		// Approval steps. Order of insertion defines the sequence.
		_, err = db.Exec(ctx, `
			INSERT INTO OTORITAS_DOKUMEN_DETAIL (ID_OTORITAS, ID_USER, TIPE_PERAN, IS_ACTION_DONE, WAKTU_AKSI, CATATAN)
			VALUES ($1, $2, 'PEMBUAT', TRUE, NOW(), 'Dokumen PR dibuat')
		`, idOtoritas, creatorID)
		if err != nil {
			return err
		}
		_, err = db.Exec(ctx, `
			INSERT INTO OTORITAS_DOKUMEN_DETAIL (ID_OTORITAS, ID_USER, TIPE_PERAN, IS_ACTION_DONE)
			VALUES ($1, $2, 'PENGECEK', FALSE)
		`, idOtoritas, produksiID)
		if err != nil {
			return err
		}
		_, err = db.Exec(ctx, `
			INSERT INTO OTORITAS_DOKUMEN_DETAIL (ID_OTORITAS, ID_USER, TIPE_PERAN, IS_ACTION_DONE)
			VALUES ($1, $2, 'PENYETUJU', FALSE)
		`, idOtoritas, managerID)
		if err != nil {
			return err
		}
		_, err = db.Exec(ctx, `
			INSERT INTO OTORITAS_DOKUMEN_DETAIL (ID_OTORITAS, ID_USER, TIPE_PERAN, IS_ACTION_DONE)
			VALUES ($1, $2, 'RELEASE', FALSE)
		`, idOtoritas, keuanganID)
		if err != nil {
			return err
		}

		slog.Info("PR internal seeded (pending approval)",
			slog.String("nama", pr.nama),
			slog.Int("id_pr", int(idPR)),
			slog.Int("id_otoritas", int(idOtoritas)))
	}

	return nil
}
