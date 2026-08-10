package usecase

import (
	"bytes"
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/xuri/excelize/v2"

	"permatatex-inventory/internal/entity"
	"permatatex-inventory/internal/model"
)

func TestMaterialListImportParserHappyPathAndNormalization(t *testing.T) {
	book := materialListImportBook(t)
	setImportRow(t, book, 2, []interface{}{" fabric ", " Item A ", " desc ", "0", " pcs ", "0.00", "", "none", "", "whole_wo", "", ""})
	setImportRow(t, book, 3, []interface{}{"SEWING", "Item B", "", "12", "PCS", "12.50", "0", "trim", "7", "color", "11", ""})
	rows, validation := parseMaterialListImportWorkbook(42, materialListImportBytes(t, book))
	if validation.ErrorCount != 0 {
		t.Fatalf("validation = %+v", validation.Errors)
	}
	if len(rows) != 2 || *rows[0].request.Category != "FABRIC" || *rows[0].request.QtyWoScope != "WHOLE_WO" || rows[0].consText != "" {
		t.Fatalf("first row = %+v", rows[0])
	}
	if rows[0].priceText != "0.00" || rows[1].priceText != "12.50" || rows[1].consText != "0.000" || rows[1].request.IDWoTrim == nil || rows[1].request.IDWoShell != nil {
		t.Fatalf("normalized decimals/source = %+v", rows)
	}
}

func TestMaterialListImportParserRejectsCoreRowRules(t *testing.T) {
	cases := []struct {
		name string
		row  []interface{}
		code string
	}{
		{"category alias", []interface{}{"Colour", "Item", "", "1", "PCS", "1", "", "NONE", "", "WHOLE_WO", "", ""}, "invalid_enum"},
		{"decimal quantity", []interface{}{"FABRIC", "Item", "", "1.5", "PCS", "1", "", "NONE", "", "WHOLE_WO", "", ""}, "invalid_integer"},
		{"negative price", []interface{}{"FABRIC", "Item", "", "1", "PCS", "-1", "", "NONE", "", "WHOLE_WO", "", ""}, "invalid_decimal"},
		{"price scale", []interface{}{"FABRIC", "Item", "", "1", "PCS", "1.001", "", "NONE", "", "WHOLE_WO", "", ""}, "invalid_precision"},
		{"none source id", []interface{}{"FABRIC", "Item", "", "1", "PCS", "1", "", "NONE", "7", "WHOLE_WO", "", ""}, "invalid_source"},
		{"color missing shell", []interface{}{"FABRIC", "Item", "", "1", "PCS", "1", "", "NONE", "", "COLOR", "", ""}, "invalid_applicability"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			book := materialListImportBook(t)
			setImportRow(t, book, 2, tc.row)
			_, validation := parseMaterialListImportWorkbook(42, materialListImportBytes(t, book))
			if !materialListImportHasCode(validation, tc.code) {
				t.Fatalf("errors = %+v, want %s", validation.Errors, tc.code)
			}
		})
	}
}

func TestMaterialListImportParserWorkbookContract(t *testing.T) {
	t.Run("formula and target mismatch", func(t *testing.T) {
		book := materialListImportBook(t)
		setImportRow(t, book, 2, []interface{}{"FABRIC", "Item", "", "1", "PCS", "1", "", "NONE", "", "WHOLE_WO", "", ""})
		if err := book.SetCellFormula("IMPORT", "D2", "1+1"); err != nil {
			t.Fatal(err)
		}
		if err := book.SetCellValue("_META", "B1", 99); err != nil {
			t.Fatal(err)
		}
		_, validation := parseMaterialListImportWorkbook(42, materialListImportBytes(t, book))
		if !materialListImportHasCode(validation, "formula_not_allowed") || !materialListImportHasCode(validation, "target_material_list_mismatch") {
			t.Fatalf("errors = %+v", validation.Errors)
		}
	})
	t.Run("gap and trailing blanks", func(t *testing.T) {
		book := materialListImportBook(t)
		valid := []interface{}{"FABRIC", "Item", "", "1", "PCS", "1", "", "NONE", "", "WHOLE_WO", "", ""}
		setImportRow(t, book, 2, valid)
		setImportRow(t, book, 4, valid)
		_, validation := parseMaterialListImportWorkbook(42, materialListImportBytes(t, book))
		if !materialListImportHasCode(validation, "malformed_row") {
			t.Fatalf("errors = %+v", validation.Errors)
		}
		book = materialListImportBook(t)
		setImportRow(t, book, 2, valid)
		_, validation = parseMaterialListImportWorkbook(42, materialListImportBytes(t, book))
		if validation.ErrorCount != 0 {
			t.Fatalf("trailing blanks rejected: %+v", validation.Errors)
		}
	})
}

func TestMaterialListImportDuplicateKeyUsesPersistedPrecision(t *testing.T) {
	cat, scope := "FABRIC", "WHOLE_WO"
	first := materialListImportKey(materialListImportRequest(cat, scope, 1), "1.20", "0.000")
	if first != materialListImportKey(materialListImportRequest(cat, scope, 1), "1.20", "0.000") {
		t.Fatal("identical tuple must match")
	}
	if first == materialListImportKey(materialListImportRequest(cat, scope, 2), "1.20", "0.000") {
		t.Fatal("quantity must be part of duplicate identity")
	}
	if materialListImportKey(materialListImportRequest(cat, scope, 1), "1.20", "") == materialListImportKey(materialListImportRequest(cat, scope, 1), "1.20", "0.000") {
		t.Fatal("nil consumption must differ from zero")
	}
	withControlInItem := materialListImportRequest(cat, scope, 1)
	withControlInItem.Item, withControlInItem.Description = "a\x1fb", "c"
	withControlInDescription := materialListImportRequest(cat, scope, 1)
	withControlInDescription.Item, withControlInDescription.Description = "a", "b\x1fc"
	if materialListImportKey(withControlInItem, "1.20", "0.000") == materialListImportKey(withControlInDescription, "1.20", "0.000") {
		t.Fatal("length-prefixed tuple must not collide on text delimiters")
	}
}

func TestMaterialListImportParserLimits(t *testing.T) {
	valid := []interface{}{"FABRIC", "Item", "", "1", "PCS", "1", "", "NONE", "", "WHOLE_WO", "", ""}
	book := materialListImportBook(t)
	for row := 2; row <= materialListImportMaxRows+1; row++ {
		setImportRow(t, book, row, valid)
	}
	reader := materialListImportBytes(t, book)
	if reader.Len() > materialListImportMaxBytes {
		t.Fatalf("test workbook exceeds core size limit: %d", reader.Len())
	}
	rows, validation := parseMaterialListImportWorkbook(42, reader)
	if validation.ErrorCount != 0 || len(rows) != materialListImportMaxRows {
		t.Fatalf("1000 rows = %d, validation = %+v", len(rows), validation.Errors)
	}

	book = materialListImportBook(t)
	for row := 2; row <= materialListImportMaxRows+2; row++ {
		setImportRow(t, book, row, valid)
	}
	_, validation = parseMaterialListImportWorkbook(42, materialListImportBytes(t, book))
	if !materialListImportHasCode(validation, "row_limit_exceeded") {
		t.Fatalf("1001 rows errors = %+v", validation.Errors)
	}
	_, validation = parseMaterialListImportWorkbook(42, bytes.NewReader(make([]byte, materialListImportMaxBytes+1)))
	if !materialListImportHasCode(validation, "invalid_workbook") {
		t.Fatalf("oversize reader errors = %+v", validation.Errors)
	}
}

func TestMaterialListCreateHelperRetainsOrdinaryPrefillSemantics(t *testing.T) {
	stub := &materialListCreateStub{prefill: numericFromDecimal("1.250")}
	category, scope := "FABRIC", "WHOLE_WO"
	shell := int32(7)
	_, err := createMaterialListItem(context.Background(), stub, 42, model.CreateMaterialListItemBody{
		Item: "Item", Unit: "PCS", EstPrice: 12.5, Category: &category, QtyWoScope: &scope, IDWoShell: &shell,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !stub.created.Valid || canonicalNumeric(stub.created, 3) != "1.250" || !stub.params.IDWoShell.Valid {
		t.Fatalf("ordinary create parameters changed: %+v", stub.params)
	}
}

type materialListCreateStub struct {
	prefill, created pgtype.Numeric
	params           entity.CreateMaterialListItemParams
}

func (s *materialListCreateStub) GetMaterialList(context.Context, int32) (entity.GetMaterialListRow, error) {
	return entity.GetMaterialListRow{}, nil
}
func (s *materialListCreateStub) CreateMaterialListItem(_ context.Context, p entity.CreateMaterialListItemParams) (entity.CreateMaterialListItemRow, error) {
	s.params = p
	s.created = p.ConsPerPc
	return entity.CreateMaterialListItemRow{IDMaterialListItem: 1, IDMaterialList: p.IDMaterialList, Item: p.Item, Description: p.Description, Qty: p.Qty, Unit: p.Unit, EstPrice: p.EstPrice, IDWoShell: p.IDWoShell, IDWoTrim: p.IDWoTrim, Category: p.Category, ConsPerPc: p.ConsPerPc, QtyWoScope: p.QtyWoScope, IDQtyWoShell: p.IDQtyWoShell, IDQtyWoSize: p.IDQtyWoSize, CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true}}, nil
}
func (s *materialListCreateStub) GetMaterialListItemConsumptionPrefill(context.Context, entity.GetMaterialListItemConsumptionPrefillParams) (pgtype.Numeric, error) {
	return s.prefill, nil
}
func (s *materialListCreateStub) MaterialListHasSourceShell(context.Context, entity.MaterialListHasSourceShellParams) (bool, error) {
	return true, nil
}
func (s *materialListCreateStub) MaterialListHasSourceTrim(context.Context, entity.MaterialListHasSourceTrimParams) (bool, error) {
	return true, nil
}
func (s *materialListCreateStub) MaterialListHasApplicabilityShell(context.Context, entity.MaterialListHasApplicabilityShellParams) (bool, error) {
	return true, nil
}
func (s *materialListCreateStub) MaterialListHasApplicabilitySize(context.Context, entity.MaterialListHasApplicabilitySizeParams) (bool, error) {
	return true, nil
}
func (s *materialListCreateStub) MaterialListHasApplicabilityShellSize(context.Context, entity.MaterialListHasApplicabilityShellSizeParams) (bool, error) {
	return true, nil
}

func materialListImportRequest(category, scope string, qty int32) model.CreateMaterialListItemBody {
	return model.CreateMaterialListItemBody{Item: "Item", Description: "", Qty: qty, Unit: "PCS", Category: &category, QtyWoScope: &scope}
}

func materialListImportBook(t *testing.T) *excelize.File {
	t.Helper()
	book := excelize.NewFile()
	if err := book.SetSheetName("Sheet1", "IMPORT"); err != nil {
		t.Fatal(err)
	}
	if _, err := book.NewSheet("REFERENSI"); err != nil {
		t.Fatal(err)
	}
	if _, err := book.NewSheet("_META"); err != nil {
		t.Fatal(err)
	}
	setImportRow(t, book, 1, stringInterfaces(materialListImportHeaders))
	if err := book.SetCellValue("_META", "A1", materialListImportMarker); err != nil {
		t.Fatal(err)
	}
	if err := book.SetCellValue("_META", "B1", 42); err != nil {
		t.Fatal(err)
	}
	return book
}

func setImportRow(t *testing.T, book *excelize.File, row int, values []interface{}) {
	t.Helper()
	if err := book.SetSheetRow("IMPORT", "A"+strconv.Itoa(row), &values); err != nil {
		t.Fatal(err)
	}
}
func materialListImportBytes(t *testing.T, book *excelize.File) *bytes.Reader {
	t.Helper()
	buffer, err := book.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	return bytes.NewReader(buffer.Bytes())
}
func stringInterfaces(values []string) []interface{} {
	result := make([]interface{}, len(values))
	for i, v := range values {
		result[i] = v
	}
	return result
}
func materialListImportHasCode(v *MaterialListExcelImportValidationError, code string) bool {
	for _, err := range v.Errors {
		if err.Code == code {
			return true
		}
	}
	return false
}
