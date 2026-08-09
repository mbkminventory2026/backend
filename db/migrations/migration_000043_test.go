package migrations_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

const migration43TestDatabaseEnv = "MATERIAL_LIST_EXPORT_MIGRATION_TEST_DATABASE_URL"

func TestMigration43DeclaresNullableExportFieldsAndDownReversesOnlyThem(t *testing.T) {
	up := strings.ToUpper(readMigration(t, "000043_material_list_export_fields.up.sql"))
	down := strings.ToUpper(readMigration(t, "000043_material_list_export_fields.down.sql"))

	for _, column := range []string{"CATEGORY VARCHAR(16)", "CONS_PER_PC DECIMAL(15,3)", "QTY_WO_SCOPE VARCHAR(16)", "ID_QTY_WO_SHELL INT", "ID_QTY_WO_SIZE INT"} {
		if !strings.Contains(up, column) {
			t.Fatalf("up migration missing %s", column)
		}
	}
	for _, check := range []string{"'FABRIC', 'SEWING', 'PACKING'", "CONS_PER_PC >= 0", "'WHOLE_WO', 'SIZE', 'COLOR', 'COLOR_SIZE'", "QTY_WO_SCOPE IS NULL AND ID_QTY_WO_SHELL IS NULL AND ID_QTY_WO_SIZE IS NULL", ") IS TRUE"} {
		if !strings.Contains(up, check) {
			t.Fatalf("up migration missing check %s", check)
		}
	}
	for _, fk := range []string{"REFERENCES WORK_ORDER_SHELL(ID_WO_SHELL) ON DELETE RESTRICT", "REFERENCES MASTER_SIZE(ID_SIZE) ON DELETE RESTRICT"} {
		if !strings.Contains(up, fk) {
			t.Fatalf("up migration missing FK %s", fk)
		}
	}
	for _, addition := range []string{"CATEGORY", "CONS_PER_PC", "QTY_WO_SCOPE", "ID_QTY_WO_SHELL", "ID_QTY_WO_SIZE"} {
		if !strings.Contains(down, "DROP COLUMN IF EXISTS "+addition) {
			t.Fatalf("down migration does not remove %s", addition)
		}
	}
	if strings.Contains(up, "UPDATE MATERIAL_LIST_ITEM") || strings.Contains(up, "INSERT INTO MATERIAL_LIST_ITEM") {
		t.Fatal("migration must not backfill or alter existing material list items")
	}
}

func TestMigration43PostgresBehavior(t *testing.T) {
	databaseURL := strings.TrimSpace(os.Getenv(migration43TestDatabaseEnv))
	if databaseURL == "" {
		t.Skip(migration43TestDatabaseEnv + " is not configured")
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)

	_, err = conn.Exec(ctx, `
		CREATE TEMP TABLE WORK_ORDER_SHELL (ID_WO_SHELL INT PRIMARY KEY);
		CREATE TEMP TABLE MASTER_SIZE (ID_SIZE INT PRIMARY KEY);
		CREATE TEMP TABLE MATERIAL_LIST_ITEM (ID_MATERIAL_LIST_ITEM SERIAL PRIMARY KEY);
		INSERT INTO WORK_ORDER_SHELL (ID_WO_SHELL) VALUES (1);
		INSERT INTO MASTER_SIZE (ID_SIZE) VALUES (1);`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(ctx, readMigration(t, "000043_material_list_export_fields.up.sql")); err != nil {
		t.Fatalf("up migration failed: %v", err)
	}
	if _, err := conn.Exec(ctx, `INSERT INTO MATERIAL_LIST_ITEM DEFAULT VALUES`); err != nil {
		t.Fatalf("legacy null row rejected: %v", err)
	}
	for _, query := range []string{
		`INSERT INTO MATERIAL_LIST_ITEM (CATEGORY) VALUES ('INVALID')`,
		`INSERT INTO MATERIAL_LIST_ITEM (CONS_PER_PC) VALUES (-0.001)`,
		`INSERT INTO MATERIAL_LIST_ITEM (QTY_WO_SCOPE) VALUES ('INVALID')`,
		`INSERT INTO MATERIAL_LIST_ITEM (ID_QTY_WO_SHELL) VALUES (1)`,
		`INSERT INTO MATERIAL_LIST_ITEM (QTY_WO_SCOPE, ID_QTY_WO_SHELL) VALUES ('SIZE', 1)`,
	} {
		if _, err := conn.Exec(ctx, query); err == nil {
			t.Fatalf("invalid row accepted: %s", query)
		}
	}
	if _, err := conn.Exec(ctx, readMigration(t, "000043_material_list_export_fields.down.sql")); err != nil {
		t.Fatalf("down migration failed: %v", err)
	}
}
