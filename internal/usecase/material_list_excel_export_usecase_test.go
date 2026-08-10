package usecase

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"

	"permatatex-inventory/internal/model"
	"permatatex-inventory/pkg/exporter/excel"
)

func TestMaterialListExcelRenderMapsStage3Composition(t *testing.T) {
	renderer := materialListTestRenderer(t)
	export := materialListFixture()
	file, err := renderMaterialListExcel(renderer, export, "")
	if err != nil {
		t.Fatal(err)
	}
	if output := os.Getenv("PERMATATEX_MATERIAL_LIST_SAMPLE"); output != "" {
		if err := os.WriteFile(output, file.Content, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if file.FileName != "MATERIAL_LIST_Stage_4_Demo.xlsx" {
		t.Fatalf("filename = %q", file.FileName)
	}
	book, err := excelize.OpenReader(bytes.NewReader(file.Content))
	if err != nil {
		t.Fatal(err)
	}
	defer book.Close()
	if sheets := book.GetSheetList(); len(sheets) != 1 || sheets[0] != "Material List" {
		t.Fatalf("sheets = %v", sheets)
	}
	sheet := "Material List"
	for cell, want := range map[string]string{"A2": "BUYER", "B2": ":", "C2": "Buyer Test", "A3": "MODEL", "C3": "Model Test", "A4": "STYLE", "C4": "Style Test", "A5": "QTY", "C5": "99", "D5": "Pcs", "A6": "FOB/CMT", "C6": "CMT", "A7": "DELIVERY", "C7": "23-Jan-2026", "I2": "1", "J2": "0", "K2": "", "I4": "10", "I6": "0", "M4": "NAVY", "M7": "NAVY", "I13": "-13", "H15": "SJ SUHO", "A18": "Fabric", "O18": "ACTUAL 0", "O20": "CUTPLAN 12"} {
		if got, _ := book.GetCellValue(sheet, cell); got != want {
			t.Errorf("%s = %q, want %q", cell, got, want)
		}
	}
	// Nil stays physically unset while an authoritative zero remains a literal zero.
	for _, cell := range []string{"D2", "D3", "D4", "D6", "D7", "M10", "M11", "M12", "M13", "B18", "C18", "H20", "D24", "E24", "F24", "R24", "S24"} {
		materialListAssertUnsetCell(t, book, sheet, cell)
	}
	for cell, want := range map[string]string{"K18": "0", "P18": "0.00", "Q18": "0", "J2": "0"} {
		if got, _ := book.GetCellValue(sheet, cell); got != want {
			t.Errorf("blank/zero %s = %q, want %q", cell, got, want)
		}
	}
	materialListAssertValueCell(t, book, sheet, "H18", "4")
	for _, category := range []string{"FABRIC", "SEWING", "PACKING", "UNCATEGORIZED / REVIEW"} {
		if !materialListBookContains(book, sheet, category) {
			t.Errorf("missing category %q", category)
		}
	}
	if materialListBookContains(book, sheet, "Employee detail") {
		t.Error("supplemental description must not be rendered")
	}
	for _, row := range []int{17, 19, 21, 23} {
		for col := 2; col <= 19; col++ {
			cell := materialListCell(col, row)
			materialListAssertUnsetCell(t, book, sheet, cell)
		}
	}
	intentionalFours := map[string]bool{"H18": true, "J18": true}
	for rowIndex, values := range materialListRows(book, sheet) {
		for columnIndex, value := range values {
			if value != "4" {
				continue
			}
			cell := materialListCell(columnIndex+1, rowIndex+1)
			if !intentionalFours[cell] {
				t.Errorf("unexpected literal 4 at %s", cell)
			}
		}
	}
	for _, cell := range []string{"H2", "I2", "M2", "H6", "M10", "M13", "A15", "H15", "K15", "S15", "A17", "S17", "A18", "S18"} {
		if !materialListHasBlackGridBorder(t, book, sheet, cell) {
			t.Errorf("missing black grid border at %s", cell)
		}
	}
	if layout, err := book.GetPageLayout(sheet); err != nil || layout.Orientation == nil || *layout.Orientation != "landscape" || layout.FitToWidth == nil || *layout.FitToWidth != 1 {
		t.Errorf("page layout = %+v, err=%v", layout, err)
	}
	if merges, err := book.GetMergeCells(sheet); err != nil || len(merges) == 0 {
		t.Errorf("merged layout missing: %v", err)
	} else {
		for _, merge := range merges {
			if merge.GetStartAxis() == "M1" || merge.GetEndAxis() == "M1" {
				t.Errorf("title must not be merged: %s:%s", merge.GetStartAxis(), merge.GetEndAxis())
			}
		}
	}
	if got, _ := book.GetCellValue(sheet, "M1"); got != "MATERIAL LIST" {
		t.Errorf("dynamic title = %q", got)
	}
	for cell, want := range map[string]string{"M1": "", "A15": "", "H15": "C5E0B3", "K15": "BDD6EE", "O15": "F7CAAC", "R15": "FFFF00", "S15": "F7CAAC", "H2": "A8D08D", "H6": "FFFF00", "A17": ""} {
		if got := materialListFill(t, book, sheet, cell); got != want {
			t.Errorf("%s fill = %q, want %q", cell, got, want)
		}
	}
	if titleStyle := materialListStyle(t, book, sheet, "M1"); titleStyle.Font == nil || titleStyle.Font.Family != "Algerian" || titleStyle.Font.Size != 18 || !titleStyle.Font.Bold || titleStyle.Alignment == nil || titleStyle.Alignment.Horizontal != "right" {
		t.Errorf("title style = %#v", titleStyle)
	}
}

func TestMaterialListExcelTemplateIsCleanAndLogoIsRuntimeOnly(t *testing.T) {
	renderer := materialListTestRenderer(t)
	book, err := renderer.OpenTemplate(materialListExportTemplateName)
	if err != nil {
		t.Fatal(err)
	}
	defer book.Close()
	if sheets := book.GetSheetList(); len(sheets) != 1 {
		t.Fatalf("template sheets = %v", sheets)
	}
	for _, sample := range []string{"CV SUHO GARMINDO", "DRESSLIM", "SHAKILA", "18-Dec"} {
		if materialListBookContains(book, book.GetSheetName(0), sample) {
			t.Fatalf("template leaked reference value %q", sample)
		}
	}
	if pictures, _ := book.GetPictures(book.GetSheetName(0), "A1"); len(pictures) != 0 {
		t.Fatalf("template contains static logo")
	}
	var imageBytes bytes.Buffer
	picture := image.NewRGBA(image.Rect(0, 0, 2, 2))
	picture.Set(0, 0, color.RGBA{R: 0, G: 175, B: 80, A: 255})
	if err := png.Encode(&imageBytes, picture); err != nil {
		t.Fatal(err)
	}
	file, err := renderMaterialListExcel(renderer, materialListFixture(), "data:image/png;base64,"+base64.StdEncoding.EncodeToString(imageBytes.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	rendered, err := excelize.OpenReader(bytes.NewReader(file.Content))
	if err != nil {
		t.Fatal(err)
	}
	defer rendered.Close()
	if pictures, err := rendered.GetPictures("Material List", "A1"); err != nil || len(pictures) != 1 {
		t.Fatalf("runtime logo pictures=%d err=%v", len(pictures), err)
	}
	if _, err := renderMaterialListExcel(renderer, materialListFixture(), "not an image"); err != nil {
		t.Fatalf("invalid logo should be ignored: %v", err)
	}
}

func TestMaterialListExcelSupportsNoTransactionDates(t *testing.T) {
	renderer := materialListTestRenderer(t)
	export := materialListFixture()
	export.SJDates = nil
	export.ReceivedDates = nil
	for i := range export.MaterialRows {
		export.MaterialRows[i].SJByDate = nil
		export.MaterialRows[i].ReceivedByDate = nil
		export.MaterialRows[i].SJTotal = nil
		export.MaterialRows[i].ReceivedTotal = nil
	}
	file, err := renderMaterialListExcel(renderer, export, "")
	if err != nil {
		t.Fatal(err)
	}
	book, err := excelize.OpenReader(bytes.NewReader(file.Content))
	if err != nil {
		t.Fatal(err)
	}
	defer book.Close()
	if got, _ := book.GetCellValue("Material List", "H15"); got != "SJ SUHO TOTAL" {
		t.Fatalf("no-date SJ header = %q", got)
	}
	if got, _ := book.GetCellValue("Material List", "I15"); got != "RCVD PERMATA TOTAL" {
		t.Fatalf("no-date RCVD header = %q", got)
	}
}

func TestBuildMaterialListExportFileName(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		want  string
	}{
		{name: "normal", input: "Material List 01", want: "MATERIAL_LIST_Material_List_01.xlsx"},
		{name: "unsafe characters", input: `A/B:C*D?E"F<G>H|I`, want: "MATERIAL_LIST_A_B_C_D_E_F_G_H_I.xlsx"},
		{name: "empty", input: "", want: "MATERIAL_LIST.xlsx"},
		{name: "whitespace", input: " \t ", want: "MATERIAL_LIST.xlsx"},
		{name: "sanitizes empty", input: "...", want: "MATERIAL_LIST.xlsx"},
	} {
		t.Run(test.name, func(t *testing.T) {
			export := &model.MaterialListExport{Header: model.MaterialListExportHeader{MaterialListName: test.input}}
			if got := buildMaterialListExportFileName(export); got != test.want {
				t.Fatalf("filename = %q, want %q", got, test.want)
			}
		})
	}
}

func materialListTestRenderer(t *testing.T) *excel.Renderer {
	t.Helper()
	renderer, err := excel.NewRenderer(filepath.Join("..", "..", "templates", "exports"))
	if err != nil {
		t.Fatal(err)
	}
	return renderer
}
func materialListBookContains(book *excelize.File, sheet, value string) bool {
	rows, _ := book.GetRows(sheet)
	for _, row := range rows {
		for _, cell := range row {
			if strings.Contains(cell, value) {
				return true
			}
		}
	}
	return false
}
func materialListRows(book *excelize.File, sheet string) [][]string {
	rows, _ := book.GetRows(sheet)
	return rows
}
func materialListAssertUnsetCell(t *testing.T, book *excelize.File, sheet, cell string) {
	t.Helper()
	if got, _ := book.GetCellValue(sheet, cell); got != "" {
		t.Errorf("%s value = %q, want blank", cell, got)
	}
	if cellType, _ := book.GetCellType(sheet, cell); cellType != excelize.CellTypeUnset {
		t.Errorf("%s type = %v, want unset", cell, cellType)
	}
}
func materialListAssertValueCell(t *testing.T, book *excelize.File, sheet, cell, want string) {
	t.Helper()
	if got, _ := book.GetCellValue(sheet, cell); got != want {
		t.Errorf("%s value = %q, want %q", cell, got, want)
	}
}
func materialListStyle(t *testing.T, book *excelize.File, sheet, cell string) *excelize.Style {
	t.Helper()
	styleID, err := book.GetCellStyle(sheet, cell)
	if err != nil {
		t.Fatal(err)
	}
	style, err := book.GetStyle(styleID)
	if err != nil {
		t.Fatal(err)
	}
	return style
}
func materialListFill(t *testing.T, book *excelize.File, sheet, cell string) string {
	t.Helper()
	style := materialListStyle(t, book, sheet, cell)
	if len(style.Fill.Color) == 0 {
		return ""
	}
	return style.Fill.Color[0]
}
func materialListHasBlackGridBorder(t *testing.T, book *excelize.File, sheet, cell string) bool {
	t.Helper()
	style := materialListStyle(t, book, sheet, cell)
	return len(style.Border) == 4 && style.Border[0].Color == "000000" && style.Border[1].Color == "000000" && style.Border[2].Color == "000000" && style.Border[3].Color == "000000"
}
func i64(v int64) *int64     { return &v }
func i32(v int32) *int32     { return &v }
func f64(v float64) *float64 { return &v }
func str(v string) *string   { return &v }

func materialListFixture() *model.MaterialListExport {
	d1, d2 := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC), time.Date(2026, 2, 6, 0, 0, 0, 0, time.UTC)
	return &model.MaterialListExport{Header: model.MaterialListExportHeader{MaterialListName: "Stage 4 Demo", Buyer: "Buyer Test", Model: "Model Test", Style: "Style Test", WorkOrderQty: 99, Delivery: time.Date(2026, 1, 23, 0, 0, 0, 0, time.UTC)}, DetailQty: model.MaterialListExportDetailQty{Sizes: []model.MaterialListExportSize{{ID: 1, Name: "XS", Ratio: i32(1)}, {ID: 2, Name: "M", Ratio: i32(0)}, {ID: 3, Name: "XL"}}, Shells: []model.MaterialListExportShellQty{{ShellID: 1, Color: "NAVY", Values: []model.MaterialListExportShellSizeQty{{Order: i64(10), MarkerPlan: i64(11), ActualCut: i64(0)}, {Order: i64(20), MarkerPlan: i64(22), ActualCut: nil}, {Order: nil, MarkerPlan: nil, ActualCut: nil}}}, {ShellID: 2, Color: "NAVY", Values: []model.MaterialListExportShellSizeQty{{Order: i64(9), MarkerPlan: i64(10), ActualCut: i64(8)}, {Order: i64(19), MarkerPlan: i64(20), ActualCut: i64(18)}, {Order: i64(3), MarkerPlan: i64(3), ActualCut: i64(2)}}}}, Summaries: []model.MaterialListExportSizeSummary{{OrderTotal: i64(19), RMSTotal: i64(21), TotalActCut: i64(8), Balance: i64(-13)}, {OrderTotal: i64(39), RMSTotal: i64(42), TotalActCut: i64(18), Balance: i64(-24)}, {OrderTotal: i64(3), RMSTotal: i64(3), TotalActCut: i64(2), Balance: i64(-1)}}}, SJDates: []time.Time{d1, d2}, ReceivedDates: []time.Time{d1, d2}, MaterialRows: []model.MaterialListExportRow{{Item: "Fabric", Description: "Employee detail", Category: str("FABRIC"), Unit: "YRD", ConsPerPC: f64(1.25), QtyWO: i64(10), Total: f64(12.5), SJByDate: map[time.Time]int64{d1: 4, d2: 0}, SJTotal: i64(4), ReceivedByDate: map[time.Time]int64{d1: 0, d2: 6}, ReceivedTotal: i64(6), BalanceReceived: i64(2), CuttingSource: model.MaterialListCuttingActual, CuttingQty: i64(0), ActualConsumption: f64(0), RejectRetur: i64(0), FinalBalance: f64(6)}, {Item: "Sewing", Category: str("SEWING"), Unit: "PCS", ConsPerPC: f64(1), QtyWO: i64(12), Total: f64(12), CuttingSource: model.MaterialListCuttingCutPlan, CuttingQty: i64(12), ActualConsumption: f64(12)}, {Item: "Packing", Category: str("PACKING"), Unit: "PCS", CuttingSource: model.MaterialListCuttingNone}, {Item: "Legacy", Category: nil, Unit: "PCS"}}}
}
