package usecase

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"permatatex-inventory/internal/model"
	"permatatex-inventory/pkg/exporter/excel"
)

const materialListExportTemplateName = "xlsx/template_material_list.xlsx"

// MaterialListExcelExportUseCase deliberately consumes the renderer-neutral
// Stage 3 composition. It does not read, calculate, or reconcile export data.
type MaterialListExcelExportUseCase struct {
	renderer *excel.Renderer
	exportUC *MaterialListExportUseCase
	profile  *ProfilPerusahaanUseCase
}

func NewMaterialListExcelExportUseCase(renderer *excel.Renderer, exportUC *MaterialListExportUseCase, profile *ProfilPerusahaanUseCase) (*MaterialListExcelExportUseCase, error) {
	if renderer == nil {
		return nil, errors.New("excel renderer is required")
	}
	if exportUC == nil {
		return nil, errors.New("material list export usecase is required")
	}
	if profile == nil {
		return nil, errors.New("profil perusahaan usecase is required")
	}
	return &MaterialListExcelExportUseCase{renderer: renderer, exportUC: exportUC, profile: profile}, nil
}

// ExportByID is intentionally not wired to HTTP in Stage 4.
func (u *MaterialListExcelExportUseCase) ExportByID(ctx context.Context, id int32) (*model.ExportedFile, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	composed, err := u.exportUC.Compose(ctx, id)
	if err != nil {
		return nil, err
	}
	profile, err := u.profile.GetProfilPerusahaan(ctx)
	if err != nil && !errors.Is(err, ErrProfilPerusahaanNotFound) {
		return nil, fmt.Errorf("get profil perusahaan: %w", err)
	}
	return renderMaterialListExcel(u.renderer, composed, profile.Logo)
}

type materialListExcelLayout struct {
	tableHeader int
	dataStart   int
	lastColumn  int
}

func renderMaterialListExcel(renderer *excel.Renderer, export *model.MaterialListExport, logo string) (*model.ExportedFile, error) {
	if renderer == nil || export == nil {
		return nil, errors.New("material list renderer and composition are required")
	}
	book, err := renderer.OpenTemplate(materialListExportTemplateName)
	if err != nil {
		return nil, fmt.Errorf("open material list export template: %w", err)
	}
	defer book.Close()
	sheet := book.GetSheetName(0)
	if sheet == "" {
		return nil, errors.New("material list template has no worksheet")
	}
	if len(book.GetSheetList()) != 1 {
		return nil, errors.New("material list template must contain one worksheet")
	}
	layout, err := writeMaterialListWorkbook(book, sheet, export, logo)
	if err != nil {
		return nil, err
	}
	if err := book.SetDefinedName(&excelize.DefinedName{Name: "_xlnm.Print_Area", RefersTo: fmt.Sprintf("'%s'!$A$1:$%s$%d", sheet, materialListColumn(layout.lastColumn), layout.dataStart+len(export.MaterialRows)), Scope: sheet}); err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := book.Write(&out); err != nil {
		return nil, fmt.Errorf("write material list workbook: %w", err)
	}
	return &model.ExportedFile{FileName: buildMaterialListExportFileName(export), ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", Content: out.Bytes()}, nil
}

func writeMaterialListWorkbook(book *excelize.File, sheet string, export *model.MaterialListExport, logo string) (materialListExcelLayout, error) {
	styles, err := newMaterialListExcelStyles(book)
	if err != nil {
		return materialListExcelLayout{}, err
	}
	lastCol := 24
	if detailLast := 10 + len(export.DetailQty.Sizes); detailLast > lastCol {
		lastCol = detailLast
	}
	if tableLast := 15 + len(export.SJDates) + len(export.ReceivedDates); tableLast > lastCol {
		lastCol = tableLast
	}
	if err := book.SetSheetView(sheet, 0, &excelize.ViewOptions{ShowGridLines: boolPtr(false)}); err != nil {
		return materialListExcelLayout{}, err
	}
	fitWidth, fitHeight := 1, 0
	landscape := "landscape"
	if err := book.SetPageLayout(sheet, &excelize.PageLayoutOptions{Orientation: &landscape, FitToWidth: &fitWidth, FitToHeight: &fitHeight}); err != nil {
		return materialListExcelLayout{}, err
	}
	if err := writeMaterialListHeader(book, sheet, export.Header, styles, 10+len(export.DetailQty.Sizes), logo); err != nil {
		return materialListExcelLayout{}, err
	}
	tableHeader, err := writeMaterialListDetailQty(book, sheet, export.DetailQty, styles)
	if err != nil {
		return materialListExcelLayout{}, err
	}
	lastCol, err = writeMaterialListTable(book, sheet, tableHeader, export, styles, lastCol)
	if err != nil {
		return materialListExcelLayout{}, err
	}
	return materialListExcelLayout{tableHeader: tableHeader, dataStart: tableHeader + 2, lastColumn: lastCol}, nil
}

type materialListExcelStyles struct {
	title, headerLabel, headerColon, headerValue                                       int
	detailHeader, detailBody, detailActual, detailSummaryBlue, detailSummaryYellow     int
	baseHeader, sjHeader, receivedHeader, operationalHeader, balanceHeader, noteHeader int
	sjDateHeader, receivedDateHeader, category, text, integer, decimal, balanceValue   int
}

func newMaterialListExcelStyles(book *excelize.File) (materialListExcelStyles, error) {
	black := "000000"
	thin := []excelize.Border{{Type: "left", Color: black, Style: 1}, {Type: "right", Color: black, Style: 1}, {Type: "top", Color: black, Style: 1}, {Type: "bottom", Color: black, Style: 1}}
	newStyle := func(s *excelize.Style) (int, error) { return book.NewStyle(s) }
	title, err := newStyle(&excelize.Style{Font: &excelize.Font{Family: "Algerian", Bold: true, Size: 18, Color: black}, Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"}})
	if err != nil {
		return materialListExcelStyles{}, err
	}
	headerLabel, err := newStyle(&excelize.Style{Font: &excelize.Font{Family: "Calibri", Bold: true, Size: 11, Color: black}, Alignment: &excelize.Alignment{Vertical: "center"}})
	if err != nil {
		return materialListExcelStyles{}, err
	}
	headerColon, err := newStyle(&excelize.Style{Font: &excelize.Font{Family: "Calibri", Bold: true, Size: 11, Color: black}, Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"}})
	if err != nil {
		return materialListExcelStyles{}, err
	}
	headerValue, err := newStyle(&excelize.Style{Font: &excelize.Font{Family: "Calibri", Bold: true, Size: 11, Color: black}, Alignment: &excelize.Alignment{Vertical: "center"}})
	if err != nil {
		return materialListExcelStyles{}, err
	}
	detailHeader, err := newStyle(&excelize.Style{Font: &excelize.Font{Family: "Calibri", Bold: true, Size: 11, Color: black}, Fill: excelize.Fill{Type: "pattern", Color: []string{"A8D08D"}, Pattern: 1}, Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"}, Border: thin})
	if err != nil {
		return materialListExcelStyles{}, err
	}
	detailBody, err := newStyle(&excelize.Style{Font: &excelize.Font{Family: "Calibri", Size: 11, Color: black}, Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"}, Border: thin})
	if err != nil {
		return materialListExcelStyles{}, err
	}
	detailActual, err := newStyle(&excelize.Style{Font: &excelize.Font{Family: "Calibri", Size: 11, Color: black}, Fill: excelize.Fill{Type: "pattern", Color: []string{"FFFF00"}, Pattern: 1}, Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"}, Border: thin})
	if err != nil {
		return materialListExcelStyles{}, err
	}
	detailSummaryBlue, err := newStyle(&excelize.Style{Font: &excelize.Font{Family: "Calibri", Bold: true, Size: 11, Color: black}, Fill: excelize.Fill{Type: "pattern", Color: []string{"BDD6EE"}, Pattern: 1}, Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"}, Border: thin})
	if err != nil {
		return materialListExcelStyles{}, err
	}
	detailSummaryYellow, err := newStyle(&excelize.Style{Font: &excelize.Font{Family: "Calibri", Bold: true, Size: 11, Color: black}, Fill: excelize.Fill{Type: "pattern", Color: []string{"FFFF00"}, Pattern: 1}, Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"}, Border: thin})
	if err != nil {
		return materialListExcelStyles{}, err
	}
	headerStyle := func(fill string) (int, error) {
		style := &excelize.Style{Font: &excelize.Font{Family: "Calibri", Bold: true, Size: 11, Color: black}, Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true}, Border: thin}
		if fill != "" {
			style.Fill = excelize.Fill{Type: "pattern", Color: []string{fill}, Pattern: 1}
		}
		return newStyle(style)
	}
	baseHeader, err := headerStyle("")
	if err != nil {
		return materialListExcelStyles{}, err
	}
	sjHeader, err := headerStyle("C5E0B3")
	if err != nil {
		return materialListExcelStyles{}, err
	}
	receivedHeader, err := headerStyle("BDD6EE")
	if err != nil {
		return materialListExcelStyles{}, err
	}
	operationalHeader, err := headerStyle("F7CAAC")
	if err != nil {
		return materialListExcelStyles{}, err
	}
	balanceHeader, err := headerStyle("FFFF00")
	if err != nil {
		return materialListExcelStyles{}, err
	}
	noteHeader, err := headerStyle("F7CAAC")
	if err != nil {
		return materialListExcelStyles{}, err
	}
	sjDateHeader, err := headerStyle("C5E0B3")
	if err != nil {
		return materialListExcelStyles{}, err
	}
	receivedDateHeader, err := headerStyle("BDD6EE")
	if err != nil {
		return materialListExcelStyles{}, err
	}
	category, err := newStyle(&excelize.Style{Font: &excelize.Font{Family: "Calibri", Bold: true, Size: 11, Color: black}, Alignment: &excelize.Alignment{Vertical: "center"}, Border: thin})
	if err != nil {
		return materialListExcelStyles{}, err
	}
	text, err := newStyle(&excelize.Style{Font: &excelize.Font{Family: "Calibri", Size: 11, Color: black}, Alignment: &excelize.Alignment{Vertical: "center", WrapText: true}, Border: thin})
	if err != nil {
		return materialListExcelStyles{}, err
	}
	integer, err := newStyle(&excelize.Style{Font: &excelize.Font{Family: "Calibri", Size: 11, Color: black}, Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"}, Border: thin, NumFmt: 1})
	if err != nil {
		return materialListExcelStyles{}, err
	}
	decimal, err := newStyle(&excelize.Style{Font: &excelize.Font{Family: "Calibri", Size: 11, Color: black}, Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"}, Border: thin, NumFmt: 4})
	if err != nil {
		return materialListExcelStyles{}, err
	}
	balanceValue, err := newStyle(&excelize.Style{Font: &excelize.Font{Family: "Calibri", Size: 11, Color: black}, Fill: excelize.Fill{Type: "pattern", Color: []string{"FFFF00"}, Pattern: 1}, Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"}, Border: thin, NumFmt: 4})
	if err != nil {
		return materialListExcelStyles{}, err
	}
	return materialListExcelStyles{title, headerLabel, headerColon, headerValue, detailHeader, detailBody, detailActual, detailSummaryBlue, detailSummaryYellow, baseHeader, sjHeader, receivedHeader, operationalHeader, balanceHeader, noteHeader, sjDateHeader, receivedDateHeader, category, text, integer, decimal, balanceValue}, nil
}

func writeMaterialListHeader(book *excelize.File, sheet string, header model.MaterialListExportHeader, s materialListExcelStyles, titleColumn int, logo string) error {
	titleCell := materialListCell(titleColumn, 1)
	if err := book.SetCellValue(sheet, titleCell, "MATERIAL LIST"); err != nil {
		return err
	}
	if err := book.SetCellStyle(sheet, titleCell, titleCell, s.title); err != nil {
		return err
	}
	for row, field := range []struct {
		label string
		value any
		unit  any
	}{{"BUYER", strings.TrimSpace(header.Buyer), nil}, {"MODEL", strings.TrimSpace(header.Model), nil}, {"STYLE", strings.TrimSpace(header.Style), nil}, {"QTY", fmt.Sprintf("%d", header.WorkOrderQty), "Pcs"}, {"FOB/CMT", materialListFOBCMT(header.FobCmt), nil}, {"DELIVERY", materialListDate(header.Delivery), nil}} {
		r := row + 2
		for _, cell := range []struct {
			column string
			value  any
			style  int
		}{{"A", field.label, s.headerLabel}, {"B", ":", s.headerColon}, {"C", field.value, s.headerValue}, {"D", field.unit, s.headerValue}} {
			if err := book.SetCellValue(sheet, fmt.Sprintf("%s%d", cell.column, r), materialListValue(cell.value)); err != nil {
				return err
			}
			if err := book.SetCellStyle(sheet, fmt.Sprintf("%s%d", cell.column, r), fmt.Sprintf("%s%d", cell.column, r), cell.style); err != nil {
				return err
			}
		}
		_ = book.SetRowHeight(sheet, r, 18)
	}
	_ = book.SetColWidth(sheet, "A", "A", 15)
	_ = book.SetColWidth(sheet, "B", "B", 3)
	_ = book.SetColWidth(sheet, "C", "C", 27)
	_ = book.SetColWidth(sheet, "D", "D", 8)
	return insertMaterialListCompanyLogo(book, sheet, logo)
}

func writeMaterialListDetailQty(book *excelize.File, sheet string, detail model.MaterialListExportDetailQty, s materialListExcelStyles) (int, error) {
	sizeStart := 9
	totalCol, colorCol := sizeStart+len(detail.Sizes), sizeStart+len(detail.Sizes)+1
	if err := book.MergeCell(sheet, "H2", "H3"); err != nil {
		return 0, err
	}
	if err := book.SetCellValue(sheet, "H2", "DETAIL QTY"); err != nil {
		return 0, err
	}
	if err := book.SetCellStyle(sheet, "H2", "H3", s.detailHeader); err != nil {
		return 0, err
	}
	for i, size := range detail.Sizes {
		if err := materialListSetAny(book, sheet, sizeStart+i, 2, materialListValue(size.Ratio), s.detailHeader); err != nil {
			return 0, err
		}
		if err := materialListSet(book, sheet, sizeStart+i, 3, size.Name, s.detailHeader); err != nil {
			return 0, err
		}
		_ = book.SetColWidth(sheet, materialListColumn(sizeStart+i), materialListColumn(sizeStart+i), 8)
	}
	for _, cell := range []int{totalCol, colorCol} {
		if err := book.MergeCell(sheet, materialListCell(cell, 2), materialListCell(cell, 3)); err != nil {
			return 0, err
		}
	}
	if err := materialListSet(book, sheet, totalCol, 2, "TOTAL", s.detailHeader); err != nil {
		return 0, err
	}
	if err := materialListSet(book, sheet, colorCol, 2, "COLOR", s.detailHeader); err != nil {
		return 0, err
	}
	if err := book.SetCellStyle(sheet, materialListCell(totalCol, 2), materialListCell(totalCol, 3), s.detailHeader); err != nil {
		return 0, err
	}
	if err := book.SetCellStyle(sheet, materialListCell(colorCol, 2), materialListCell(colorCol, 3), s.detailHeader); err != nil {
		return 0, err
	}
	_ = book.SetColWidth(sheet, materialListColumn(totalCol), materialListColumn(totalCol), 10)
	_ = book.SetColWidth(sheet, materialListColumn(colorCol), materialListColumn(colorCol), 18)
	row := 4
	for _, shell := range detail.Shells {
		for _, label := range []string{"ORDER", "MARKER PLAN", "ACT CUT"} {
			rowStyle := s.detailBody
			if label == "ACT CUT" {
				rowStyle = s.detailActual
			}
			if err := materialListSet(book, sheet, 8, row, label, rowStyle); err != nil {
				return 0, err
			}
			for i, cell := range shell.Values {
				var v *int64
				if label == "ORDER" {
					v = cell.Order
				} else if label == "MARKER PLAN" {
					v = cell.MarkerPlan
				} else {
					v = cell.ActualCut
				}
				if err := materialListSetPtr(book, sheet, sizeStart+i, row, v, rowStyle); err != nil {
					return 0, err
				}
			}
			if err := materialListSetPtr(book, sheet, totalCol, row, materialListDetailTotal(shell.Values, label), rowStyle); err != nil {
				return 0, err
			}
			row++
		}
		if err := book.MergeCell(sheet, materialListCell(colorCol, row-3), materialListCell(colorCol, row-1)); err != nil {
			return 0, err
		}
		if err := materialListSet(book, sheet, colorCol, row-3, shell.Color, s.detailBody); err != nil {
			return 0, err
		}
		if err := book.SetCellStyle(sheet, materialListCell(colorCol, row-3), materialListCell(colorCol, row-1), s.detailBody); err != nil {
			return 0, err
		}
	}
	for summaryIndex, summary := range []struct {
		label string
		value func(model.MaterialListExportSizeSummary) *int64
	}{{"ORDER TOTAL", func(v model.MaterialListExportSizeSummary) *int64 { return v.OrderTotal }}, {"RMS/TOTAL", func(v model.MaterialListExportSizeSummary) *int64 { return v.RMSTotal }}, {"TOTAL ACT CUT", func(v model.MaterialListExportSizeSummary) *int64 { return v.TotalActCut }}, {"BALANCE", func(v model.MaterialListExportSizeSummary) *int64 { return v.Balance }}} {
		labelStyle, numericStyle := s.detailSummaryBlue, s.detailSummaryBlue
		if summaryIndex == 2 {
			labelStyle = s.detailSummaryYellow
		}
		if err := materialListSet(book, sheet, 8, row, summary.label, labelStyle); err != nil {
			return 0, err
		}
		var total int64
		complete := true
		for i := range detail.Sizes {
			var value *int64
			if i < len(detail.Summaries) {
				value = summary.value(detail.Summaries[i])
			}
			if value == nil {
				complete = false
			} else {
				total += *value
			}
			if err := materialListSetPtr(book, sheet, sizeStart+i, row, value, numericStyle); err != nil {
				return 0, err
			}
		}
		if complete && len(detail.Sizes) > 0 {
			if err := materialListSet(book, sheet, totalCol, row, total, numericStyle); err != nil {
				return 0, err
			}
		} else if err := materialListSet(book, sheet, totalCol, row, nil, numericStyle); err != nil {
			return 0, err
		}
		if err := materialListSet(book, sheet, colorCol, row, nil, numericStyle); err != nil {
			return 0, err
		}
		row++
	}
	return row + 1, nil
}

func writeMaterialListTable(book *excelize.File, sheet string, headerRow int, export *model.MaterialListExport, s materialListExcelStyles, lastCol int) (int, error) {
	col := 1
	for _, header := range []string{"DESCRIPTION", "SIZE", "COLOR / TYPE", "QTY WO", "CONS. / PC", "TOTAL", "UOM"} {
		if err := materialListMergeHeader(book, sheet, col, headerRow, 1, header, s.baseHeader); err != nil {
			return 0, err
		}
		col++
	}
	sjStart := col
	var sjTotal int
	if len(export.SJDates) == 0 {
		if err := materialListMergeHeader(book, sheet, col, headerRow, 1, "SJ SUHO TOTAL", s.sjHeader); err != nil {
			return 0, err
		}
		sjTotal = col
		col++
	} else {
		if err := materialListGroupHeader(book, sheet, col, headerRow, len(export.SJDates), "SJ SUHO", s.sjHeader); err != nil {
			return 0, err
		}
		col += len(export.SJDates)
		if err := materialListMergeHeader(book, sheet, col, headerRow, 1, "TOTAL", s.sjHeader); err != nil {
			return 0, err
		}
		sjTotal = col
		col++
	}
	rcvStart := col
	var rcvTotal int
	if len(export.ReceivedDates) == 0 {
		if err := materialListMergeHeader(book, sheet, col, headerRow, 1, "RCVD PERMATA TOTAL", s.receivedHeader); err != nil {
			return 0, err
		}
		rcvTotal = col
		col++
	} else {
		if err := materialListGroupHeader(book, sheet, col, headerRow, len(export.ReceivedDates), "RCVD PERMATA", s.receivedHeader); err != nil {
			return 0, err
		}
		col += len(export.ReceivedDates)
		if err := materialListMergeHeader(book, sheet, col, headerRow, 1, "TOTAL", s.receivedHeader); err != nil {
			return 0, err
		}
		rcvTotal = col
		col++
	}
	for _, header := range []struct {
		value string
		style int
	}{{"BALANCE RCVD", s.receivedHeader}, {"CUTTING (ACT/CUTPLAN)", s.operationalHeader}, {"ACTUAL CONS. (+allow)", s.operationalHeader}, {"REJECT / RETUR", s.operationalHeader}, {"BALANCE", s.balanceHeader}, {"NOTE", s.noteHeader}} {
		if err := materialListMergeHeader(book, sheet, col, headerRow, 1, header.value, header.style); err != nil {
			return 0, err
		}
		col++
	}
	for i, date := range export.SJDates {
		if err := materialListSet(book, sheet, sjStart+i, headerRow+1, materialListDateShort(date), s.sjDateHeader); err != nil {
			return 0, err
		}
	}
	for i, date := range export.ReceivedDates {
		if err := materialListSet(book, sheet, rcvStart+i, headerRow+1, materialListDateShort(date), s.receivedDateHeader); err != nil {
			return 0, err
		}
	}
	for c := 1; c < col; c++ {
		_ = book.SetColWidth(sheet, materialListColumn(c), materialListColumn(c), 11)
	}
	_ = book.SetColWidth(sheet, "A", "A", 28)
	_ = book.SetColWidth(sheet, "B", "C", 14)
	_ = book.SetColWidth(sheet, materialListColumn(col-1), materialListColumn(col-1), 22)
	row := headerRow + 2
	for _, category := range []string{"FABRIC", "SEWING", "PACKING", "UNCATEGORIZED / REVIEW"} {
		var group []model.MaterialListExportRow
		for _, item := range export.MaterialRows {
			if materialListCategory(item.Category) == category {
				group = append(group, item)
			}
		}
		if len(group) == 0 {
			continue
		}
		for c := 1; c < col; c++ {
			if err := materialListSet(book, sheet, c, row, nil, s.category); err != nil {
				return 0, err
			}
		}
		if err := materialListSet(book, sheet, 1, row, category, s.category); err != nil {
			return 0, err
		}
		_ = book.SetRowHeight(sheet, row, 16)
		row++
		for _, item := range group {
			if err := materialListWriteItem(book, sheet, row, item, export.SJDates, export.ReceivedDates, sjStart, sjTotal, rcvStart, rcvTotal, s); err != nil {
				return 0, err
			}
			_ = book.SetRowHeight(sheet, row, 24)
			row++
		}
	}
	return max(lastCol, col-1), nil
}

func materialListWriteItem(book *excelize.File, sheet string, row int, item model.MaterialListExportRow, sjDates, rcvDates []time.Time, sjStart, sjTotal, rcvStart, rcvTotal int, s materialListExcelStyles) error {
	values := []struct {
		col   int
		value any
		style int
	}{{1, materialListDescription(item), s.text}, {2, item.Applicability.SizeName, s.text}, {3, item.Applicability.ShellColor, s.text}, {4, item.QtyWO, s.integer}, {5, item.ConsPerPC, s.decimal}, {6, item.Total, s.decimal}, {7, item.Unit, s.text}}
	for _, v := range values {
		if err := materialListSetAny(book, sheet, v.col, row, v.value, v.style); err != nil {
			return err
		}
	}
	for i, date := range sjDates {
		if err := materialListSetTransaction(book, sheet, sjStart+i, row, item.SJByDate, date, s.integer); err != nil {
			return err
		}
	}
	if err := materialListSetAny(book, sheet, sjTotal, row, item.SJTotal, s.integer); err != nil {
		return err
	}
	for i, date := range rcvDates {
		if err := materialListSetTransaction(book, sheet, rcvStart+i, row, item.ReceivedByDate, date, s.integer); err != nil {
			return err
		}
	}
	if err := materialListSetAny(book, sheet, rcvTotal, row, item.ReceivedTotal, s.integer); err != nil {
		return err
	}
	for _, v := range []struct {
		col   int
		value any
		style int
	}{{rcvTotal + 1, item.BalanceReceived, s.integer}, {rcvTotal + 2, materialListCuttingDisplay(item), s.text}, {rcvTotal + 3, item.ActualConsumption, s.decimal}, {rcvTotal + 4, item.RejectRetur, s.integer}, {rcvTotal + 5, item.FinalBalance, s.balanceValue}, {rcvTotal + 6, nil, s.text}} {
		if err := materialListSetAny(book, sheet, v.col, row, v.value, v.style); err != nil {
			return err
		}
	}
	return nil
}

func insertMaterialListCompanyLogo(book *excelize.File, sheet, logo string) error {
	encoded := strings.TrimSpace(logo)
	if encoded == "" {
		return nil
	}
	comma := strings.Index(encoded, ",")
	if !strings.HasPrefix(encoded, "data:image/") || comma < 0 {
		return nil
	}
	content, err := base64.StdEncoding.DecodeString(encoded[comma+1:])
	if err != nil {
		return nil
	}
	ext := ".png"
	if strings.HasPrefix(encoded, "data:image/jpeg") {
		ext = ".jpg"
	}
	return book.AddPictureFromBytes(sheet, "A1", &excelize.Picture{Extension: ext, File: content, Format: &excelize.GraphicOptions{ScaleX: 0.45, ScaleY: 0.45, OffsetX: 4, OffsetY: 4}})
}

func buildMaterialListExportFileName(export *model.MaterialListExport) string {
	name := materialListExportFileNameSegment(export.Header.MaterialListName)
	baseName := "MATERIAL_LIST"
	if name != "" {
		baseName += "_" + name
	}
	return ensureExportExtension(baseName, ".xlsx")
}

// materialListExportFileNameSegment follows the established export sanitizer
// while deliberately preserving an empty result for this document's clean
// no-name fallback (MATERIAL_LIST.xlsx).
func materialListExportFileNameSegment(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	replacer := strings.NewReplacer(
		"\\", "_",
		"/", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		" ", "_",
	)
	return strings.Trim(replacer.Replace(trimmed), "._")
}
func materialListFOBCMT(v bool) string {
	if v {
		return "FOB"
	}
	return "CMT"
}
func materialListDate(v time.Time) string {
	if v.IsZero() {
		return ""
	}
	return v.Format("02-Jan-2006")
}
func materialListDateShort(v time.Time) string {
	if v.IsZero() {
		return ""
	}
	return v.Format("02-Jan")
}
func materialListColumn(col int) string    { name, _ := excelize.ColumnNumberToName(col); return name }
func materialListCell(col, row int) string { return fmt.Sprintf("%s%d", materialListColumn(col), row) }
func materialListSet(book *excelize.File, sheet string, col, row int, value any, style int) error {
	return materialListSetAny(book, sheet, col, row, value, style)
}
func materialListSetAny(book *excelize.File, sheet string, col, row int, value any, style int) error {
	cell := materialListCell(col, row)
	if err := book.SetCellValue(sheet, cell, materialListValue(value)); err != nil {
		return err
	}
	return book.SetCellStyle(sheet, cell, cell, style)
}
func materialListSetPtr(book *excelize.File, sheet string, col, row int, value *int64, style int) error {
	return materialListSetAny(book, sheet, col, row, value, style)
}
func materialListValue(v any) any {
	switch n := v.(type) {
	case nil:
		return nil
	case string:
		if n == "" {
			return nil
		}
		return n
	case *int64:
		if n == nil {
			return nil
		}
		return *n
	case *int32:
		if n == nil {
			return nil
		}
		return *n
	case *float64:
		if n == nil {
			return nil
		}
		return *n
	case *string:
		if n == nil || *n == "" {
			return nil
		}
		return *n
	default:
		return v
	}
}
func materialListSetTransaction(book *excelize.File, sheet string, col, row int, values map[time.Time]int64, date time.Time, style int) error {
	value, ok := values[date]
	if !ok {
		return materialListSet(book, sheet, col, row, nil, style)
	}
	return materialListSet(book, sheet, col, row, value, style)
}
func materialListMergeHeader(book *excelize.File, sheet string, col, row, span int, value string, style int) error {
	end := col + max(span, 1) - 1
	if err := book.MergeCell(sheet, materialListCell(col, row), materialListCell(end, row+1)); err != nil {
		return err
	}
	if err := materialListSet(book, sheet, col, row, value, style); err != nil {
		return err
	}
	return book.SetCellStyle(sheet, materialListCell(col, row), materialListCell(end, row+1), style)
}
func materialListGroupHeader(book *excelize.File, sheet string, col, row, span int, value string, style int) error {
	end := col + span - 1
	if span > 1 {
		if err := book.MergeCell(sheet, materialListCell(col, row), materialListCell(end, row)); err != nil {
			return err
		}
	}
	if err := materialListSet(book, sheet, col, row, value, style); err != nil {
		return err
	}
	return book.SetCellStyle(sheet, materialListCell(col, row), materialListCell(end, row), style)
}
func materialListDetailTotal(cells []model.MaterialListExportShellSizeQty, label string) *int64 {
	var total int64
	any := false
	for _, cell := range cells {
		var v *int64
		if label == "ORDER" {
			v = cell.Order
		} else if label == "MARKER PLAN" {
			v = cell.MarkerPlan
		} else {
			v = cell.ActualCut
		}
		if v != nil {
			total += *v
			any = true
		}
	}
	if !any {
		return nil
	}
	return &total
}
func materialListCategory(category *string) string {
	if category == nil {
		return "UNCATEGORIZED / REVIEW"
	}
	switch strings.ToUpper(strings.TrimSpace(*category)) {
	case "FABRIC", "SEWING", "PACKING":
		return strings.ToUpper(strings.TrimSpace(*category))
	default:
		return "UNCATEGORIZED / REVIEW"
	}
}
func materialListDescription(item model.MaterialListExportRow) string {
	return strings.TrimSpace(item.Item)
}
func materialListCuttingDisplay(item model.MaterialListExportRow) string {
	if item.CuttingQty == nil {
		return ""
	}
	switch item.CuttingSource {
	case model.MaterialListCuttingActual:
		return fmt.Sprintf("ACTUAL %d", *item.CuttingQty)
	case model.MaterialListCuttingCutPlan:
		return fmt.Sprintf("CUTPLAN %d", *item.CuttingQty)
	default:
		return ""
	}
}
func boolPtr(v bool) *bool { return &v }
