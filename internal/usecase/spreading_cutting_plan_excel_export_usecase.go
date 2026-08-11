package usecase

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"

	"permatatex-inventory/internal/model"
	excelexporter "permatatex-inventory/pkg/exporter/excel"
)

const (
	spreadingCuttingPlanExportTemplateName = "xlsx/template_spreading_cutting_plan.xlsx"
	spreadingCuttingPlanExportSheetName    = "SCP"
	spreadingCuttingPlanSizeStartColumn    = 7
	spreadingCuttingPlanTableHeaderRow     = 8
	spreadingCuttingPlanSubHeaderRow       = 9
	spreadingCuttingPlanBodyStartRow       = 10
)

type spreadingCuttingPlanExportComposer interface {
	Compose(context.Context, int32) (*model.SpreadingCuttingPlanExport, error)
}

type SpreadingCuttingPlanExcelExportUseCase struct {
	renderer *excelexporter.Renderer
	composer spreadingCuttingPlanExportComposer
}

func NewSpreadingCuttingPlanExcelExportUseCase(
	renderer *excelexporter.Renderer,
	composer spreadingCuttingPlanExportComposer,
) (*SpreadingCuttingPlanExcelExportUseCase, error) {
	if renderer == nil {
		return nil, errors.New("excel renderer is required")
	}
	if composer == nil {
		return nil, errors.New("spreading cutting plan export composer is required")
	}
	return &SpreadingCuttingPlanExcelExportUseCase{renderer: renderer, composer: composer}, nil
}

func (u *SpreadingCuttingPlanExcelExportUseCase) ExportByID(ctx context.Context, idSpreadingCuttingPlan int32) (*model.ExportedFile, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	export, err := u.composer.Compose(ctx, idSpreadingCuttingPlan)
	if err != nil {
		return nil, err
	}
	return renderSpreadingCuttingPlanExcel(u.renderer, export)
}

type spreadingCuttingPlanExcelColumns struct {
	sizeStart, sizeEnd int
	total, spread      int
	subCons, roll      int
	joined, sambungan  int
	totalCons, reject  int
	balance, note      int
	headerLabel        int
	headerColon        int
	headerValue        int
	printEnd           int
}

type spreadingCuttingPlanExcelStyles struct {
	title                    int
	header, subHeader        int
	bodyText, bodyInt        int
	bodyDecimal              int
	planSpreading            int
	totalText, totalInt      int
	headerLabel, headerValue int
	headerDate               int
	border                   []excelize.Border
}

type spreadingCuttingPlanExcelShellStyles struct {
	text, integer, decimal int
}

func renderSpreadingCuttingPlanExcel(renderer *excelexporter.Renderer, export *model.SpreadingCuttingPlanExport) (*model.ExportedFile, error) {
	if renderer == nil {
		return nil, errors.New("excel renderer is required")
	}
	if export == nil {
		return nil, errors.New("spreading cutting plan export data is required")
	}
	if len(export.Sizes) == 0 || len(export.Shells) == 0 {
		return nil, spreadingCuttingPlanExportConflict("renderable sizes and shells are required")
	}

	book, err := renderer.OpenTemplate(spreadingCuttingPlanExportTemplateName)
	if err != nil {
		return nil, fmt.Errorf("open spreading cutting plan export template: %w", err)
	}
	defer book.Close()

	if index, err := book.GetSheetIndex(spreadingCuttingPlanExportSheetName); err != nil || index < 0 {
		return nil, fmt.Errorf("open spreading cutting plan export template: missing %s sheet", spreadingCuttingPlanExportSheetName)
	}

	columns := spreadingCuttingPlanExcelColumnLayout(len(export.Sizes))
	styles, err := newSpreadingCuttingPlanExcelStyles(book)
	if err != nil {
		return nil, fmt.Errorf("create spreading cutting plan export styles: %w", err)
	}
	if err := writeSpreadingCuttingPlanExcelHeader(book, export, columns, styles); err != nil {
		return nil, fmt.Errorf("write spreading cutting plan export header: %w", err)
	}
	if err := writeSpreadingCuttingPlanExcelTableHeader(book, export, columns, styles); err != nil {
		return nil, fmt.Errorf("write spreading cutting plan export table header: %w", err)
	}
	endRow, err := writeSpreadingCuttingPlanExcelBody(book, export, columns, styles)
	if err != nil {
		return nil, fmt.Errorf("write spreading cutting plan export body: %w", err)
	}
	if err := applySpreadingCuttingPlanExcelPageLayout(book, columns, endRow); err != nil {
		return nil, fmt.Errorf("apply spreading cutting plan export page layout: %w", err)
	}
	if err := book.SetCalcProps(&excelize.CalcPropsOptions{
		CalcMode: spreadingCuttingPlanStringPtr("auto"), FullCalcOnLoad: spreadingCuttingPlanBoolPtr(true),
		CalcOnSave: spreadingCuttingPlanBoolPtr(true), ForceFullCalc: spreadingCuttingPlanBoolPtr(true),
	}); err != nil {
		return nil, fmt.Errorf("apply spreading cutting plan export calculation mode: %w", err)
	}

	var output bytes.Buffer
	if err := book.Write(&output); err != nil {
		return nil, fmt.Errorf("write spreading cutting plan export workbook: %w", err)
	}
	return &model.ExportedFile{
		FileName:    buildSpreadingCuttingPlanExportFileName(export),
		ContentType: excelContentType,
		Content:     output.Bytes(),
	}, nil
}

func spreadingCuttingPlanExcelColumnLayout(sizeCount int) spreadingCuttingPlanExcelColumns {
	columns := spreadingCuttingPlanExcelColumns{
		sizeStart: spreadingCuttingPlanSizeStartColumn,
		sizeEnd:   spreadingCuttingPlanSizeStartColumn + sizeCount - 1,
	}
	columns.total = columns.sizeEnd + 1
	columns.spread = columns.total + 1
	columns.subCons = columns.spread + 1
	columns.roll = columns.subCons + 1
	columns.joined = columns.roll + 1
	columns.sambungan = columns.joined + 1
	columns.totalCons = columns.sambungan + 1
	columns.reject = columns.totalCons + 1
	columns.balance = columns.reject + 1
	columns.note = columns.balance + 1
	columns.headerLabel = max(columns.total, 13)
	columns.headerColon = columns.headerLabel + 1
	columns.headerValue = columns.headerColon + 1
	columns.printEnd = max(columns.note, columns.headerValue+2)
	return columns
}

func newSpreadingCuttingPlanExcelStyles(book *excelize.File) (spreadingCuttingPlanExcelStyles, error) {
	border := []excelize.Border{
		{Type: "left", Color: "000000", Style: 1},
		{Type: "top", Color: "000000", Style: 1},
		{Type: "right", Color: "000000", Style: 1},
		{Type: "bottom", Color: "000000", Style: 1},
	}
	detailBorder := []excelize.Border{
		{Type: "left", Color: "92D050", Style: 1},
		{Type: "top", Color: "92D050", Style: 1},
		{Type: "right", Color: "92D050", Style: 1},
		{Type: "bottom", Color: "92D050", Style: 1},
	}
	center := &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true}
	left := &excelize.Alignment{Horizontal: "left", Vertical: "center", WrapText: true}

	newStyle := func(style *excelize.Style) (int, error) { return book.NewStyle(style) }
	title, err := newStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "000000", Family: "Calibri", Size: 18},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	})
	if err != nil {
		return spreadingCuttingPlanExcelStyles{}, err
	}
	header, err := newStyle(&excelize.Style{
		Font:   &excelize.Font{Bold: true, Color: "000000", Family: "Calibri", Size: 11},
		Fill:   excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"92D050"}},
		Border: border, Alignment: center,
	})
	if err != nil {
		return spreadingCuttingPlanExcelStyles{}, err
	}
	subHeader, err := newStyle(&excelize.Style{
		Font:   &excelize.Font{Bold: true, Color: "000000", Family: "Calibri", Size: 11},
		Fill:   excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"92D050"}},
		Border: border, Alignment: center,
	})
	if err != nil {
		return spreadingCuttingPlanExcelStyles{}, err
	}
	bodyText, err := newStyle(&excelize.Style{
		Font: &excelize.Font{Family: "Calibri", Size: 11}, Border: detailBorder, Alignment: center,
	})
	if err != nil {
		return spreadingCuttingPlanExcelStyles{}, err
	}
	bodyInt, err := newStyle(&excelize.Style{
		Font: &excelize.Font{Family: "Calibri", Size: 11}, Border: detailBorder, Alignment: center,
		NumFmt: 3,
	})
	if err != nil {
		return spreadingCuttingPlanExcelStyles{}, err
	}
	bodyDecimal, err := newStyle(&excelize.Style{
		Font: &excelize.Font{Family: "Calibri", Size: 11}, Border: detailBorder, Alignment: center,
		CustomNumFmt: spreadingCuttingPlanStringPtr("#,##0.###"),
	})
	if err != nil {
		return spreadingCuttingPlanExcelStyles{}, err
	}
	planSpreading, err := newStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Family: "Calibri", Size: 11}, Border: detailBorder, Alignment: center,
		CustomNumFmt: spreadingCuttingPlanStringPtr("#,##0.### \"LAYER\""),
	})
	if err != nil {
		return spreadingCuttingPlanExcelStyles{}, err
	}
	totalText, err := newStyle(&excelize.Style{
		Font:   &excelize.Font{Bold: true, Color: "000000", Family: "Calibri", Size: 11},
		Fill:   excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"92D050"}},
		Border: border, Alignment: center,
	})
	if err != nil {
		return spreadingCuttingPlanExcelStyles{}, err
	}
	totalInt, err := newStyle(&excelize.Style{
		Font:   &excelize.Font{Bold: true, Color: "000000", Family: "Calibri", Size: 11},
		Fill:   excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"92D050"}},
		Border: border, Alignment: center, NumFmt: 3,
	})
	if err != nil {
		return spreadingCuttingPlanExcelStyles{}, err
	}
	headerLabel, err := newStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Family: "Calibri", Size: 11}, Alignment: left,
	})
	if err != nil {
		return spreadingCuttingPlanExcelStyles{}, err
	}
	headerValue, err := newStyle(&excelize.Style{
		Font: &excelize.Font{Family: "Calibri", Size: 11}, Alignment: left,
	})
	if err != nil {
		return spreadingCuttingPlanExcelStyles{}, err
	}

	headerDate, err := newStyle(&excelize.Style{
		Font: &excelize.Font{Family: "Calibri", Size: 11}, Alignment: left,
		CustomNumFmt: spreadingCuttingPlanStringPtr("dd-mmm-yyyy"),
	})
	if err != nil {
		return spreadingCuttingPlanExcelStyles{}, err
	}
	return spreadingCuttingPlanExcelStyles{
		title: title, header: header, subHeader: subHeader, bodyText: bodyText, bodyInt: bodyInt,
		bodyDecimal: bodyDecimal, planSpreading: planSpreading, totalText: totalText, totalInt: totalInt,
		headerLabel: headerLabel, headerValue: headerValue, headerDate: headerDate, border: border,
	}, nil
}

func newSpreadingCuttingPlanExcelShellStyles(book *excelize.File, border []excelize.Border, colorName string) (spreadingCuttingPlanExcelShellStyles, error) {
	fillColor := spreadingCuttingPlanShellFillColor(colorName)
	font := &excelize.Font{Bold: true, Color: "FFFFFF", Family: "Calibri", Size: 11}
	fill := excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{fillColor}}
	alignment := &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true}

	textStyle, err := book.NewStyle(&excelize.Style{Font: font, Fill: fill, Border: border, Alignment: alignment})
	if err != nil {
		return spreadingCuttingPlanExcelShellStyles{}, err
	}
	integerStyle, err := book.NewStyle(&excelize.Style{Font: font, Fill: fill, Border: border, Alignment: alignment, NumFmt: 3})
	if err != nil {
		return spreadingCuttingPlanExcelShellStyles{}, err
	}
	decimalStyle, err := book.NewStyle(&excelize.Style{
		Font: font, Fill: fill, Border: border, Alignment: alignment,
		CustomNumFmt: spreadingCuttingPlanStringPtr("#,##0.###"),
	})
	if err != nil {
		return spreadingCuttingPlanExcelShellStyles{}, err
	}
	return spreadingCuttingPlanExcelShellStyles{text: textStyle, integer: integerStyle, decimal: decimalStyle}, nil
}

func spreadingCuttingPlanShellFillColor(colorName string) string {
	switch strings.ToUpper(strings.TrimSpace(colorName)) {
	case "NAVY":
		return "1E4E79"
	case "BLACK":
		return "000000"
	case "MAROON":
		return "8E0407"
	default:
		return "1F4E78"
	}
}

func writeSpreadingCuttingPlanExcelHeader(
	book *excelize.File,
	export *model.SpreadingCuttingPlanExport,
	columns spreadingCuttingPlanExcelColumns,
	styles spreadingCuttingPlanExcelStyles,
) error {
	sheet := spreadingCuttingPlanExportSheetName
	if err := book.SetCellValue(sheet, "A5", "SPREADING & CUTTING PLAN"); err != nil {
		return err
	}
	if err := book.SetCellStyle(sheet, "A5", "A5", styles.title); err != nil {
		return err
	}

	styleModel := strings.TrimSpace(export.Header.Style)
	modelName := strings.TrimSpace(export.Header.Model)
	if styleModel == "" {
		styleModel = modelName
	} else if modelName != "" && !strings.EqualFold(styleModel, modelName) {
		styleModel += " / " + modelName
	}
	ratioPO := ""
	if export.Header.RatioPO != nil {
		ratioPO = *export.Header.RatioPO
	}
	headerRows := []struct {
		row   int
		label string
		value any
	}{
		{row: 1, label: "BUYER", value: export.Header.Buyer},
		{row: 2, label: "TANGGAL", value: export.Header.EffectiveDate},
		{row: 3, label: "STYLE", value: styleModel},
		{row: 4, label: "WARNA", value: strings.Join(export.Header.Colors, " - ")},
		{row: 6, label: "RATIO PO", value: ratioPO},
	}
	for _, header := range headerRows {
		labelCell := spreadingCuttingPlanCell(columns.headerLabel, header.row)
		colonCell := spreadingCuttingPlanCell(columns.headerColon, header.row)
		valueCell := spreadingCuttingPlanCell(columns.headerValue, header.row)
		if err := book.MergeCell(sheet, valueCell, spreadingCuttingPlanCell(columns.printEnd, header.row)); err != nil {
			return err
		}
		if err := book.SetCellValue(sheet, labelCell, header.label); err != nil {
			return err
		}
		if err := book.SetCellValue(sheet, colonCell, ":"); err != nil {
			return err
		}
		if err := book.SetCellValue(sheet, valueCell, header.value); err != nil {
			return err
		}
		if err := book.SetCellStyle(sheet, labelCell, labelCell, styles.headerLabel); err != nil {
			return err
		}
		if err := book.SetCellStyle(sheet, colonCell, colonCell, styles.headerLabel); err != nil {
			return err
		}
		if err := book.SetCellStyle(sheet, valueCell, spreadingCuttingPlanCell(columns.printEnd, header.row), styles.headerValue); err != nil {
			return err
		}
	}
	return book.SetCellStyle(sheet, spreadingCuttingPlanCell(columns.headerValue, 2), spreadingCuttingPlanCell(columns.headerValue, 2), styles.headerDate)
}

func writeSpreadingCuttingPlanExcelTableHeader(
	book *excelize.File,
	export *model.SpreadingCuttingPlanExport,
	columns spreadingCuttingPlanExcelColumns,
	styles spreadingCuttingPlanExcelStyles,
) error {
	sheet := spreadingCuttingPlanExportSheetName
	merges := [][4]int{
		{1, 8, 2, 8}, {3, 8, 3, 9}, {5, 8, 5, 9}, {6, 8, 6, 9},
		{columns.sizeStart, 8, columns.sizeEnd, 8},
		{columns.total, 8, columns.total, 9},
		{columns.spread, 8, columns.spread, 9},
		{columns.subCons, 8, columns.subCons, 9},
		{columns.roll, 8, columns.totalCons, 8},
		{columns.note, 8, columns.note, 9},
	}
	for _, merge := range merges {
		if err := book.MergeCell(sheet, spreadingCuttingPlanCell(merge[0], merge[1]), spreadingCuttingPlanCell(merge[2], merge[3])); err != nil {
			return err
		}
	}
	values := map[[2]int]any{
		{1, 8}: "Kedatangan kain (actual)", {1, 9}: "Grade", {2, 9}: "(yard)",
		{3, 8}: "Cons.", {5, 8}: "Cons. /pc",
		{6, 8}: "Ratio", {columns.sizeStart, 8}: "QTY CUT PLAN",
		{columns.total, 8}: "TOTAL", {columns.spread, 8}: "PLAN SPREADING",
		{columns.subCons, 8}: "Sub Cons.", {columns.roll, 8}: "Pemakaian kain",
		{columns.roll, 9}: "(Roll)", {columns.joined, 9}: "Sambungan /roll",
		{columns.sambungan, 9}: "Sambungan /yard", {columns.totalCons, 9}: "Total Cons. (yard)",
		{columns.reject, 8}: "Reject", {columns.reject, 9}: "(yard)",
		{columns.balance, 8}: "Balance", {columns.balance, 9}: "(yard)",
		{columns.note, 8}: "NOTE",
	}
	for index, size := range export.Sizes {
		values[[2]int{columns.sizeStart + index, 9}] = size.Label
	}
	for location, value := range values {
		if err := book.SetCellValue(sheet, spreadingCuttingPlanCell(location[0], location[1]), value); err != nil {
			return err
		}
	}
	if err := book.SetCellStyle(sheet, "A8", spreadingCuttingPlanCell(columns.note, 8), styles.header); err != nil {
		return err
	}
	if err := book.SetCellStyle(sheet, "A9", spreadingCuttingPlanCell(columns.note, 9), styles.subHeader); err != nil {
		return err
	}
	allowanceCell := spreadingCuttingPlanCell(4, 9)
	if err := book.SetCellValue(sheet, allowanceCell, export.AllowanceHeader); err != nil {
		return err
	}
	if err := book.SetRowHeight(sheet, 8, 15); err != nil {
		return err
	}
	return book.SetRowHeight(sheet, 9, 30)
}

func writeSpreadingCuttingPlanExcelBody(
	book *excelize.File,
	export *model.SpreadingCuttingPlanExport,
	columns spreadingCuttingPlanExcelColumns,
	styles spreadingCuttingPlanExcelStyles,
) (int, error) {
	sheet := spreadingCuttingPlanExportSheetName
	row := spreadingCuttingPlanBodyStartRow
	poRows := make([]int, 0, len(export.Shells))
	subtotalRows := make([]int, 0, len(export.Shells))
	for _, shell := range export.Shells {
		shellStyles, err := newSpreadingCuttingPlanExcelShellStyles(book, styles.border, shell.Color)
		if err != nil {
			return 0, err
		}
		poRows = append(poRows, row)
		if err := book.MergeCell(sheet, spreadingCuttingPlanCell(1, row), spreadingCuttingPlanCell(6, row)); err != nil {
			return 0, err
		}
		if err := spreadingCuttingPlanSet(book, row, 1, "PO "+shell.Color); err != nil {
			return 0, err
		}
		for index, qty := range shell.OrderQty {
			if err := spreadingCuttingPlanSet(book, row, columns.sizeStart+index, qty); err != nil {
				return 0, err
			}
		}
		if err := spreadingCuttingPlanSet(book, row, columns.total, shell.OrderTotal); err != nil {
			return 0, err
		}
		if err := spreadingCuttingPlanFormula(book, row, columns.total, fmt.Sprintf("SUM(%s:%s)", spreadingCuttingPlanCell(columns.sizeStart, row), spreadingCuttingPlanCell(columns.sizeEnd, row))); err != nil {
			return 0, err
		}
		if err := book.SetCellStyle(sheet, spreadingCuttingPlanCell(1, row), spreadingCuttingPlanCell(6, row), shellStyles.text); err != nil {
			return 0, err
		}
		if err := book.SetCellStyle(sheet, spreadingCuttingPlanCell(columns.sizeStart, row), spreadingCuttingPlanCell(columns.note, row), shellStyles.integer); err != nil {
			return 0, err
		}
		if err := book.SetRowHeight(sheet, row, 16.5); err != nil {
			return 0, err
		}
		row++

		plannedRows := make([]int, 0, len(shell.Ratios))
		for ratioIndex, ratio := range shell.Ratios {
			metaRow, plannedRow := row, row+1
			plannedRows = append(plannedRows, plannedRow)
			if ratioIndex == 0 {
				if err := spreadingCuttingPlanSet(book, metaRow, 1, shell.Color); err != nil {
					return 0, err
				}
				if shell.ReceivedActual != nil {
					if err := spreadingCuttingPlanSet(book, metaRow, 2, *shell.ReceivedActual); err != nil {
						return 0, err
					}
				}
			}
			for _, assignment := range []struct {
				column int
				value  any
			}{
				{3, ratio.Cons}, {4, ratio.AllowanceConsumption}, {5, ratio.ConsPerPC}, {6, ratio.Sequence},
			} {
				if err := spreadingCuttingPlanSet(book, metaRow, assignment.column, assignment.value); err != nil {
					return 0, err
				}
			}
			allowanceFormula := fmt.Sprintf("%s*%s", spreadingCuttingPlanCell(3, metaRow), strconv.FormatFloat(ratio.AllowancePercent/100, 'f', -1, 64))
			if err := spreadingCuttingPlanFormula(book, metaRow, 4, allowanceFormula); err != nil {
				return 0, err
			}
			if err := spreadingCuttingPlanFormula(book, metaRow, 5, fmt.Sprintf("%s+%s", spreadingCuttingPlanCell(3, metaRow), spreadingCuttingPlanCell(4, metaRow))); err != nil {
				return 0, err
			}
			for index, value := range ratio.RatioPlan {
				if value != nil {
					if err := spreadingCuttingPlanSet(book, metaRow, columns.sizeStart+index, *value); err != nil {
						return 0, err
					}
				}
			}
			for index, value := range ratio.PlannedQty {
				if value != nil {
					if err := spreadingCuttingPlanSet(book, plannedRow, columns.sizeStart+index, *value); err != nil {
						return 0, err
					}
					if err := spreadingCuttingPlanFormula(book, plannedRow, columns.sizeStart+index, fmt.Sprintf("ROUND(%s*%s,0)", spreadingCuttingPlanCell(columns.sizeStart+index, metaRow), spreadingCuttingPlanCell(columns.spread, plannedRow))); err != nil {
						return 0, err
					}
				}
			}
			for _, assignment := range []struct {
				column int
				value  any
			}{
				{columns.total, ratio.PlannedTotal}, {columns.spread, ratio.PlanSpreadingGelaran},
				{columns.subCons, ratio.SubConsumption}, {columns.roll, ratio.RollQty},
				{columns.joined, ratio.JoinedRoll}, {columns.sambungan, ratio.SambunganYard},
				{columns.totalCons, ratio.TotalConsumption}, {columns.reject, ratio.Reject},
				{columns.note, ratio.Note},
			} {
				if err := spreadingCuttingPlanSet(book, plannedRow, assignment.column, assignment.value); err != nil {
					return 0, err
				}
			}
			if err := spreadingCuttingPlanFormula(book, plannedRow, columns.total, fmt.Sprintf("SUM(%s:%s)", spreadingCuttingPlanCell(columns.sizeStart, plannedRow), spreadingCuttingPlanCell(columns.sizeEnd, plannedRow))); err != nil {
				return 0, err
			}
			if err := spreadingCuttingPlanFormula(book, plannedRow, columns.subCons, fmt.Sprintf("%s*%s", spreadingCuttingPlanCell(columns.total, plannedRow), spreadingCuttingPlanCell(5, metaRow))); err != nil {
				return 0, err
			}
			if err := spreadingCuttingPlanFormula(book, plannedRow, columns.sambungan, fmt.Sprintf("%s*%s", spreadingCuttingPlanCell(columns.roll, plannedRow), spreadingCuttingPlanCell(columns.joined, plannedRow))); err != nil {
				return 0, err
			}
			if err := spreadingCuttingPlanFormula(book, plannedRow, columns.totalCons, fmt.Sprintf("%s+%s", spreadingCuttingPlanCell(columns.subCons, plannedRow), spreadingCuttingPlanCell(columns.sambungan, plannedRow))); err != nil {
				return 0, err
			}
			if ratio.RunningBalance != nil {
				if err := spreadingCuttingPlanSet(book, plannedRow, columns.balance, *ratio.RunningBalance); err != nil {
					return 0, err
				}
				balanceStart := spreadingCuttingPlanCell(2, metaRow)
				if ratioIndex > 0 {
					balanceStart = spreadingCuttingPlanCell(columns.balance, plannedRows[ratioIndex-1])
				}
				if err := spreadingCuttingPlanFormula(book, plannedRow, columns.balance, fmt.Sprintf("%s-%s-%s", balanceStart, spreadingCuttingPlanCell(columns.totalCons, plannedRow), spreadingCuttingPlanCell(columns.reject, plannedRow))); err != nil {
					return 0, err
				}
			}
			if err := book.SetCellStyle(sheet, spreadingCuttingPlanCell(1, metaRow), spreadingCuttingPlanCell(columns.note, plannedRow), styles.bodyDecimal); err != nil {
				return 0, err
			}
			if err := book.SetCellStyle(sheet, spreadingCuttingPlanCell(1, metaRow), spreadingCuttingPlanCell(1, plannedRow), styles.bodyText); err != nil {
				return 0, err
			}
			if err := book.SetCellStyle(sheet, spreadingCuttingPlanCell(columns.note, metaRow), spreadingCuttingPlanCell(columns.note, plannedRow), styles.bodyText); err != nil {
				return 0, err
			}
			if err := book.SetCellStyle(sheet, spreadingCuttingPlanCell(6, metaRow), spreadingCuttingPlanCell(columns.total, plannedRow), styles.bodyInt); err != nil {
				return 0, err
			}
			for _, column := range []int{2, columns.roll, columns.joined, columns.sambungan} {
				if err := book.SetCellStyle(sheet, spreadingCuttingPlanCell(column, metaRow), spreadingCuttingPlanCell(column, plannedRow), styles.bodyInt); err != nil {
					return 0, err
				}
			}
			if err := book.SetCellStyle(sheet, spreadingCuttingPlanCell(columns.spread, plannedRow), spreadingCuttingPlanCell(columns.spread, plannedRow), styles.planSpreading); err != nil {
				return 0, err
			}
			if err := book.SetRowHeight(sheet, metaRow, 16.5); err != nil {
				return 0, err
			}
			if err := book.SetRowHeight(sheet, plannedRow, 16.5); err != nil {
				return 0, err
			}
			row += 2
		}

		subtotalRows = append(subtotalRows, row)
		if err := book.MergeCell(sheet, spreadingCuttingPlanCell(1, row), spreadingCuttingPlanCell(6, row)); err != nil {
			return 0, err
		}
		if err := spreadingCuttingPlanSet(book, row, 1, "Sub Total "+shell.Color); err != nil {
			return 0, err
		}
		for index, value := range shell.Subtotal.PlannedQty {
			if value != nil {
				if err := spreadingCuttingPlanSet(book, row, columns.sizeStart+index, *value); err != nil {
					return 0, err
				}
				if err := spreadingCuttingPlanFormula(book, row, columns.sizeStart+index, spreadingCuttingPlanSumFormula(plannedRows, columns.sizeStart+index)); err != nil {
					return 0, err
				}
			}
		}
		for _, assignment := range []struct {
			column int
			value  any
		}{
			{columns.total, shell.Subtotal.PlannedTotal}, {columns.subCons, shell.Subtotal.SubConsumption},
			{columns.sambungan, shell.Subtotal.SambunganYard}, {columns.totalCons, shell.Subtotal.TotalConsumption},
			{columns.reject, shell.Subtotal.Reject},
		} {
			if err := spreadingCuttingPlanSet(book, row, assignment.column, assignment.value); err != nil {
				return 0, err
			}
		}
		if err := spreadingCuttingPlanFormula(book, row, columns.total, fmt.Sprintf("SUM(%s:%s)", spreadingCuttingPlanCell(columns.sizeStart, row), spreadingCuttingPlanCell(columns.sizeEnd, row))); err != nil {
			return 0, err
		}
		for _, column := range []int{columns.subCons, columns.sambungan, columns.totalCons, columns.reject} {
			if err := spreadingCuttingPlanFormula(book, row, column, spreadingCuttingPlanSumFormula(plannedRows, column)); err != nil {
				return 0, err
			}
		}
		if shell.Subtotal.Balance != nil {
			if err := spreadingCuttingPlanSet(book, row, columns.balance, *shell.Subtotal.Balance); err != nil {
				return 0, err
			}
			if err := spreadingCuttingPlanFormula(book, row, columns.balance, spreadingCuttingPlanCell(columns.balance, plannedRows[len(plannedRows)-1])); err != nil {
				return 0, err
			}
		}
		if err := book.SetCellStyle(sheet, spreadingCuttingPlanCell(1, row), spreadingCuttingPlanCell(6, row), shellStyles.text); err != nil {
			return 0, err
		}
		if err := book.SetCellStyle(sheet, spreadingCuttingPlanCell(columns.sizeStart, row), spreadingCuttingPlanCell(columns.total, row), shellStyles.integer); err != nil {
			return 0, err
		}
		if err := book.SetCellStyle(sheet, spreadingCuttingPlanCell(columns.spread, row), spreadingCuttingPlanCell(columns.note, row), shellStyles.decimal); err != nil {
			return 0, err
		}
		if err := book.SetRowHeight(sheet, row, 16.5); err != nil {
			return 0, err
		}
		row++
	}

	row++
	totalRows := []struct {
		label  string
		values []any
		total  int64
	}{
		{label: "TOTAL QTY PO", values: spreadingCuttingPlanInt64Values(export.Totals.OrderQty), total: export.Totals.OrderTotal},
		{label: "TOTAL QTY CUT PLAN", values: spreadingCuttingPlanOptionalInt64Values(export.Totals.PlannedQty), total: export.Totals.PlannedTotal},
		{label: "BALANCE", values: spreadingCuttingPlanOptionalInt64Values(export.Totals.BalanceQty), total: export.Totals.BalanceTotal},
	}
	totalStartRow := row
	for totalIndex, totalRow := range totalRows {
		if err := book.MergeCell(sheet, spreadingCuttingPlanCell(1, row), spreadingCuttingPlanCell(6, row)); err != nil {
			return 0, err
		}
		if err := spreadingCuttingPlanSet(book, row, 1, totalRow.label); err != nil {
			return 0, err
		}
		for index, value := range totalRow.values {
			if value != nil {
				if err := spreadingCuttingPlanSet(book, row, columns.sizeStart+index, value); err != nil {
					return 0, err
				}
				var formula string
				switch totalIndex {
				case 0:
					formula = spreadingCuttingPlanSumFormula(poRows, columns.sizeStart+index)
				case 1:
					formula = spreadingCuttingPlanSumFormula(subtotalRows, columns.sizeStart+index)
				default:
					formula = fmt.Sprintf("%s-%s", spreadingCuttingPlanCell(columns.sizeStart+index, totalStartRow+1), spreadingCuttingPlanCell(columns.sizeStart+index, totalStartRow))
				}
				if err := spreadingCuttingPlanFormula(book, row, columns.sizeStart+index, formula); err != nil {
					return 0, err
				}
			}
		}
		if err := spreadingCuttingPlanSet(book, row, columns.total, totalRow.total); err != nil {
			return 0, err
		}
		if err := spreadingCuttingPlanFormula(book, row, columns.total, fmt.Sprintf("SUM(%s:%s)", spreadingCuttingPlanCell(columns.sizeStart, row), spreadingCuttingPlanCell(columns.sizeEnd, row))); err != nil {
			return 0, err
		}
		if err := book.SetCellStyle(sheet, spreadingCuttingPlanCell(1, row), spreadingCuttingPlanCell(6, row), styles.totalText); err != nil {
			return 0, err
		}
		if err := book.SetCellStyle(sheet, spreadingCuttingPlanCell(columns.sizeStart, row), spreadingCuttingPlanCell(columns.note, row), styles.totalInt); err != nil {
			return 0, err
		}
		if err := book.SetRowHeight(sheet, row, 16.5); err != nil {
			return 0, err
		}
		row++
	}
	return row - 1, nil
}

func applySpreadingCuttingPlanExcelPageLayout(book *excelize.File, columns spreadingCuttingPlanExcelColumns, endRow int) error {
	sheet := spreadingCuttingPlanExportSheetName
	for column := 1; column <= columns.printEnd; column++ {
		width := 12.71
		switch {
		case column == 1:
			width = 18.57
		case column >= columns.sizeStart && column <= columns.sizeEnd:
			width = 9
		case column == columns.balance:
			width = 15
		case column == columns.note:
			width = 20
		}
		name, _ := excelize.ColumnNumberToName(column)
		if err := book.SetColWidth(sheet, name, name, width); err != nil {
			return err
		}
	}
	if err := book.SetSheetView(sheet, 0, &excelize.ViewOptions{ShowGridLines: spreadingCuttingPlanBoolPtr(false), ZoomScale: spreadingCuttingPlanFloat64Ptr(80)}); err != nil {
		return err
	}
	if err := book.SetPageLayout(sheet, &excelize.PageLayoutOptions{
		Orientation: spreadingCuttingPlanStringPtr("landscape"),
		Size:        spreadingCuttingPlanIntPtr(9),
		FitToWidth:  spreadingCuttingPlanIntPtr(1),
		FitToHeight: spreadingCuttingPlanIntPtr(0),
	}); err != nil {
		return err
	}
	if err := book.SetPageMargins(sheet, &excelize.PageLayoutMarginsOptions{
		Left: spreadingCuttingPlanFloat64Ptr(0.25), Right: spreadingCuttingPlanFloat64Ptr(0.25),
		Top: spreadingCuttingPlanFloat64Ptr(0.35), Bottom: spreadingCuttingPlanFloat64Ptr(0.35),
		Header: spreadingCuttingPlanFloat64Ptr(0.1), Footer: spreadingCuttingPlanFloat64Ptr(0.1),
		Horizontally: spreadingCuttingPlanBoolPtr(true),
	}); err != nil {
		return err
	}
	printArea := fmt.Sprintf("'%s'!$A$1:$%s$%d", sheet, spreadingCuttingPlanColumn(columns.printEnd), endRow)
	return book.SetDefinedName(&excelize.DefinedName{Name: "_xlnm.Print_Area", RefersTo: printArea, Scope: sheet})
}

func spreadingCuttingPlanSet(book *excelize.File, row, column int, value any) error {
	return book.SetCellValue(spreadingCuttingPlanExportSheetName, spreadingCuttingPlanCell(column, row), value)
}

func spreadingCuttingPlanFormula(book *excelize.File, row, column int, formula string) error {
	return book.SetCellFormula(spreadingCuttingPlanExportSheetName, spreadingCuttingPlanCell(column, row), formula)
}

func spreadingCuttingPlanSumFormula(rows []int, column int) string {
	references := make([]string, len(rows))
	for index, row := range rows {
		references[index] = spreadingCuttingPlanCell(column, row)
	}
	return "SUM(" + strings.Join(references, ",") + ")"
}

func spreadingCuttingPlanCell(column, row int) string {
	cell, _ := excelize.CoordinatesToCellName(column, row)
	return cell
}

func spreadingCuttingPlanColumn(column int) string {
	name, _ := excelize.ColumnNumberToName(column)
	return name
}

func spreadingCuttingPlanInt64Values(values []int64) []any {
	result := make([]any, len(values))
	for index, value := range values {
		result[index] = value
	}
	return result
}

func spreadingCuttingPlanOptionalInt64Values(values []*int64) []any {
	result := make([]any, len(values))
	for index, value := range values {
		if value != nil {
			result[index] = *value
		}
	}
	return result
}

func buildSpreadingCuttingPlanExportFileName(export *model.SpreadingCuttingPlanExport) string {
	parts := []string{"SPREADING_CUTTING_PLAN"}
	for _, value := range []string{export.Header.DocumentNumber, export.Header.Style, export.Header.Model} {
		if sanitized := spreadingCuttingPlanExportFileSegment(value); sanitized != "" {
			parts = append(parts, sanitized)
		}
	}
	return strings.Join(parts, "_") + ".xlsx"
}

func spreadingCuttingPlanExportFileSegment(value string) string {
	var builder strings.Builder
	lastUnderscore := false
	for _, character := range strings.TrimSpace(value) {
		switch {
		case character >= 'a' && character <= 'z', character >= 'A' && character <= 'Z', character >= '0' && character <= '9':
			builder.WriteRune(character)
			lastUnderscore = false
		default:
			if !lastUnderscore {
				builder.WriteByte('_')
				lastUnderscore = true
			}
		}
	}
	return strings.Trim(builder.String(), "_")
}

func spreadingCuttingPlanStringPtr(value string) *string    { return &value }
func spreadingCuttingPlanIntPtr(value int) *int             { return &value }
func spreadingCuttingPlanBoolPtr(value bool) *bool          { return &value }
func spreadingCuttingPlanFloat64Ptr(value float64) *float64 { return &value }
