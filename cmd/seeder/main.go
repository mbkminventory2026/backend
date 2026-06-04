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

	// 13. Sync Sequences
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
		{"po_client_id_po_client_seq", "po_client", "id_po_client"},
		{"po_client_item_id_po_client_item_seq", "po_client_item", "id_po_client_item"},
		{"work_order_id_wo_seq", "work_order", "id_wo"},
		{"work_order_shell_id_wo_shell_seq", "work_order_shell", "id_wo_shell"},
		{"work_order_shell_size_id_wo_shell_size_seq", "work_order_shell_size", "id_wo_shell_size"},
		{"work_order_trim_id_wo_trim_seq", "work_order_trim", "id_wo_trim"},
		{"material_list_id_material_list_seq", "material_list", "id_material_list"},
		{"report_cutting_id_report_cutting_seq", "report_cutting", "id_report_cutting"},
		{"report_sewing_id_report_sewing_seq", "report_sewing", "id_report_sewing"},
		{"report_qc_finish_id_report_qc_finishing_seq", "report_qc_finish", "id_report_qc_finishing"},
		{"report_packing_id_report_packing_seq", "report_packing", "id_report_packing"},
		{"report_pengiriman_id_report_pengiriman_seq", "report_pengiriman", "id_report_pengiriman"},
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
		"AUTH_CHANGE_PASSWORD",
		"USER_READ", "USER_CREATE", "USER_UPDATE", "USER_DELETE", "USER_APPROVE",
		"USER_TEMP_PASSWORD_CREATE",
		"ROLE_READ", "ROLE_CREATE", "ROLE_UPDATE", "ROLE_DELETE", "USER_ROLE_ASSIGN",
		"PERMISSION_READ", "PERMISSION_CREATE", "PERMISSION_UPDATE", "PERMISSION_DELETE",
		"MASTER_BARANG_READ", "MASTER_BARANG_CREATE", "MASTER_BARANG_UPDATE", "MASTER_BARANG_DELETE",
		"MASTER_WARNA_READ", "MASTER_WARNA_CREATE", "MASTER_WARNA_UPDATE", "MASTER_WARNA_DELETE",
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
		"PASSWORD_RESET_REQUEST_CREATE",
		"PASSWORD_RESET_REQUEST_READ", "PASSWORD_RESET_REQUEST_APPROVE", "PASSWORD_RESET_REQUEST_REJECT",
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
			"PASSWORD_RESET_REQUEST_CREATE",
			"PASSWORD_RESET_REQUEST_READ", "PASSWORD_RESET_REQUEST_APPROVE", "PASSWORD_RESET_REQUEST_REJECT",
			"LOG_READ",
		},
		"ADMIN_KEUANGAN": {
			"AUTH_CHANGE_PASSWORD", "PASSWORD_RESET_REQUEST_CREATE",
			"MASTER_MITRA_READ", "MASTER_BARANG_READ", "MASTER_JENIS_BARANG_READ", "MASTER_COMPANY_READ",
			"PO_CLIENT_READ",
			"PR_INTERNAL_READ", "PR_INTERNAL_APPROVE",
			"PO_INTERNAL_READ", "PO_INTERNAL_CREATE", "PO_INTERNAL_UPDATE", "PO_INTERNAL_APPROVE",
			"REPORT_READ",
		},
		"ADMIN_PRODUKSI": {
			"AUTH_CHANGE_PASSWORD", "PASSWORD_RESET_REQUEST_CREATE",
			"MASTER_BARANG_READ", "MASTER_WARNA_READ", "MASTER_MITRA_READ", "MASTER_JENIS_BARANG_READ", "MASTER_COMPANY_READ", "MASTER_DEPARTEMEN_READ",
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
			"AUTH_CHANGE_PASSWORD", "PASSWORD_RESET_REQUEST_CREATE",
			"MASTER_BARANG_READ", "MASTER_MITRA_READ", "MASTER_JENIS_BARANG_READ",
			"PR_INTERNAL_READ", "PR_INTERNAL_CREATE", "PR_INTERNAL_UPDATE",
			"INVENTORY_RECEIVE", "INVENTORY_ISSUE",
			"SURAT_JALAN_CLIENT_READ", "SURAT_JALAN_INTERNAL_READ",
			"REPORT_READ",
		},
		"MANAGER": {
			"AUTH_CHANGE_PASSWORD", "PASSWORD_RESET_REQUEST_CREATE",
			"USER_READ", "USER_APPROVE",
			"MASTER_BARANG_READ", "MASTER_WARNA_READ", "MASTER_MITRA_READ", "MASTER_JENIS_BARANG_READ", "MASTER_COMPANY_READ", "MASTER_DEPARTEMEN_READ",
			"PO_CLIENT_READ", "PR_INTERNAL_READ", "PO_INTERNAL_READ",
			"WO_READ", "PRODUCTION_SUMMARY_READ", "PRODUCTION_REPORT_READ",
			"TIMELINE_READ", "MARKER_PLAN_READ", "CUTTING_PLAN_READ",
			"PACKING_LIST_READ", "SURAT_JALAN_CLIENT_READ", "SURAT_JALAN_INTERNAL_READ",
			"REPORT_READ", "LOG_READ", "DASHBOARD_READ",
			"PR_INTERNAL_APPROVE", "PO_INTERNAL_APPROVE", "PACKING_LIST_APPROVE",
		},
		"CLIENT": {
			"AUTH_CHANGE_PASSWORD", "PASSWORD_RESET_REQUEST_CREATE",
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
	// 1. PO-CLI-2026-001 (Mitra ID 4 = Garment Client Global)
	var exists bool
	err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM PO_CLIENT WHERE PO_NUMBER = 'PO-CLI-2026-001')`).Scan(&exists)
	if err != nil {
		return err
	}

	if !exists {
		// Insert PO Client
		var idPoClient int32
		err = db.QueryRow(ctx, `
			INSERT INTO PO_CLIENT (PO_NUMBER, TANGGAL, SEASON, DELIVERY, PAYMENT_TERM, FILE, ID_MITRA)
			VALUES ('PO-CLI-2026-001', '2026-06-01', 'Summer 2026', '2026-08-30', 'Net 30', 'po_file_001.pdf', 4)
			RETURNING ID_PO_CLIENT
		`).Scan(&idPoClient)
		if err != nil {
			return err
		}
		slog.Info("po client seeded: PO-CLI-2026-001", slog.Int("id", int(idPoClient)))

		// Insert PO Client Items
		_, err = db.Exec(ctx, `
			INSERT INTO PO_CLIENT_ITEM (ID_PO_CLIENT, STYLE, COLOUR, DESCRIPTION, QTY, PRICE)
			VALUES 
			($1, 'TSH-BASIC-01', 'Black', 'Basic Crewneck Cotton T-Shirt Black', 1000, 5.50),
			($1, 'TSH-BASIC-02', 'White', 'Basic Crewneck Cotton T-Shirt White', 500, 5.50)
		`, idPoClient)
		if err != nil {
			return err
		}
		slog.Info("po client items seeded for PO-CLI-2026-001")
	}

	// 2. PO-CLI-2026-002 (Mitra ID 5 = Fashion Brand Indonesia)
	err = db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM PO_CLIENT WHERE PO_NUMBER = 'PO-CLI-2026-002')`).Scan(&exists)
	if err != nil {
		return err
	}

	if !exists {
		var idPoClient int32
		err = db.QueryRow(ctx, `
			INSERT INTO PO_CLIENT (PO_NUMBER, TANGGAL, SEASON, DELIVERY, PAYMENT_TERM, FILE, ID_MITRA)
			VALUES ('PO-CLI-2026-002', '2026-06-02', 'Summer 2026', '2026-09-15', 'Net 30', 'po_file_002.pdf', 5)
			RETURNING ID_PO_CLIENT
		`).Scan(&idPoClient)
		if err != nil {
			return err
		}
		slog.Info("po client seeded: PO-CLI-2026-002", slog.Int("id", int(idPoClient)))

		_, err = db.Exec(ctx, `
			INSERT INTO PO_CLIENT_ITEM (ID_PO_CLIENT, STYLE, COLOUR, DESCRIPTION, QTY, PRICE)
			VALUES ($1, 'HOO-OVER-01', 'Misty Grey', 'Oversized Hoodie Misty Grey', 800, 12.00)
		`, idPoClient)
		if err != nil {
			return err
		}
		slog.Info("po client items seeded for PO-CLI-2026-002")
	}

	return nil
}

func seedWorkOrder(ctx context.Context, db *pgxpool.Pool) error {
	// Check if we already have Work Orders
	var exists bool
	err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM WORK_ORDER WHERE MODEL IN ('Basic Crewneck T-Shirt Black', 'Basic Crewneck T-Shirt White'))`).Scan(&exists)
	if err != nil {
		return err
	}

	if !exists {
		// 1. Get PO Client Item IDs
		var idItem1, idItem2 int32
		err = db.QueryRow(ctx, `SELECT ID_PO_CLIENT_ITEM FROM PO_CLIENT_ITEM WHERE STYLE = 'TSH-BASIC-01' LIMIT 1`).Scan(&idItem1)
		if err != nil {
			return fmt.Errorf("failed to find PO Item TSH-BASIC-01: %w", err)
		}
		err = db.QueryRow(ctx, `SELECT ID_PO_CLIENT_ITEM FROM PO_CLIENT_ITEM WHERE STYLE = 'TSH-BASIC-02' LIMIT 1`).Scan(&idItem2)
		if err != nil {
			return fmt.Errorf("failed to find PO Item TSH-BASIC-02: %w", err)
		}

		// WO 1: Basic Crewneck T-Shirt Black
		var idWo1 int32
		err = db.QueryRow(ctx, `
			INSERT INTO WORK_ORDER (BUYER, MODEL, QTY, FOB_CMT, DELIVERY, ID_PO_CLIENT_ITEM)
			VALUES ('Garment Client Global', 'Basic Crewneck T-Shirt Black', 1000, true, '2026-08-30', $1)
			RETURNING ID_WO
		`, idItem1).Scan(&idWo1)
		if err != nil {
			return err
		}
		slog.Info("work order 1 seeded: Basic Crewneck T-Shirt Black", slog.Int("id", int(idWo1)))

		// WO 1 Details: Shell
		var idShell1 int32
		err = db.QueryRow(ctx, `
			INSERT INTO WORK_ORDER_SHELL (FABRIC, CONS, COLOR, ALLOW, BERAT_1_YD, ID_WO)
			VALUES ('Cotton Combed 30s', 0.35, 'Black', 3, 0.22, $1)
			RETURNING ID_WO_SHELL
		`, idWo1).Scan(&idShell1)
		if err != nil {
			return err
		}

		// WO 1 Details: Sizes
		_, err = db.Exec(ctx, `
			INSERT INTO WORK_ORDER_SHELL_SIZE (SIZE, QTY, RATIO, ID_WO_SHELL)
			VALUES 
			('S', 200, 2, $1),
			('M', 300, 3, $1),
			('L', 300, 3, $1),
			('XL', 200, 2, $1)
		`, idShell1)
		if err != nil {
			return err
		}

		// WO 1 Details: Trims
		_, err = db.Exec(ctx, `
			INSERT INTO WORK_ORDER_TRIM (ITEM, DESCRIPTION, COLOR, CODE, CONS, QTY, UOM, POSITION, CREATED_BY, ALLOW, ID_WO)
			VALUES 
			('Label Leher', 'Satin Label', 'Black', 'TRM-LBL-01', 1.0, 1030, 'pcs', 'Neck', 'super-admin', 3, $1),
			('Thread 120', 'Sewing Thread', 'Black', 'TRM-THR-01', 0.05, 50, 'cones', 'Seam', 'super-admin', 0, $1)
		`, idWo1)
		if err != nil {
			return err
		}

		// WO 1 Details: Material List
		_, err = db.Exec(ctx, `
			INSERT INTO MATERIAL_LIST (DESCRIPTION, SIZE, COLOR, UOM, ID_WO)
			VALUES 
			('Cotton Combed 30s Fabric', 'All Size', 'Black', 'kg', $1),
			('Satin Neck Label', 'Standard', 'Black', 'pcs', $1)
		`, idWo1)
		if err != nil {
			return err
		}

		// WO 2: Basic Crewneck T-Shirt White
		var idWo2 int32
		err = db.QueryRow(ctx, `
			INSERT INTO WORK_ORDER (BUYER, MODEL, QTY, FOB_CMT, DELIVERY, ID_PO_CLIENT_ITEM)
			VALUES ('Garment Client Global', 'Basic Crewneck T-Shirt White', 500, true, '2026-08-30', $1)
			RETURNING ID_WO
		`, idItem2).Scan(&idWo2)
		if err != nil {
			return err
		}
		slog.Info("work order 2 seeded: Basic Crewneck T-Shirt White", slog.Int("id", int(idWo2)))

		// WO 2 Details: Shell
		var idShell2 int32
		err = db.QueryRow(ctx, `
			INSERT INTO WORK_ORDER_SHELL (FABRIC, CONS, COLOR, ALLOW, BERAT_1_YD, ID_WO)
			VALUES ('Cotton Combed 30s', 0.35, 'White', 3, 0.22, $1)
			RETURNING ID_WO_SHELL
		`, idWo2).Scan(&idShell2)
		if err != nil {
			return err
		}

		// WO 2 Details: Sizes
		_, err = db.Exec(ctx, `
			INSERT INTO WORK_ORDER_SHELL_SIZE (SIZE, QTY, RATIO, ID_WO_SHELL)
			VALUES 
			('M', 200, 2, $1),
			('L', 200, 2, $1),
			('XL', 100, 1, $1)
		`, idShell2)
		if err != nil {
			return err
		}

		// WO 2 Details: Trims
		_, err = db.Exec(ctx, `
			INSERT INTO WORK_ORDER_TRIM (ITEM, DESCRIPTION, COLOR, CODE, CONS, QTY, UOM, POSITION, CREATED_BY, ALLOW, ID_WO)
			VALUES ('Label Leher', 'Satin Label', 'White', 'TRM-LBL-02', 1.0, 515, 'pcs', 'Neck', 'super-admin', 3, $1)
		`, idWo2)
		if err != nil {
			return err
		}

		// WO 2 Details: Material List
		_, err = db.Exec(ctx, `
			INSERT INTO MATERIAL_LIST (DESCRIPTION, SIZE, COLOR, UOM, ID_WO)
			VALUES ('Cotton Combed 30s Fabric', 'All Size', 'White', 'kg', $1)
		`, idWo2)
		if err != nil {
			return err
		}
	}

	return nil
}

func seedProductionReports(ctx context.Context, db *pgxpool.Pool) error {
	// Check if reports already exist
	var exists bool
	err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM REPORT_CUTTING)`).Scan(&exists)
	if err != nil {
		return err
	}

	if !exists {
		// Get shell sizes of WO 1 (Black: S, M, L, XL)
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
			WHERE wo.MODEL = 'Basic Crewneck T-Shirt Black'
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

		// Insert report history per size
		for _, sz := range sizes {
			switch sz.Size {
			case "S": // target 200
				// Cutting: 200
				_, err = db.Exec(ctx, `INSERT INTO REPORT_CUTTING (TANGGAL, QTY, ID_WO_SHELL_SIZE) VALUES ('2026-06-01', 200, $1)`, sz.ID)
				if err != nil {
					return err
				}
				// Sewing: 180
				_, err = db.Exec(ctx, `INSERT INTO REPORT_SEWING (TANGGAL, QTY, ID_WO_SHELL_SIZE) VALUES ('2026-06-02', 180, $1)`, sz.ID)
				if err != nil {
					return err
				}
				// QC: 150
				_, err = db.Exec(ctx, `INSERT INTO REPORT_QC_FINISH (TANGGAL, QTY, ID_WO_SHELL_SIZE) VALUES ('2026-06-03', 150, $1)`, sz.ID)
				if err != nil {
					return err
				}
				// Packing: 120
				_, err = db.Exec(ctx, `INSERT INTO REPORT_PACKING (TANGGAL, QTY, ID_WO_SHELL_SIZE) VALUES ('2026-06-03', 120, $1)`, sz.ID)
				if err != nil {
					return err
				}
				// Shipping: 100
				_, err = db.Exec(ctx, `INSERT INTO REPORT_PENGIRIMAN (REPORT_DATE, QTY, ID_WO_SHELL_SIZE) VALUES ('2026-06-03', 100, $1)`, sz.ID)
				if err != nil {
					return err
				}
			case "M": // target 300
				// Cutting: 300
				_, err = db.Exec(ctx, `INSERT INTO REPORT_CUTTING (TANGGAL, QTY, ID_WO_SHELL_SIZE) VALUES ('2026-06-01', 300, $1)`, sz.ID)
				if err != nil {
					return err
				}
				// Sewing: 250
				_, err = db.Exec(ctx, `INSERT INTO REPORT_SEWING (TANGGAL, QTY, ID_WO_SHELL_SIZE) VALUES ('2026-06-02', 250, $1)`, sz.ID)
				if err != nil {
					return err
				}
				// QC: 200
				_, err = db.Exec(ctx, `INSERT INTO REPORT_QC_FINISH (TANGGAL, QTY, ID_WO_SHELL_SIZE) VALUES ('2026-06-03', 200, $1)`, sz.ID)
				if err != nil {
					return err
				}
				// Packing: 150
				_, err = db.Exec(ctx, `INSERT INTO REPORT_PACKING (TANGGAL, QTY, ID_WO_SHELL_SIZE) VALUES ('2026-06-03', 150, $1)`, sz.ID)
				if err != nil {
					return err
				}
				// Shipping: 0
			case "L": // target 300
				// Cutting: 280
				_, err = db.Exec(ctx, `INSERT INTO REPORT_CUTTING (TANGGAL, QTY, ID_WO_SHELL_SIZE) VALUES ('2026-06-02', 280, $1)`, sz.ID)
				if err != nil {
					return err
				}
				// Sewing: 100
				_, err = db.Exec(ctx, `INSERT INTO REPORT_SEWING (TANGGAL, QTY, ID_WO_SHELL_SIZE) VALUES ('2026-06-03', 100, $1)`, sz.ID)
				if err != nil {
					return err
				}
			case "XL": // target 200
				// Cutting: 100
				_, err = db.Exec(ctx, `INSERT INTO REPORT_CUTTING (TANGGAL, QTY, ID_WO_SHELL_SIZE) VALUES ('2026-06-03', 100, $1)`, sz.ID)
				if err != nil {
					return err
				}
			}
		}
		slog.Info("production reports seeded successfully for Basic Crewneck T-Shirt Black")
	}

	return nil
}
