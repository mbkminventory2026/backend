package usecase

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"

	"permatatex-inventory/internal/model"
	"permatatex-inventory/pkg/exporter/excel"
)

func TestSpreadingCuttingPlanExcelExportDynamicLayoutAndFormulas(t *testing.T) {
	export := spreadingCuttingPlanExcelFixture(8)
	file, err := renderSpreadingCuttingPlanExcel(spreadingCuttingPlanTestRenderer(t), export)
	if err != nil {
		t.Fatal(err)
	}
	if file.ContentType != excelContentType || len(file.Content) == 0 || !strings.HasSuffix(file.FileName, ".xlsx") {
		t.Fatalf("exported file = %+v", file)
	}
	book, err := excelize.OpenReader(bytes.NewReader(file.Content))
	if err != nil {
		t.Fatal(err)
	}
	defer book.Close()
	sheet := spreadingCuttingPlanExportSheetName
	spreadingCuttingPlanAssertCell(t, book, sheet, "A5", "SPREADING & CUTTING PLAN")
	spreadingCuttingPlanAssertCell(t, book, sheet, "N9", "SIZE-8")
	spreadingCuttingPlanAssertCell(t, book, sheet, "O8", "TOTAL")
	spreadingCuttingPlanAssertCell(t, book, sheet, "P8", "PLAN SPREADING")
	spreadingCuttingPlanAssertCell(t, book, sheet, "X8", "NOTE")
	spreadingCuttingPlanAssertCell(t, book, sheet, "D9", "1%")
	spreadingCuttingPlanAssertCell(t, book, sheet, "A10", "PO EMERALD")
	spreadingCuttingPlanAssertCell(t, book, sheet, "A13", "Sub Total EMERALD")
	spreadingCuttingPlanAssertCell(t, book, sheet, "A15", "TOTAL QTY PO")
	spreadingCuttingPlanAssertCell(t, book, sheet, "A16", "TOTAL QTY CUT PLAN")
	spreadingCuttingPlanAssertCell(t, book, sheet, "A17", "BALANCE")
	spreadingCuttingPlanAssertCell(t, book, sheet, "P12", "2.5 LAYER")
	if raw, err := book.GetCellValue(sheet, "P12", excelize.Options{RawCellValue: true}); err != nil || raw != "2.5" {
		t.Fatalf("P12 raw numeric value = %q, err=%v, want 2.5", raw, err)
	}
	spreadingCuttingPlanAssertFont(t, book, sheet, "P12", "Calibri", 11, true, "")
	spreadingCuttingPlanAssertHorizontalAlignment(t, book, sheet, "P12", "center")
	spreadingCuttingPlanAssertNumberFormat(t, book, sheet, "P12", `#,##0.### "LAYER"`)

	formulas := map[string]string{
		"O10": "SUM(G10:N10)",
		"D11": "C11*0.01",
		"E11": "C11+D11",
		"G12": "ROUND(G11*P12,0)",
		"O12": "SUM(G12:N12)",
		"Q12": "O12*E11",
		"T12": "R12*S12",
		"U12": "Q12+T12",
		"W12": "B11-U12-V12",
		"G17": "G16-G15",
	}
	for cell, want := range formulas {
		got, err := book.GetCellFormula(sheet, cell)
		if err != nil || got != want {
			t.Errorf("%s formula = %q, err=%v, want %q", cell, got, err, want)
		}
	}
	if calculated, err := book.CalcCellValue(sheet, "G12"); err != nil || calculated != "3" {
		t.Errorf("G12 calculated = %q, err=%v", calculated, err)
	}
	if calculated, err := book.CalcCellValue(sheet, "W12"); err != nil || calculated != "944.52" {
		t.Errorf("W12 calculated = %q, err=%v", calculated, err)
	}
	if !spreadingCuttingPlanHasMerge(book, sheet, "G8", "N8") || !spreadingCuttingPlanHasMerge(book, sheet, "A15", "F15") {
		t.Fatalf("expected dynamic/reference merges are missing: %+v", mustSpreadingCuttingPlanMerges(book, sheet))
	}
	for _, merge := range mustSpreadingCuttingPlanMerges(book, sheet) {
		if merge.GetStartAxis() == "A5" {
			t.Fatalf("title must not be merged across the table: %s:%s", merge.GetStartAxis(), merge.GetEndAxis())
		}
	}
	spreadingCuttingPlanAssertHorizontalAlignment(t, book, sheet, "A5", "left")
	for _, cell := range []string{"A11", "G11", "P12", "X12"} {
		spreadingCuttingPlanAssertBorder(t, book, sheet, cell, "92D050", 1)
	}
	if layout, err := book.GetPageLayout(sheet); err != nil || layout.Orientation == nil || *layout.Orientation != "landscape" || layout.FitToWidth == nil || *layout.FitToWidth != 1 {
		t.Fatalf("page layout = %+v, err=%v", layout, err)
	}
	if !spreadingCuttingPlanHasPrintArea(book, "'SCP'!$A$1:$X$17") {
		t.Fatalf("defined names = %+v", book.GetDefinedName())
	}
	if spreadingCuttingPlanFill(t, book, sheet, "G8") != "92D050" || spreadingCuttingPlanFill(t, book, sheet, "G10") != "1F4E78" || spreadingCuttingPlanFill(t, book, sheet, "G13") != "1F4E78" || spreadingCuttingPlanFill(t, book, sheet, "G15") != "92D050" {
		t.Fatal("reference green/accent/subtotal/total fills were not preserved")
	}
	if extent := spreadingCuttingPlanWorksheetCellExtent(t, file.Content); extent != "A1:X17" {
		t.Fatalf("worksheet cell extent = %q, want compact dynamic area A1:X17", extent)
	}
	if pictures, err := book.GetPictures(sheet, "A1"); err != nil || len(pictures) != 1 {
		t.Fatalf("PERMATATEX logo at A1 = %d pictures, err=%v", len(pictures), err)
	}
}

func TestSpreadingCuttingPlanExcelExportReferenceColorAndTypographyFidelity(t *testing.T) {
	export, err := composeSpreadingCuttingPlanExport(spreadingCuttingPlanExportFixture())
	if err != nil {
		t.Fatal(err)
	}
	file, err := renderSpreadingCuttingPlanExcel(spreadingCuttingPlanTestRenderer(t), export)
	if err != nil {
		t.Fatal(err)
	}
	book, err := excelize.OpenReader(bytes.NewReader(file.Content))
	if err != nil {
		t.Fatal(err)
	}
	defer book.Close()

	for _, cell := range []string{"A10", "G10", "A15", "G15"} {
		if fill := spreadingCuttingPlanFill(t, book, "SCP", cell); fill != "1E4E79" {
			t.Errorf("NAVY %s fill = %s, want 1E4E79", cell, fill)
		}
		spreadingCuttingPlanAssertFont(t, book, "SCP", cell, "Calibri", 11, true, "FFFFFF")
		spreadingCuttingPlanAssertHorizontalAlignment(t, book, "SCP", cell, "center")
	}
	for _, cell := range []string{"A16", "G16", "A19", "G19"} {
		if fill := spreadingCuttingPlanFill(t, book, "SCP", cell); fill != "000000" {
			t.Errorf("BLACK %s fill = %s, want 000000", cell, fill)
		}
		spreadingCuttingPlanAssertFont(t, book, "SCP", cell, "Calibri", 11, true, "FFFFFF")
		spreadingCuttingPlanAssertHorizontalAlignment(t, book, "SCP", cell, "center")
	}
	if fill := spreadingCuttingPlanShellFillColor("MAROON"); fill != "8E0407" {
		t.Fatalf("MAROON fill = %s, want reference 8E0407", fill)
	}
	if fill := spreadingCuttingPlanShellFillColor("unmapped color"); fill != "1F4E78" {
		t.Fatalf("fallback fill = %s, want corporate navy 1F4E78", fill)
	}

	for _, cell := range []string{"A8", "G8", "A9", "G9", "A11", "G11", "A21", "G21"} {
		spreadingCuttingPlanAssertFont(t, book, "SCP", cell, "Calibri", 11, strings.HasPrefix(cell, "A8") || strings.HasPrefix(cell, "G8") || strings.HasPrefix(cell, "A9") || strings.HasPrefix(cell, "G9") || strings.HasPrefix(cell, "A21") || strings.HasPrefix(cell, "G21"), "")
		spreadingCuttingPlanAssertHorizontalAlignment(t, book, "SCP", cell, "center")
	}
	spreadingCuttingPlanAssertFont(t, book, "SCP", "A5", "Calibri", 18, true, "000000")
	for row, want := range map[int]float64{8: 15, 9: 30, 10: 16.5, 11: 16.5, 15: 16.5, 21: 16.5} {
		got, err := book.GetRowHeight("SCP", row)
		if err != nil || got != want {
			t.Errorf("row %d height = %g, err=%v, want %g", row, got, err, want)
		}
	}
	spreadingCuttingPlanAssertCell(t, book, "SCP", "A21", "TOTAL QTY PO")
	spreadingCuttingPlanAssertCell(t, book, "SCP", "A22", "TOTAL QTY CUT PLAN")
	spreadingCuttingPlanAssertCell(t, book, "SCP", "A23", "BALANCE")
	if extent := spreadingCuttingPlanWorksheetCellExtent(t, file.Content); extent != "A1:T23" {
		t.Fatalf("worksheet cell extent = %q, want compact four-size area A1:T23", extent)
	}
	rows, err := book.GetRows("SCP")
	if err != nil {
		t.Fatal(err)
	}
	for rowIndex, row := range rows {
		for columnIndex, value := range row {
			if value == "62" {
				t.Fatalf("literal placeholder 62 at row %d column %d", rowIndex+1, columnIndex+1)
			}
		}
	}
}

func TestSpreadingCuttingPlanExcelExportMixedAllowanceAndBlankValues(t *testing.T) {
	export := spreadingCuttingPlanExcelFixture(3)
	export.AllowanceHeader = "ALLOW."
	export.Shells[0].Ratios[0].AllowancePercent = 2
	export.Shells[0].Ratios[0].AllowanceConsumption = 0.04
	export.Shells[0].Ratios[0].ConsPerPC = 2.04
	zero := int32(0)
	zeroPlanned := int64(0)
	export.Shells[0].Ratios[0].RatioPlan[1] = &zero
	export.Shells[0].Ratios[0].PlannedQty[1] = &zeroPlanned
	export.Shells[0].Ratios[0].RatioPlan[2] = nil
	export.Shells[0].Ratios[0].PlannedQty[2] = nil
	export.Shells[0].ReceivedActual = nil
	export.Shells[0].Ratios[0].RunningBalance = nil
	export.Shells[0].Subtotal.Balance = nil

	file, err := renderSpreadingCuttingPlanExcel(spreadingCuttingPlanTestRenderer(t), export)
	if err != nil {
		t.Fatal(err)
	}
	book, err := excelize.OpenReader(bytes.NewReader(file.Content))
	if err != nil {
		t.Fatal(err)
	}
	defer book.Close()
	spreadingCuttingPlanAssertCell(t, book, "SCP", "D9", "ALLOW.")
	spreadingCuttingPlanAssertCell(t, book, "SCP", "H11", "0")
	spreadingCuttingPlanAssertCell(t, book, "SCP", "I11", "")
	spreadingCuttingPlanAssertCell(t, book, "SCP", "B11", "")
	if formula, _ := book.GetCellFormula("SCP", "D11"); formula != "C11*0.02" {
		t.Fatalf("mixed allowance formula = %q", formula)
	}
	if formula, _ := book.GetCellFormula("SCP", "H12"); formula != "ROUND(H11*K12,0)" {
		t.Fatalf("explicit zero planned formula = %q", formula)
	}
	if formula, _ := book.GetCellFormula("SCP", "I12"); formula != "" {
		t.Fatalf("unavailable planned cell formula = %q, want blank", formula)
	}
}

func TestSpreadingCuttingPlanTemplateIsSanitized(t *testing.T) {
	templatePath := filepath.Join("..", "..", "templates", "exports", "xlsx", "template_spreading_cutting_plan.xlsx")
	templateContent, err := os.ReadFile(templatePath)
	if err != nil {
		t.Fatal(err)
	}
	if dimension := spreadingCuttingPlanWorksheetDimension(t, templateContent); dimension != "A5" {
		t.Fatalf("sanitized template dimension = %q, want A5", dimension)
	}
	book, err := excelize.OpenFile(templatePath)
	if err != nil {
		t.Fatal(err)
	}
	defer book.Close()
	if sheets := book.GetSheetList(); len(sheets) != 1 || sheets[0] != "SCP" {
		t.Fatalf("template sheets = %v", sheets)
	}
	rows, err := book.GetRows("SCP")
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(spreadingCuttingPlanFlattenRows(rows), " ")
	for _, forbidden := range []string{"CV SUHO", "DRESSLIM", "NAVY", "BLACK", "MAROON", "FRAPUCINNO"} {
		if strings.Contains(strings.ToUpper(joined), forbidden) {
			t.Fatalf("sanitized template contains sample value %q", forbidden)
		}
	}
	if joined != "SPREADING & CUTTING PLAN" {
		t.Fatalf("template reusable text = %q", joined)
	}
	for rowIndex, row := range rows {
		for columnIndex, value := range row {
			if value == "62" {
				t.Fatalf("sanitized template contains literal placeholder 62 at row %d column %d", rowIndex+1, columnIndex+1)
			}
		}
	}
	if pictures, err := book.GetPictures("SCP", "A1"); err != nil || len(pictures) != 1 {
		t.Fatalf("sanitized template logo = %d pictures, err=%v", len(pictures), err)
	}
}

func spreadingCuttingPlanExcelFixture(sizeCount int) *model.SpreadingCuttingPlanExport {
	ratioTextParts := make([]string, sizeCount)
	export := &model.SpreadingCuttingPlanExport{
		Header: model.SpreadingCuttingPlanExportHeader{
			DocumentNumber: "SCP-001", Buyer: "Buyer", EffectiveDate: time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC),
			Style: "Style", Model: "Model", Colors: []string{"EMERALD"},
		},
		AllowanceHeader: "1%",
	}
	shell := model.SpreadingCuttingPlanExportShell{ShellID: 10, Color: "EMERALD"}
	received := float64(1000)
	shell.ReceivedActual = &received
	ratio := model.SpreadingCuttingPlanExportRatio{
		RatioID: 1, Sequence: 1, Cons: 2, AllowancePercent: 1, AllowanceConsumption: 0.02,
		ConsPerPC: 2.02, PlanSpreadingGelaran: 2.5, RollQty: 2, JoinedRoll: 3,
		SambunganYard: 6, Reject: 1, Note: "note",
	}
	for index := 0; index < sizeCount; index++ {
		export.Sizes = append(export.Sizes, model.SpreadingCuttingPlanExportSize{SizeID: int32(index + 1), Label: fmt.Sprintf("SIZE-%d", index+1)})
		shell.OrderQty = append(shell.OrderQty, 10)
		shell.OrderTotal += 10
		ratioPlan := int32(1)
		planned := int64(3)
		ratio.RatioPlan = append(ratio.RatioPlan, &ratioPlan)
		ratio.PlannedQty = append(ratio.PlannedQty, &planned)
		ratio.PlannedTotal += planned
		shell.Subtotal.PlannedQty = append(shell.Subtotal.PlannedQty, &planned)
		orderTotal := int64(10)
		export.Totals.OrderQty = append(export.Totals.OrderQty, orderTotal)
		export.Totals.PlannedQty = append(export.Totals.PlannedQty, &planned)
		balance := planned - orderTotal
		export.Totals.BalanceQty = append(export.Totals.BalanceQty, &balance)
		ratioTextParts[index] = "1"
	}
	ratioText := strings.Join(ratioTextParts, "-")
	export.Header.RatioPO = &ratioText
	ratio.SubConsumption = float64(ratio.PlannedTotal) * ratio.ConsPerPC
	ratio.TotalConsumption = ratio.SubConsumption + ratio.SambunganYard
	running := received - ratio.TotalConsumption - ratio.Reject
	ratio.RunningBalance = &running
	shell.Ratios = []model.SpreadingCuttingPlanExportRatio{ratio}
	shell.Subtotal.PlannedTotal = ratio.PlannedTotal
	shell.Subtotal.SubConsumption = ratio.SubConsumption
	shell.Subtotal.SambunganYard = ratio.SambunganYard
	shell.Subtotal.TotalConsumption = ratio.TotalConsumption
	shell.Subtotal.Reject = ratio.Reject
	shell.Subtotal.Balance = &running
	export.Shells = []model.SpreadingCuttingPlanExportShell{shell}
	export.Totals.OrderTotal = shell.OrderTotal
	export.Totals.PlannedTotal = ratio.PlannedTotal
	export.Totals.BalanceTotal = export.Totals.PlannedTotal - export.Totals.OrderTotal
	return export
}

func spreadingCuttingPlanTestRenderer(t *testing.T) *excel.Renderer {
	t.Helper()
	renderer, err := excel.NewRenderer(filepath.Join("..", "..", "templates", "exports"))
	if err != nil {
		t.Fatal(err)
	}
	return renderer
}

func spreadingCuttingPlanAssertCell(t *testing.T, book *excelize.File, sheet, cell, want string) {
	t.Helper()
	got, err := book.GetCellValue(sheet, cell)
	if err != nil || got != want {
		t.Errorf("%s value = %q, err=%v, want %q", cell, got, err, want)
	}
}

func spreadingCuttingPlanHasMerge(book *excelize.File, sheet, start, end string) bool {
	for _, merge := range mustSpreadingCuttingPlanMerges(book, sheet) {
		if merge.GetStartAxis() == start && merge.GetEndAxis() == end {
			return true
		}
	}
	return false
}

func mustSpreadingCuttingPlanMerges(book *excelize.File, sheet string) []excelize.MergeCell {
	merges, _ := book.GetMergeCells(sheet)
	return merges
}

func spreadingCuttingPlanHasPrintArea(book *excelize.File, want string) bool {
	for _, name := range book.GetDefinedName() {
		if name.Name == "_xlnm.Print_Area" && name.Scope == "SCP" && name.RefersTo == want {
			return true
		}
	}
	return false
}

func spreadingCuttingPlanFill(t *testing.T, book *excelize.File, sheet, cell string) string {
	t.Helper()
	styleID, err := book.GetCellStyle(sheet, cell)
	if err != nil {
		t.Fatal(err)
	}
	style, err := book.GetStyle(styleID)
	if err != nil {
		t.Fatal(err)
	}
	if len(style.Fill.Color) == 0 {
		return ""
	}
	return style.Fill.Color[0]
}

func spreadingCuttingPlanAssertFont(t *testing.T, book *excelize.File, sheet, cell, family string, size float64, bold bool, color string) {
	t.Helper()
	styleID, err := book.GetCellStyle(sheet, cell)
	if err != nil {
		t.Fatal(err)
	}
	style, err := book.GetStyle(styleID)
	if err != nil {
		t.Fatal(err)
	}
	if style.Font == nil || style.Font.Family != family || style.Font.Size != size || style.Font.Bold != bold {
		t.Errorf("%s font = %+v, want %s/%g bold=%v", cell, style.Font, family, size, bold)
	}
	if color != "" && style.Font.Color != color {
		t.Errorf("%s font color = %s, want %s", cell, style.Font.Color, color)
	}
}

func spreadingCuttingPlanAssertHorizontalAlignment(t *testing.T, book *excelize.File, sheet, cell, want string) {
	t.Helper()
	styleID, err := book.GetCellStyle(sheet, cell)
	if err != nil {
		t.Fatal(err)
	}
	style, err := book.GetStyle(styleID)
	if err != nil {
		t.Fatal(err)
	}
	if style.Alignment == nil || style.Alignment.Horizontal != want {
		t.Errorf("%s alignment = %+v, want horizontal %s", cell, style.Alignment, want)
	}
}

func spreadingCuttingPlanAssertNumberFormat(t *testing.T, book *excelize.File, sheet, cell, want string) {
	t.Helper()
	styleID, err := book.GetCellStyle(sheet, cell)
	if err != nil {
		t.Fatal(err)
	}
	style, err := book.GetStyle(styleID)
	if err != nil {
		t.Fatal(err)
	}
	if style.CustomNumFmt == nil || *style.CustomNumFmt != want {
		t.Errorf("%s number format = %v, want %q", cell, style.CustomNumFmt, want)
	}
}

func spreadingCuttingPlanAssertBorder(t *testing.T, book *excelize.File, sheet, cell, color string, lineStyle int) {
	t.Helper()
	styleID, err := book.GetCellStyle(sheet, cell)
	if err != nil {
		t.Fatal(err)
	}
	style, err := book.GetStyle(styleID)
	if err != nil {
		t.Fatal(err)
	}
	if len(style.Border) != 4 {
		t.Fatalf("%s borders = %+v, want four edges", cell, style.Border)
	}
	for _, border := range style.Border {
		if border.Color != color || border.Style != lineStyle {
			t.Errorf("%s %s border = color %s style %d, want color %s style %d", cell, border.Type, border.Color, border.Style, color, lineStyle)
		}
	}
}

func spreadingCuttingPlanWorksheetDimension(t *testing.T, content []byte) string {
	t.Helper()
	xmlContent := spreadingCuttingPlanWorksheetXML(t, content)
	const marker = `<dimension ref="`
	start := strings.Index(xmlContent, marker)
	if start < 0 {
		t.Fatal("worksheet dimension is missing")
	}
	start += len(marker)
	end := strings.Index(xmlContent[start:], `"`)
	if end < 0 {
		t.Fatal("worksheet dimension is malformed")
	}
	return xmlContent[start : start+end]
}

func spreadingCuttingPlanWorksheetCellExtent(t *testing.T, content []byte) string {
	t.Helper()
	xmlContent := spreadingCuttingPlanWorksheetXML(t, content)
	matches := regexp.MustCompile(`<c\s[^>]*r="([A-Z]+[0-9]+)"`).FindAllStringSubmatch(xmlContent, -1)
	if len(matches) == 0 {
		t.Fatal("worksheet has no cells")
	}
	maxColumn, maxRow := 0, 0
	for _, match := range matches {
		column, row, err := excelize.CellNameToCoordinates(match[1])
		if err != nil {
			t.Fatal(err)
		}
		maxColumn = max(maxColumn, column)
		maxRow = max(maxRow, row)
	}
	end, err := excelize.CoordinatesToCellName(maxColumn, maxRow)
	if err != nil {
		t.Fatal(err)
	}
	return "A1:" + end
}

func spreadingCuttingPlanWorksheetXML(t *testing.T, content []byte) string {
	t.Helper()
	archive, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range archive.File {
		if entry.Name != "xl/worksheets/sheet1.xml" {
			continue
		}
		reader, err := entry.Open()
		if err != nil {
			t.Fatal(err)
		}
		xmlContent, err := io.ReadAll(reader)
		reader.Close()
		if err != nil {
			t.Fatal(err)
		}
		return string(xmlContent)
	}
	t.Fatal("worksheet XML is missing")
	return ""
}

func spreadingCuttingPlanFlattenRows(rows [][]string) []string {
	var values []string
	for _, row := range rows {
		for _, value := range row {
			if strings.TrimSpace(value) != "" {
				values = append(values, strings.TrimSpace(value))
			}
		}
	}
	return values
}
