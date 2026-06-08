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

	// 9. Seed bootstrap users
	err = seedSystemUsers(ctx, dbPool)
	if err != nil {
		slog.Error("failed to seed system users", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// 10. Seed PO Client & Items
	err = seedPOClient(ctx, dbPool)
	if err != nil {
		slog.Error("failed to seed PO Client", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// 11. Seed Work Order
	err = seedWorkOrder(ctx, dbPool)
	if err != nil {
		slog.Error("failed to seed Work Order", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// 12. Seed Production Reports
	err = seedProductionReports(ctx, dbPool)
	if err != nil {
		slog.Error("failed to seed Production Reports", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// 13. Seed Marker Plan
	err = seedMarkerPlan(ctx, dbPool)
	if err != nil {
		slog.Error("failed to seed Marker Plan", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// 14. Seed Timeline Plan
	err = seedTimelinePlan(ctx, dbPool)
	if err != nil {
		slog.Error("failed to seed Timeline Plan", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// 15. Sync Sequences
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
		{"company_id_company_seq", "profil_perusahaan", "id_profil_perusahaan"},
		{"mitra_id_mitra_seq", "mitra", "id_mitra"},
		{"hak_akses_id_hak_akses_seq", "hak_akses", "id_hak_akses"},
		{"roles_id_role_seq", "roles", "id_role"},
		{"departemen_id_departemen_seq", "departemen", "id_departemen"},
		{"jenis_barang_id_jenis_barang_seq", "jenis_barang", "id_jenis_barang"},
		{"barang_id_barang_seq", "barang", "id_barang"},
		{"users_id_user_seq", "users", "id_user"},
		{"po_client_id_po_client_seq", "po_client", "id_po_client"},
		{"po_client_item_id_po_client_item_seq", "po_client_item", "id_po_client_item"},
		{"work_order_id_wo_seq", "work_order", "id_wo"},
		{"work_order_shell_id_wo_shell_seq", "work_order_shell", "id_wo_shell"},
		{"work_order_shell_size_id_wo_shell_size_seq", "work_order_shell_size", "id_wo_shell_size"},
		{"work_order_trim_id_wo_trim_seq", "work_order_trim", "id_wo_trim"},
		{"material_list_item_id_material_list_item_seq", "material_list_item", "id_material_list_item"},
		{"material_list_id_material_list_seq", "material_list", "id_material_list"},
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

func seedHakAkses(ctx context.Context, db *pgxpool.Pool) error {
	permissionCodes := []string{
		"AUTH_CHANGE_PASSWORD",
		"USER_READ", "USER_CREATE", "USER_UPDATE", "USER_DELETE", "USER_APPROVE",
		"USER_TEMP_PASSWORD_CREATE",
		"ROLE_READ", "ROLE_CREATE", "ROLE_UPDATE", "ROLE_DELETE", "USER_ROLE_ASSIGN",
		"PERMISSION_READ", "PERMISSION_CREATE", "PERMISSION_UPDATE", "PERMISSION_DELETE",
		"MASTER_BARANG_READ", "MASTER_BARANG_CREATE", "MASTER_BARANG_UPDATE", "MASTER_BARANG_DELETE",
		"MASTER_WARNA_READ", "MASTER_WARNA_CREATE", "MASTER_WARNA_UPDATE", "MASTER_WARNA_DELETE",
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
		"INVENTORY_RECEIVE", "INVENTORY_ISSUE",
		"PACKING_LIST_READ", "PACKING_LIST_CREATE", "PACKING_LIST_UPDATE", "PACKING_LIST_APPROVE",
		"SURAT_JALAN_CLIENT_READ", "SURAT_JALAN_INTERNAL_READ", "SURAT_JALAN_CREATE", "SURAT_JALAN_UPDATE",
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
		return "Password Reset Request Read", "Allows operators to read password reset requests", "password_reset_request", "read"
	case "PASSWORD_RESET_REQUEST_APPROVE":
		return "Password Reset Request Approve", "Allows operators to approve password reset requests", "password_reset_request", "approve"
	case "PASSWORD_RESET_REQUEST_REJECT":
		return "Password Reset Request Reject", "Allows operators to reject password reset requests", "password_reset_request", "reject"
	case "USER_TEMP_PASSWORD_CREATE":
		return "User Temporary Password Create", "Allows operators to generate a temporary password while creating or resetting users", "user_temp_password", "create"
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
			"AUTH_CHANGE_PASSWORD",
			"USER_READ", "USER_CREATE", "USER_UPDATE", "USER_DELETE", "USER_APPROVE",
			"USER_TEMP_PASSWORD_CREATE",
			"ROLE_READ", "ROLE_CREATE", "ROLE_UPDATE", "ROLE_DELETE", "USER_ROLE_ASSIGN",
			"PERMISSION_READ", "PERMISSION_CREATE", "PERMISSION_UPDATE", "PERMISSION_DELETE",
			"MASTER_DEPARTEMEN_READ", "MASTER_MITRA_READ",
			"PASSWORD_RESET_REQUEST_CREATE",
			"PASSWORD_RESET_REQUEST_READ", "PASSWORD_RESET_REQUEST_APPROVE", "PASSWORD_RESET_REQUEST_REJECT",
			"LOG_READ", "DASHBOARD_READ",
		},
		"ADMIN_KEUANGAN": {
			"AUTH_CHANGE_PASSWORD", "PASSWORD_RESET_REQUEST_CREATE",
			"MASTER_MITRA_READ", "MASTER_BARANG_READ", "MASTER_JENIS_BARANG_READ", "MASTER_PROFIL_PERUSAHAAN_READ",
			"PO_CLIENT_READ",
			"PR_INTERNAL_READ", "PR_INTERNAL_APPROVE",
			"PO_INTERNAL_READ", "PO_INTERNAL_CREATE", "PO_INTERNAL_UPDATE", "PO_INTERNAL_APPROVE",
			"REPORT_READ", "DASHBOARD_READ",
		},
		"ADMIN_PRODUKSI": {
			"AUTH_CHANGE_PASSWORD", "PASSWORD_RESET_REQUEST_CREATE",
			"MASTER_BARANG_READ", "MASTER_WARNA_READ", "MASTER_MITRA_READ", "MASTER_JENIS_BARANG_READ", "MASTER_PROFIL_PERUSAHAAN_READ", "MASTER_DEPARTEMEN_READ",
			"PO_CLIENT_READ", "PO_CLIENT_CREATE", "PO_CLIENT_UPDATE",
			"WO_READ", "WO_CREATE", "WO_UPDATE", "WO_CLOSE",
			"PRODUCTION_SUMMARY_READ",
			"PRODUCTION_REPORT_READ", "PRODUCTION_REPORT_CREATE", "PRODUCTION_REPORT_UPDATE",
			"TIMELINE_READ", "TIMELINE_CREATE", "TIMELINE_UPDATE",
			"MARKER_PLAN_READ", "MARKER_PLAN_CREATE", "MARKER_PLAN_UPDATE",
			"CUTTING_PLAN_READ", "CUTTING_PLAN_CREATE", "CUTTING_PLAN_UPDATE",
			"PACKING_LIST_READ", "PACKING_LIST_CREATE", "PACKING_LIST_UPDATE",
			"SURAT_JALAN_CLIENT_READ", "SURAT_JALAN_INTERNAL_READ", "SURAT_JALAN_CREATE", "SURAT_JALAN_UPDATE",
			"REPORT_READ", "DASHBOARD_READ", "AI_ESTIMATION_READ",
		},
		"ADMIN_GUDANG": {
			"AUTH_CHANGE_PASSWORD", "PASSWORD_RESET_REQUEST_CREATE",
			"MASTER_BARANG_READ", "MASTER_MITRA_READ", "MASTER_JENIS_BARANG_READ",
			"PR_INTERNAL_READ", "PR_INTERNAL_CREATE", "PR_INTERNAL_UPDATE",
			"INVENTORY_RECEIVE", "INVENTORY_ISSUE",
			"SURAT_JALAN_CLIENT_READ", "SURAT_JALAN_INTERNAL_READ",
			"REPORT_READ", "DASHBOARD_READ",
		},
		"MANAGER": {
			"AUTH_CHANGE_PASSWORD", "PASSWORD_RESET_REQUEST_CREATE",
			"USER_READ", "USER_APPROVE",
			"MASTER_BARANG_READ", "MASTER_WARNA_READ", "MASTER_MITRA_READ", "MASTER_JENIS_BARANG_READ", "MASTER_PROFIL_PERUSAHAAN_READ", "MASTER_DEPARTEMEN_READ",
			"PO_CLIENT_READ", "PR_INTERNAL_READ", "PO_INTERNAL_READ",
			"WO_READ", "PRODUCTION_SUMMARY_READ", "PRODUCTION_REPORT_READ",
			"TIMELINE_READ", "MARKER_PLAN_READ", "CUTTING_PLAN_READ",
			"PACKING_LIST_READ", "SURAT_JALAN_CLIENT_READ", "SURAT_JALAN_INTERNAL_READ",
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
		Username:                 "operator",
		Password:                 "admin123",
		RoleName:                 "OPERATOR",
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
			WHERE w.MODEL = 'WO Test 1' AND s.COLOR = 'Navy' LIMIT 1
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
			($1, 'PO Test 1', 'Navy', 'PO Test 1 Item Description', 1000, 10.00),
			($1, 'PO Test 2', 'Maroon', 'PO Test 2 Item Description', 1000, 12.00)
		`, idPoClient)
		if err != nil {
			return err
		}
		slog.Info("po client items seeded for PO Test")
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
			INSERT INTO WORK_ORDER_SHELL (FABRIC, CONS, COLOR, ALLOW, BERAT_1_YD, ID_WO, PROVIDED_BY, MATERIAL_TYPE)
			VALUES ('Cotton Fleece', 0.35, 'Navy', 3, 0.22, $1, 'permata', 'fabric')
			RETURNING ID_WO_SHELL
		`, idWo1).Scan(&idShell1)
		if err != nil {
			return err
		}

		// Shell 2: Fabric (Cotton Fleece, Maroon)
		var idShell2 int32
		err = db.QueryRow(ctx, `
			INSERT INTO WORK_ORDER_SHELL (FABRIC, CONS, COLOR, ALLOW, BERAT_1_YD, ID_WO, PROVIDED_BY, MATERIAL_TYPE)
			VALUES ('Cotton Fleece', 0.35, 'Maroon', 3, 0.22, $1, 'permata', 'fabric')
			RETURNING ID_WO_SHELL
		`, idWo1).Scan(&idShell2)
		if err != nil {
			return err
		}

		// Shell 3: Interlining (2016F, White)
		var idShell3 int32
		err = db.QueryRow(ctx, `
			INSERT INTO WORK_ORDER_SHELL (FABRIC, CONS, COLOR, ALLOW, BERAT_1_YD, ID_WO, PROVIDED_BY, MATERIAL_TYPE)
			VALUES ('2016F', 0.1, 'White', 3, 0.05, $1, 'permata', 'interlining')
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
				INSERT INTO WORK_ORDER_SHELL_SIZE (SIZE, QTY, RATIO, ID_WO_SHELL)
				VALUES ($1, $2, $3, $4), ($1, $2, $3, $5), ($1, $2, $3, $6)
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
			VALUES ('Thread', 'Sewing Thread Navy', 'Navy', 'TRM-THR-NAVY', 0.05, 50, 'cones', 'Seam', 'super-admin', 0, $1, 'permata')
			RETURNING ID_WO_TRIM
		`, idWo1).Scan(&idTrimThreadNavy)
		if err != nil {
			return err
		}

		// 2. Thread Maroon
		var idTrimThreadMaroon int32
		err = db.QueryRow(ctx, `
			INSERT INTO WORK_ORDER_TRIM (ITEM, DESCRIPTION, COLOR, CODE, CONS, QTY, UOM, POSITION, CREATED_BY, ALLOW, ID_WO, PROVIDED_BY)
			VALUES ('Thread', 'Sewing Thread Maroon', 'Maroon', 'TRM-THR-MAROON', 0.05, 50, 'cones', 'Seam', 'super-admin', 0, $1, 'permata')
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
				VALUES ('Main Size Label', 'Main Size Label ' || $1, 'White', 'TRM-LBL-' || $1, 1.0, $2 + 3, 'pcs', 'Neck', 'super-admin', 3, $3, 'permata')
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
			VALUES ('Kancing', 'Button Navy 24L', 'Navy', 'TRM-BTN-NAVY', 10.0, 10300, 'pcs', 'Front', 'super-admin', 3, $1, 'permata')
			RETURNING ID_WO_TRIM
		`, idWo1).Scan(&idTrimButtonNavy)
		if err != nil {
			return err
		}

		// 5. Buttons Maroon
		var idTrimButtonMaroon int32
		err = db.QueryRow(ctx, `
			INSERT INTO WORK_ORDER_TRIM (ITEM, DESCRIPTION, COLOR, CODE, CONS, QTY, UOM, POSITION, CREATED_BY, ALLOW, ID_WO, PROVIDED_BY)
			VALUES ('Kancing', 'Button Maroon 24L', 'Maroon', 'TRM-BTN-MAROON', 10.0, 10300, 'pcs', 'Front', 'super-admin', 3, $1, 'permata')
			RETURNING ID_WO_TRIM
		`, idWo1).Scan(&idTrimButtonMaroon)
		if err != nil {
			return err
		}

		// Material List Items for Shells and Trims of WO 1
		var mliIDs []int32

		// Shells Material List Items
		for i, idShell := range []int32{idShell1, idShell2, idShell3} {
			var idMli int32
			desc := fmt.Sprintf("WO Test 1 Shell %d Fabric", i+1)
			err = db.QueryRow(ctx, `
				INSERT INTO MATERIAL_LIST_ITEM (DESCRIPTION, ID_WO_SHELL, ID_WO_TRIM)
				VALUES ($1, $2, NULL)
				RETURNING ID_MATERIAL_LIST_ITEM
			`, desc, idShell).Scan(&idMli)
			if err != nil {
				return err
			}
			mliIDs = append(mliIDs, idMli)
		}

		// Trims Material List Items
		trimIDs := append([]int32{idTrimThreadNavy, idTrimThreadMaroon}, labelTrimIDs...)
		trimIDs = append(trimIDs, idTrimButtonNavy, idTrimButtonMaroon)

		for i, idTrim := range trimIDs {
			var idMli int32
			desc := fmt.Sprintf("WO Test 1 Trim %d", i+1)
			err = db.QueryRow(ctx, `
				INSERT INTO MATERIAL_LIST_ITEM (DESCRIPTION, ID_WO_SHELL, ID_WO_TRIM)
				VALUES ($1, NULL, $2)
				RETURNING ID_MATERIAL_LIST_ITEM
			`, desc, idTrim).Scan(&idMli)
			if err != nil {
				return err
			}
			mliIDs = append(mliIDs, idMli)
		}

		// Insert into Material List
		for _, idMli := range mliIDs {
			_, err = db.Exec(ctx, `
				INSERT INTO MATERIAL_LIST (ID_MATERIAL_LIST_ITEM)
				VALUES ($1)
			`, idMli)
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
			INSERT INTO WORK_ORDER_SHELL (FABRIC, CONS, COLOR, ALLOW, BERAT_1_YD, ID_WO, PROVIDED_BY, MATERIAL_TYPE)
			VALUES ('Cotton Combed 30s', 0.35, 'Maroon', 3, 0.22, $1, 'permata', 'fabric')
			RETURNING ID_WO_SHELL
		`, idWo2).Scan(&idShellWO2)
		if err != nil {
			return err
		}

		// WO 2 Details: Sizes
		for _, sz := range sizes {
			_, err = db.Exec(ctx, `
				INSERT INTO WORK_ORDER_SHELL_SIZE (SIZE, QTY, RATIO, ID_WO_SHELL)
				VALUES ($1, $2, $3, $4)
			`, sz.size, sz.qty, sz.ratio, idShellWO2)
			if err != nil {
				return err
			}
		}

		// WO 2 Details: Material List Item
		var idMliWO2 int32
		err = db.QueryRow(ctx, `
			INSERT INTO MATERIAL_LIST_ITEM (DESCRIPTION, ID_WO_SHELL, ID_WO_TRIM)
			VALUES ('WO Test 2 Fabric', $1, NULL)
			RETURNING ID_MATERIAL_LIST_ITEM
		`, idShellWO2).Scan(&idMliWO2)
		if err != nil {
			return err
		}

		// WO 2 Details: Material List
		_, err = db.Exec(ctx, `
			INSERT INTO MATERIAL_LIST (ID_MATERIAL_LIST_ITEM)
			VALUES ($1)
		`, idMliWO2)
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
			SELECT woss.ID_WO_SHELL_SIZE, woss.SIZE, woss.QTY
			FROM WORK_ORDER_SHELL_SIZE woss
			JOIN WORK_ORDER_SHELL wos ON wos.ID_WO_SHELL = woss.ID_WO_SHELL
			JOIN WORK_ORDER wo ON wo.ID_WO = wos.ID_WO
			WHERE wo.MODEL = 'WO Test 1' AND wos.COLOR = 'Navy'
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
			WHERE wo.MODEL = 'WO Test 1' AND wos.COLOR = 'Navy' LIMIT 1
		`).Scan(&idShell)
	if err != nil {
		return fmt.Errorf("failed to find WO Shell for Navy Fabric WO Test 1: %w", err)
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

	// 3. Insert Komponen Marker Plan
	var idKomponen int32
	err = db.QueryRow(ctx, `
			INSERT INTO KOMPONEN_MARKER_PLAN (ID_MARKER_PLAN, NAMA_KOMPONEN)
			VALUES ($1, 'Cotton Fleece Navy')
			RETURNING ID_KOMPONEN_MARKER
		`, idMarkerPlan).Scan(&idKomponen)
	if err != nil {
		return err
	}

	// 4. Insert Ratio Marker
	var idRatioMarker int32
	err = db.QueryRow(ctx, `
			INSERT INTO RATIO_MARKER (
				ID_KOMPONEN_MARKER, ID_WO_SHELL, CONS, PLAN_SPREADING_GELARAN,
				PANJANG_MARKER, EFFICIENCY_MARKER, ALLOWANCE, CONS_BUYER,
				PLOT, LEBAR_KAIN, PANJANG_MARKER_UNIT, KET
			) VALUES ($1, $2, 0.350, 100, 35.000, 85.50, 3.00, 0.360, 1, 1.450, 'yard', 'Seeded Ratio')
			RETURNING ID_RATIO_MARKER
		`, idKomponen, idShell).Scan(&idRatioMarker)
	if err != nil {
		return err
	}

	// 5. Get Shell size IDs of WO 1 (Navy Fabric: XS to XXL)
	rows, err := db.Query(ctx, `
			SELECT ID_WO_SHELL_SIZE, SIZE
			FROM WORK_ORDER_SHELL_SIZE
			WHERE ID_WO_SHELL = $1
		`, idShell)
	if err != nil {
		return err
	}
	defer rows.Close()

	type SizeInfo struct {
		ID   int32
		Size string
	}
	var sizes []SizeInfo
	for rows.Next() {
		var sz SizeInfo
		if err := rows.Scan(&sz.ID, &sz.Size); err != nil {
			return err
		}
		sizes = append(sizes, sz)
	}

	// 6. Insert Ratio Size Markers
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
			`, idRatioMarker, sz.ID, ratioPlan)
		if err != nil {
			return err
		}
	}
	slog.Info("ratio size markers seeded for MP-2026-001")

	return nil
}
