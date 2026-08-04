package migrations_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

const migration42TestDatabaseEnv = "BACKUP_MIGRATION_TEST_DATABASE_URL"

func TestMigration42SQLDeclaresExclusiveOwnership(t *testing.T) {
	up := readMigration(t, "000042_system_backup_permissions.up.sql")
	down := readMigration(t, "000042_system_backup_permissions.down.sql")
	permissionInsert := strings.Split(strings.ToUpper(up), "INSERT INTO ROLE_HAK_AKSES")[0]
	if strings.Contains(permissionInsert, "ON CONFLICT") {
		t.Fatal("up migration must fail rather than adopt a pre-existing permission")
	}
	for _, metadata := range []string{"KODE_PERMISSION", "NAMA_HALAMAN", "DESKRIPSI", "DOMAIN_PERMISSION", "AKSI_PERMISSION"} {
		if !strings.Contains(strings.ToUpper(down), metadata) {
			t.Fatalf("down migration does not identify owned records by %s", metadata)
		}
	}
	if strings.Contains(strings.ToUpper(down), "DELETE FROM USER_AKSES") {
		t.Fatal("down migration must not broadly delete user overrides")
	}
}

func TestMigration42PostgresBehavior(t *testing.T) {
	databaseURL := strings.TrimSpace(os.Getenv(migration42TestDatabaseEnv))
	if databaseURL == "" {
		t.Skip(migration42TestDatabaseEnv + " is not configured")
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	up := readMigration(t, "000042_system_backup_permissions.up.sql")
	down := readMigration(t, "000042_system_backup_permissions.down.sql")

	t.Run("clean up assigns roles and down preserves unrelated data", func(t *testing.T) {
		tx, err := conn.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(ctx)
		setupMigration42Schema(t, ctx, tx)
		if _, err := tx.Exec(ctx, up); err != nil {
			t.Fatalf("clean up migration failed: %v", err)
		}
		assertCount(t, ctx, tx, `
			SELECT COUNT(*) FROM ROLE_HAK_AKSES ra
			JOIN ROLES r ON r.ID_ROLE = ra.ID_ROLE
			JOIN HAK_AKSES h ON h.ID_HAK_AKSES = ra.ID_HAK_AKSES
			WHERE r.NAMA_ROLE = 'ADMIN_SISTEM' AND h.KODE_PERMISSION LIKE 'SYSTEM_BACKUP_%'`, 3)
		assertCount(t, ctx, tx, `
			SELECT COUNT(*) FROM ROLE_HAK_AKSES ra
			JOIN ROLES r ON r.ID_ROLE = ra.ID_ROLE
			JOIN HAK_AKSES h ON h.ID_HAK_AKSES = ra.ID_HAK_AKSES
			WHERE r.NAMA_ROLE = 'MANAGER' AND h.KODE_PERMISSION = 'SYSTEM_BACKUP_READ'`, 1)
		assertCount(t, ctx, tx, `
			SELECT COUNT(*) FROM ROLE_HAK_AKSES ra
			JOIN ROLES r ON r.ID_ROLE = ra.ID_ROLE
			JOIN HAK_AKSES h ON h.ID_HAK_AKSES = ra.ID_HAK_AKSES
			WHERE r.NAMA_ROLE = 'MANAGER' AND h.KODE_PERMISSION IN ('SYSTEM_BACKUP_CREATE', 'SYSTEM_BACKUP_DOWNLOAD')`, 0)
		if _, err := tx.Exec(ctx, down); err != nil {
			t.Fatalf("down migration failed: %v", err)
		}
		assertCount(t, ctx, tx, `SELECT COUNT(*) FROM HAK_AKSES WHERE KODE_PERMISSION LIKE 'SYSTEM_BACKUP_%'`, 0)
		assertCount(t, ctx, tx, `
			SELECT COUNT(*) FROM ROLE_HAK_AKSES ra
			JOIN HAK_AKSES h ON h.ID_HAK_AKSES = ra.ID_HAK_AKSES
			WHERE h.KODE_PERMISSION = 'UNRELATED_READ'`, 1)
		assertCount(t, ctx, tx, `
			SELECT COUNT(*) FROM USER_AKSES ua
			JOIN HAK_AKSES h ON h.ID_HAK_AKSES = ua.ID_HAK_AKSES
			WHERE h.KODE_PERMISSION = 'UNRELATED_READ'`, 1)
	})

	t.Run("pre-existing code aborts up", func(t *testing.T) {
		tx, err := conn.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(ctx)
		setupMigration42Schema(t, ctx, tx)
		if _, err := tx.Exec(ctx, `
			INSERT INTO HAK_AKSES (KODE_PERMISSION, NAMA_HALAMAN, DESKRIPSI, DOMAIN_PERMISSION, AKSI_PERMISSION)
			VALUES ('SYSTEM_BACKUP_READ', 'Existing', 'Existing', 'custom', 'read')`); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(ctx, up); err == nil {
			t.Fatal("up migration adopted a pre-existing permission code")
		}
	})
}

func setupMigration42Schema(t *testing.T, ctx context.Context, tx pgx.Tx) {
	t.Helper()
	_, err := tx.Exec(ctx, `
		CREATE TEMP TABLE ROLES (
			ID_ROLE SERIAL PRIMARY KEY,
			NAMA_ROLE VARCHAR(100) NOT NULL UNIQUE
		);
		CREATE TEMP TABLE HAK_AKSES (
			ID_HAK_AKSES SERIAL PRIMARY KEY,
			KODE_PERMISSION VARCHAR(150) NOT NULL UNIQUE,
			NAMA_HALAMAN VARCHAR(100) NOT NULL,
			DESKRIPSI TEXT NOT NULL DEFAULT '',
			DOMAIN_PERMISSION VARCHAR(100) NOT NULL,
			AKSI_PERMISSION VARCHAR(50) NOT NULL
		);
		CREATE TEMP TABLE ROLE_HAK_AKSES (
			ID_ROLE INT NOT NULL REFERENCES ROLES(ID_ROLE) ON DELETE CASCADE,
			ID_HAK_AKSES INT NOT NULL REFERENCES HAK_AKSES(ID_HAK_AKSES) ON DELETE CASCADE,
			PRIMARY KEY (ID_ROLE, ID_HAK_AKSES)
		);
		CREATE TEMP TABLE USER_AKSES (
			ID_USER INT NOT NULL,
			ID_HAK_AKSES INT NOT NULL REFERENCES HAK_AKSES(ID_HAK_AKSES) ON DELETE CASCADE,
			PRIMARY KEY (ID_USER, ID_HAK_AKSES)
		);
		INSERT INTO ROLES (NAMA_ROLE) VALUES ('ADMIN_SISTEM'), ('MANAGER'), ('OTHER');
		INSERT INTO HAK_AKSES (KODE_PERMISSION, NAMA_HALAMAN, DESKRIPSI, DOMAIN_PERMISSION, AKSI_PERMISSION)
		VALUES ('UNRELATED_READ', 'Unrelated Read', 'Unrelated permission', 'unrelated', 'read');
		INSERT INTO ROLE_HAK_AKSES (ID_ROLE, ID_HAK_AKSES)
		SELECT r.ID_ROLE, h.ID_HAK_AKSES FROM ROLES r CROSS JOIN HAK_AKSES h
		WHERE r.NAMA_ROLE = 'OTHER' AND h.KODE_PERMISSION = 'UNRELATED_READ';
		INSERT INTO USER_AKSES (ID_USER, ID_HAK_AKSES)
		SELECT 99, ID_HAK_AKSES FROM HAK_AKSES WHERE KODE_PERMISSION = 'UNRELATED_READ';`)
	if err != nil {
		t.Fatal(err)
	}
}

func assertCount(t *testing.T, ctx context.Context, tx pgx.Tx, query string, want int) {
	t.Helper()
	var got int
	if err := tx.QueryRow(ctx, query).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("query count=%d want=%d", got, want)
	}
}

func readMigration(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
