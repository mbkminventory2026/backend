package usecase

import (
	"bytes"
	"testing"

	"github.com/xuri/excelize/v2"
	"permatatex-inventory/internal/entity"
)

func TestWriteMaterialListImportTemplateContract(t *testing.T) {
	book := excelize.NewFile()
	defer book.Close()
	if err := book.SetSheetName("Sheet1", "IMPORT"); err != nil {
		t.Fatal(err)
	}
	if _, err := book.NewSheet("REFERENSI"); err != nil {
		t.Fatal(err)
	}
	if _, err := book.NewSheet("_META"); err != nil {
		t.Fatal(err)
	}
	if err := book.SetSheetVisible("_META", false); err != nil {
		t.Fatal(err)
	}
	if err := writeMaterialListImportTemplate(book, 7,
		[]entity.WorkOrderShell{{IDWoShell: 11, Color: "RED", Deskripsi: "shell"}},
		[]entity.WorkOrderTrim{{IDWoTrim: 21, Item: "zip", Color: "BLACK", Description: "trim"}},
		[]entity.ListWorkOrderShellSizesByWorkOrderIDRow{{IDWoShellSize: 999, IDWoShell: 11, IDSize: 31, Size: "M"}}); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := book.Write(&out); err != nil {
		t.Fatal(err)
	}
	got, err := excelize.OpenReader(bytes.NewReader(out.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	defer got.Close()
	sheets := got.GetSheetList()
	want := []string{"IMPORT", "REFERENSI", "_META"}
	if len(sheets) != len(want) {
		t.Fatalf("sheets=%v", sheets)
	}
	for i := range want {
		if sheets[i] != want[i] {
			t.Fatalf("sheets=%v", sheets)
		}
	}
	if visible, _ := got.GetSheetVisible("_META"); visible {
		t.Fatal("_META must be hidden")
	}
	if marker, _ := got.GetCellValue("_META", "A1"); marker != materialListImportMarker {
		t.Fatalf("marker=%q", marker)
	}
	if target, _ := got.GetCellValue("_META", "B1"); target != "7" {
		t.Fatalf("target=%q", target)
	}
	for i, header := range materialListImportHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		actual, _ := got.GetCellValue("IMPORT", cell)
		if actual != header {
			t.Fatalf("%s=%q", cell, actual)
		}
	}
	if rows, err := got.GetRows("IMPORT"); err != nil || len(rows) != 1 {
		t.Fatalf("unexpected import rows: %d %v", len(rows), err)
	}
	ref, _ := got.GetRows("REFERENSI")
	text := ""
	for _, row := range ref {
		for _, v := range row {
			text += v + "|"
		}
	}
	if !bytes.Contains([]byte(text), []byte("31")) || bytes.Contains([]byte(text), []byte("999")) {
		t.Fatalf("size reference must contain master size, not shell-size ID: %q", text)
	}
	if _, validation := parseMaterialListImportWorkbook(7, bytes.NewReader(out.Bytes())); validation.ErrorCount != 0 {
		t.Fatalf("template must satisfy Stage 6B parser: %+v", validation.Errors)
	}
}

func TestMaterialListImportTemplateFileName(t *testing.T) {
	for _, tc := range []struct{ name, want string }{
		{"List Utama", "MATERIAL_LIST_IMPORT_List_Utama.xlsx"},
		{"A/B:C?", "MATERIAL_LIST_IMPORT_A_B_C.xlsx"},
		{"", "MATERIAL_LIST_IMPORT.xlsx"}, {"   ", "MATERIAL_LIST_IMPORT.xlsx"}, {"////", "MATERIAL_LIST_IMPORT.xlsx"},
	} {
		if got := buildMaterialListImportTemplateFileName(tc.name); got != tc.want {
			t.Errorf("%q: got %q want %q", tc.name, got, tc.want)
		}
	}
}
