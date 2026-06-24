package usecase

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"permatatex-inventory/internal/model"
	excelexporter "permatatex-inventory/pkg/exporter/excel"
)

const (
	workOrderExportTemplateName = "xlsx/template_wo.xlsx"
	workOrderTemplateMaxSizes   = 6

	workOrderShellStartRow     = 14
	workOrderShellMiddleRow    = 16
	workOrderShellLastRow      = 17
	workOrderShellBaseCapacity = 4
	workOrderShellTotalBaseRow = 18
	workOrderTrimHeaderBaseRow = 21
	workOrderTrimStartBaseRow  = 22
	workOrderTrimBaseCapacity  = 12
	workOrderTrimDuplicateRow  = 33
	workOrderTrimInsertBefore  = 34
	workOrderFooterApproveRow  = 35
	workOrderFooterPreparedRow = 36
)

var errWorkOrderExportTooManySizes = errors.New("work order export supports up to 6 size columns")

type WorkOrderExcelExportUseCase struct {
	renderer         *excelexporter.Renderer
	workOrderUseCase *WorkOrderProductionUseCase
}

type workOrderExcelLayout struct {
	shellExtraRows    int
	trimExtraRows     int
	shellTotalRow     int
	trimHeaderRow     int
	trimStartRow      int
	trimDuplicateRow  int
	trimInsertBefore  int
	footerApproveRow  int
	footerPreparedRow int
}

type workOrderExcelShellRow struct {
	label     string
	desc      string
	cons      float64
	color     string
	sizeQtys  []int32
	totalQty  int32
	allow     int32
	totalCons float64
}

type workOrderExcelTrimRow struct {
	number    int
	item      string
	desc      string
	color     string
	code      string
	cons      float64
	qty       int32
	total     float64
	allowText string
	totalCons int64
	uom       string
	position  string
	by        string
}

func NewWorkOrderExcelExportUseCase(
	renderer *excelexporter.Renderer,
	workOrderUseCase *WorkOrderProductionUseCase,
) (*WorkOrderExcelExportUseCase, error) {
	if renderer == nil {
		return nil, errors.New("excel renderer is required")
	}
	if workOrderUseCase == nil {
		return nil, errors.New("work order usecase is required")
	}

	return &WorkOrderExcelExportUseCase{
		renderer:         renderer,
		workOrderUseCase: workOrderUseCase,
	}, nil
}

func (u *WorkOrderExcelExportUseCase) ExportByID(ctx context.Context, id int32) (*model.ExportedFile, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	detail, err := u.workOrderUseCase.GetWorkOrderDetail(ctx, id, nil)
	if err != nil {
		return nil, err
	}

	sizeHeaders, err := buildWorkOrderExportSizeHeaders(detail.Shells)
	if err != nil {
		return nil, err
	}

	shellRows, totalSizeQtys, totalQty := buildWorkOrderExportShellRows(detail.Shells, sizeHeaders)
	if len(shellRows) == 0 {
		return nil, fmt.Errorf("%w: no shell rows available", ErrWorkOrderValidation)
	}

	trimRows := buildWorkOrderExportTrimRows(detail)

	workbook, err := u.renderer.OpenTemplate(workOrderExportTemplateName)
	if err != nil {
		return nil, fmt.Errorf("open work order export template: %w", err)
	}
	defer func() {
		_ = workbook.Close()
	}()

	sheetName := workbook.GetSheetName(0)
	if sheetName == "" {
		return nil, fmt.Errorf("open work order export template: empty sheet name")
	}

	layout, err := prepareWorkOrderExportLayout(workbook, sheetName, len(shellRows), len(trimRows))
	if err != nil {
		return nil, fmt.Errorf("prepare work order export layout: %w", err)
	}

	if err := writeWorkOrderExportHeader(workbook, sheetName, detail); err != nil {
		return nil, fmt.Errorf("write work order export header: %w", err)
	}
	if err := writeWorkOrderExportShellSection(workbook, sheetName, layout, sizeHeaders, shellRows, totalSizeQtys, totalQty); err != nil {
		return nil, fmt.Errorf("write work order export shell section: %w", err)
	}
	if err := writeWorkOrderExportTrimSection(workbook, sheetName, layout, trimRows); err != nil {
		return nil, fmt.Errorf("write work order export trim section: %w", err)
	}
	if err := writeWorkOrderExportFooter(workbook, sheetName, layout, detail.CreatedAt); err != nil {
		return nil, fmt.Errorf("write work order export footer: %w", err)
	}

	var buffer bytes.Buffer
	if err := workbook.Write(&buffer); err != nil {
		return nil, fmt.Errorf("write work order export workbook: %w", err)
	}

	return &model.ExportedFile{
		FileName:    buildWorkOrderExportFileName(detail),
		ContentType: excelContentType,
		Content:     buffer.Bytes(),
	}, nil
}

func prepareWorkOrderExportLayout(workbook *excelize.File, sheetName string, shellRowCount int, trimRowCount int) (workOrderExcelLayout, error) {
	shellExtraRows := max(0, shellRowCount-workOrderShellBaseCapacity)
	for range shellExtraRows {
		if err := workbook.DuplicateRowTo(sheetName, workOrderShellMiddleRow, workOrderShellLastRow); err != nil {
			return workOrderExcelLayout{}, err
		}
	}

	trimExtraRows := max(0, trimRowCount-workOrderTrimBaseCapacity)
	trimDuplicateRow := workOrderTrimDuplicateRow + shellExtraRows
	trimInsertBefore := workOrderTrimInsertBefore + shellExtraRows
	for range trimExtraRows {
		if err := workbook.DuplicateRowTo(sheetName, trimDuplicateRow, trimInsertBefore); err != nil {
			return workOrderExcelLayout{}, err
		}
	}

	return workOrderExcelLayout{
		shellExtraRows:    shellExtraRows,
		trimExtraRows:     trimExtraRows,
		shellTotalRow:     workOrderShellTotalBaseRow + shellExtraRows,
		trimHeaderRow:     workOrderTrimHeaderBaseRow + shellExtraRows,
		trimStartRow:      workOrderTrimStartBaseRow + shellExtraRows,
		trimDuplicateRow:  trimDuplicateRow,
		trimInsertBefore:  trimInsertBefore,
		footerApproveRow:  workOrderFooterApproveRow + shellExtraRows + trimExtraRows,
		footerPreparedRow: workOrderFooterPreparedRow + shellExtraRows + trimExtraRows,
	}, nil
}

func buildWorkOrderExportSizeHeaders(shells []model.WorkOrderShellResponse) ([]string, error) {
	sizeHeaders := make([]string, 0, workOrderTemplateMaxSizes)
	seen := make(map[string]struct{})

	for _, shell := range shells {
		for _, size := range shell.Sizes {
			label := strings.TrimSpace(size.Size)
			if label == "" {
				continue
			}
			key := strings.ToUpper(label)
			if _, ok := seen[key]; ok {
				continue
			}

			sizeHeaders = append(sizeHeaders, label)
			seen[key] = struct{}{}
			if len(sizeHeaders) > workOrderTemplateMaxSizes {
				return nil, errWorkOrderExportTooManySizes
			}
		}
	}

	if len(sizeHeaders) == 0 {
		return nil, ErrWorkOrderValidation
	}

	return sizeHeaders, nil
}

func buildWorkOrderExportShellRows(
	shells []model.WorkOrderShellResponse,
	sizeHeaders []string,
) ([]workOrderExcelShellRow, []int32, int32) {
	rows := make([]workOrderExcelShellRow, 0, len(shells))
	totalSizeQtys := make([]int32, len(sizeHeaders))
	totalQty := int32(0)

	for _, shell := range shells {
		row := workOrderExcelShellRow{
			label:    buildWorkOrderShellLabel(shell.MaterialType),
			desc:     strings.TrimSpace(shell.Deskripsi),
			cons:     shell.Cons,
			color:    strings.TrimSpace(shell.Color),
			sizeQtys: make([]int32, len(sizeHeaders)),
			allow:    shell.Allow,
		}

		sizeIndex := make(map[string]int, len(sizeHeaders))
		for i, label := range sizeHeaders {
			sizeIndex[strings.ToUpper(label)] = i
		}

		for _, size := range shell.Sizes {
			idx, ok := sizeIndex[strings.ToUpper(strings.TrimSpace(size.Size))]
			if !ok {
				continue
			}
			row.sizeQtys[idx] = size.Qty
			row.totalQty += size.Qty
		}

		row.totalCons = float64(row.totalQty) * row.cons
		rows = append(rows, row)

		if strings.EqualFold(strings.TrimSpace(shell.MaterialType), "fabric") {
			totalQty += row.totalQty
			for i, qty := range row.sizeQtys {
				totalSizeQtys[i] += qty
			}
		}
	}

	return rows, totalSizeQtys, totalQty
}

func buildWorkOrderExportTrimRows(detail *model.WorkOrderDetailResponse) []workOrderExcelTrimRow {
	rows := make([]workOrderExcelTrimRow, 0, len(detail.Trims))
	itemNumbers := make(map[string]int)
	nextNumber := 1

	for _, trim := range detail.Trims {
		key := strings.ToUpper(strings.TrimSpace(trim.Item))
		number, ok := itemNumbers[key]
		if !ok {
			number = nextNumber
			itemNumbers[key] = number
			nextNumber++
		}

		total := trim.Cons * float64(trim.Qty)
		allowValue := total * (float64(trim.Allow) / 100)

		rows = append(rows, workOrderExcelTrimRow{
			number:    number,
			item:      strings.TrimSpace(trim.Item),
			desc:      strings.TrimSpace(trim.Description),
			color:     strings.TrimSpace(trim.Color),
			code:      strings.TrimSpace(trim.Code),
			cons:      trim.Cons,
			qty:       trim.Qty,
			total:     total,
			allowText: fmt.Sprintf("%d%%", trim.Allow),
			totalCons: roundFloatToInt64(total + allowValue),
			uom:       strings.TrimSpace(trim.UOM),
			position:  strings.TrimSpace(trim.Position),
			by:        buildWorkOrderTrimByLabel(trim.ProvidedBy, detail.Buyer),
		})
	}

	return rows
}

func writeWorkOrderExportHeader(workbook *excelize.File, sheetName string, detail *model.WorkOrderDetailResponse) error {
	values := map[string]any{
		"D3": strings.TrimSpace(detail.Buyer),
		"D4": strings.TrimSpace(detail.Model),
		"D5": strings.TrimSpace(detail.POClientItemStyle),
		"D6": formatWorkOrderInteger(detail.Qty),
		"D7": buildWorkOrderFOBCMTLabel(detail.FOBCMT),
		"D8": strings.TrimSpace(detail.Delivery),
	}

	for cell, value := range values {
		if err := workbook.SetCellValue(sheetName, cell, value); err != nil {
			return err
		}
	}

	return nil
}

func writeWorkOrderExportShellSection(
	workbook *excelize.File,
	sheetName string,
	layout workOrderExcelLayout,
	sizeHeaders []string,
	rows []workOrderExcelShellRow,
	totalSizeQtys []int32,
	totalQty int32,
) error {
	fabricRows, interliningRows := splitWorkOrderShellRows(rows)

	sizeHeaderCells := []string{"E13", "F13", "G13", "H13", "I13", "J13"}
	for i, cell := range sizeHeaderCells {
		value := ""
		if i < len(sizeHeaders) {
			value = sizeHeaders[i]
		}
		if err := workbook.SetCellValue(sheetName, cell, value); err != nil {
			return err
		}
	}

	if err := workbook.UnmergeCell(sheetName, "A14", "A16"); err != nil {
		return err
	}
	if err := workbook.UnmergeCell(sheetName, "B14", "B16"); err != nil {
		return err
	}

	currentRow := workOrderShellStartRow
	if len(fabricRows) > 0 {
		fabricStartRow := currentRow
		fabricEndRow := fabricStartRow + len(fabricRows) - 1

		if err := workbook.SetCellValue(sheetName, "A"+strconv.Itoa(fabricStartRow), "SHELL BODY"); err != nil {
			return err
		}
		if err := workbook.SetCellValue(sheetName, "B"+strconv.Itoa(fabricStartRow), fabricRows[0].desc); err != nil {
			return err
		}
		if len(fabricRows) > 1 {
			if err := workbook.MergeCell(sheetName, "A"+strconv.Itoa(fabricStartRow), "A"+strconv.Itoa(fabricEndRow)); err != nil {
				return err
			}
			if err := workbook.MergeCell(sheetName, "B"+strconv.Itoa(fabricStartRow), "B"+strconv.Itoa(fabricEndRow)); err != nil {
				return err
			}
		}

		for i, row := range fabricRows {
			rowIndex := fabricStartRow + i
			if err := writeWorkOrderShellValueRow(workbook, sheetName, rowIndex, row); err != nil {
				return err
			}
		}
		currentRow = fabricEndRow + 1
	}

	for _, row := range interliningRows {
		if err := workbook.SetCellValue(sheetName, "A"+strconv.Itoa(currentRow), row.label); err != nil {
			return err
		}
		if err := workbook.SetCellValue(sheetName, "B"+strconv.Itoa(currentRow), row.desc); err != nil {
			return err
		}
		if err := writeWorkOrderShellValueRow(workbook, sheetName, currentRow, row); err != nil {
			return err
		}
		currentRow++
	}

	if layout.shellExtraRows == 0 {
		for rowIndex := currentRow; rowIndex < workOrderShellTotalBaseRow; rowIndex++ {
			for _, cell := range []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N"} {
				if err := workbook.SetCellValue(sheetName, cell+strconv.Itoa(rowIndex), ""); err != nil {
					return err
				}
			}
		}
	}

	totalRow := layout.shellTotalRow
	for _, item := range []struct {
		cell  string
		value any
	}{
		{cell: "A", value: "TOTAL"},
		{cell: "B", value: "TOTAL"},
		{cell: "C", value: "TOTAL"},
		{cell: "D", value: "TOTAL"},
		{cell: "K", value: formatWorkOrderInteger(totalQty)},
		{cell: "L", value: ""},
		{cell: "M", value: ""},
		{cell: "N", value: ""},
	} {
		if err := workbook.SetCellValue(sheetName, item.cell+strconv.Itoa(totalRow), item.value); err != nil {
			return err
		}
	}

	for sizeIndex := range sizeHeaderCells {
		cell := string(rune('E'+sizeIndex)) + strconv.Itoa(totalRow)
		var value any = ""
		if sizeIndex < len(totalSizeQtys) && totalSizeQtys[sizeIndex] > 0 {
			value = formatWorkOrderInteger(totalSizeQtys[sizeIndex])
		}
		if err := workbook.SetCellValue(sheetName, cell, value); err != nil {
			return err
		}
	}

	return nil
}

func splitWorkOrderShellRows(rows []workOrderExcelShellRow) ([]workOrderExcelShellRow, []workOrderExcelShellRow) {
	fabricRows := make([]workOrderExcelShellRow, 0, len(rows))
	interliningRows := make([]workOrderExcelShellRow, 0, len(rows))

	for _, row := range rows {
		if row.label == "INTERLINING" {
			interliningRows = append(interliningRows, row)
			continue
		}
		fabricRows = append(fabricRows, row)
	}

	return fabricRows, interliningRows
}

func writeWorkOrderShellValueRow(
	workbook *excelize.File,
	sheetName string,
	rowIndex int,
	row workOrderExcelShellRow,
) error {
	for _, item := range []struct {
		cell  string
		value any
	}{
		{cell: "C", value: formatWorkOrderFloat(row.cons, 2)},
		{cell: "D", value: row.color},
		{cell: "K", value: formatWorkOrderInteger(row.totalQty)},
		{cell: "L", value: fmt.Sprintf("%d%%", row.allow)},
		{cell: "M", value: formatWorkOrderFloat(row.totalCons, 1)},
		{cell: "N", value: ""},
	} {
		if err := workbook.SetCellValue(sheetName, item.cell+strconv.Itoa(rowIndex), item.value); err != nil {
			return err
		}
	}

	for sizeIndex := 0; sizeIndex < workOrderTemplateMaxSizes; sizeIndex++ {
		cell := string(rune('E'+sizeIndex)) + strconv.Itoa(rowIndex)
		var value any = ""
		if sizeIndex < len(row.sizeQtys) && row.sizeQtys[sizeIndex] > 0 {
			value = formatWorkOrderInteger(row.sizeQtys[sizeIndex])
		}
		if err := workbook.SetCellValue(sheetName, cell, value); err != nil {
			return err
		}
	}

	return nil
}

func writeWorkOrderExportTrimSection(
	workbook *excelize.File,
	sheetName string,
	layout workOrderExcelLayout,
	rows []workOrderExcelTrimRow,
) error {
	for i, row := range rows {
		rowIndex := layout.trimStartRow + i
		assignments := []struct {
			cell  string
			value any
		}{
			{cell: "A", value: row.number},
			{cell: "B", value: row.item},
			{cell: "C", value: row.desc},
			{cell: "D", value: row.color},
			{cell: "E", value: row.code},
			{cell: "F", value: formatWorkOrderFloat(row.cons, 3)},
			{cell: "G", value: formatWorkOrderInteger(row.qty)},
			{cell: "H", value: formatWorkOrderFloat(row.total, 1)},
			{cell: "I", value: row.allowText},
			{cell: "J", value: formatWorkOrderInteger64(row.totalCons)},
			{cell: "K", value: row.uom},
			{cell: "L", value: row.position},
			{cell: "M", value: row.position},
			{cell: "N", value: row.by},
		}
		for _, item := range assignments {
			if err := workbook.SetCellValue(sheetName, item.cell+strconv.Itoa(rowIndex), item.value); err != nil {
				return err
			}
		}
	}

	return nil
}

func writeWorkOrderExportFooter(
	workbook *excelize.File,
	sheetName string,
	layout workOrderExcelLayout,
	createdAt string,
) error {
	exportDateLabel := ""
	if parsed, err := time.Parse(time.RFC3339, createdAt); err == nil {
		exportDateLabel = fmt.Sprintf("Boyolali, %s", strings.ToUpper(parsed.Format("02-January-2006")))
	}

	values := map[string]any{
		"D" + strconv.Itoa(layout.footerApproveRow):  "Diketahui & disetujui oleh,",
		"M" + strconv.Itoa(layout.footerApproveRow):  exportDateLabel,
		"M" + strconv.Itoa(layout.footerPreparedRow): "Dibuat Oleh,",
	}

	for cell, value := range values {
		if err := workbook.SetCellValue(sheetName, cell, value); err != nil {
			return err
		}
	}

	return nil
}

func buildWorkOrderShellLabel(materialType string) string {
	if strings.EqualFold(strings.TrimSpace(materialType), "interlining") {
		return "INTERLINING"
	}
	return "SHELL BODY"
}

func buildWorkOrderTrimByLabel(providedBy string, buyer string) string {
	if strings.EqualFold(strings.TrimSpace(providedBy), "client") {
		return strings.TrimSpace(buyer)
	}
	return "PERMATATEX"
}

func buildWorkOrderFOBCMTLabel(isFOB bool) string {
	if isFOB {
		return "FOB"
	}
	return "CMT"
}

func buildWorkOrderExportFileName(detail *model.WorkOrderDetailResponse) string {
	parts := []string{
		"WO",
		sanitizeWorkOrderExportFilePart(detail.Buyer),
		sanitizeWorkOrderExportFilePart(detail.Model),
		strconv.FormatInt(int64(detail.ID), 10),
	}

	fileName := strings.Join(parts, "_")
	fileName = strings.Trim(fileName, "_")
	if fileName == "" {
		return "work_order_export.xlsx"
	}

	return fileName + filepath.Ext(workOrderExportTemplateName)
}

func sanitizeWorkOrderExportFilePart(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}

	var builder strings.Builder
	lastUnderscore := false
	for _, r := range trimmed {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			builder.WriteRune(r)
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

func formatWorkOrderInteger(value int32) string {
	return formatWorkOrderInteger64(int64(value))
}

func formatWorkOrderInteger64(value int64) string {
	negative := value < 0
	if negative {
		value = -value
	}

	raw := strconv.FormatInt(value, 10)
	if len(raw) <= 3 {
		if negative {
			return "-" + raw
		}
		return raw
	}

	var parts []string
	for len(raw) > 3 {
		parts = append([]string{raw[len(raw)-3:]}, parts...)
		raw = raw[:len(raw)-3]
	}
	if raw != "" {
		parts = append([]string{raw}, parts...)
	}

	result := strings.Join(parts, ",")
	if negative {
		return "-" + result
	}

	return result
}

func formatWorkOrderFloat(value float64, decimals int) string {
	formatted := strconv.FormatFloat(value, 'f', decimals, 64)
	formatted = strings.TrimRight(formatted, "0")
	formatted = strings.TrimRight(formatted, ".")
	if formatted == "" {
		return "0"
	}
	return formatted
}

func roundFloatToInt64(value float64) int64 {
	if value >= 0 {
		return int64(value + 0.5)
	}
	return int64(value - 0.5)
}
