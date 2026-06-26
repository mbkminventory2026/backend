package usecase

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	"permatatex-inventory/internal/model"
	"permatatex-inventory/pkg/exporter/excel"

	"github.com/xuri/excelize/v2"
)

const (
	packingListExportTemplateName = "xlsx/template_packing_list.xlsx"
	packingListItemBaseCapacity   = 2
	packingListItemStartRow       = 11
	packingListSummaryStartRow    = 13
)

var packingListSizeColumnOrder = []string{"F", "G", "H", "I", "J", "K"}

var packingListSizeRank = map[string]int{
	"XXS": 0,
	"XS":  1,
	"S":   2,
	"M":   3,
	"L":   4,
	"XL":  5,
	"XXL": 6,
	"3XL": 7,
	"4XL": 8,
}

type packingListExcelLayout struct {
	itemExtraRows int
	summaryRow    int
	footerDateRow int
}

type packingListExportSize struct {
	Key   string
	Label string
}

type PackingListExcelExportUseCase struct {
	renderer              *excel.Renderer
	warehouseDeliveryUC   *WarehouseDeliveryUseCase
	workOrderProductionUC *WorkOrderProductionUseCase
}

func NewPackingListExcelExportUseCase(
	renderer *excel.Renderer,
	warehouseDeliveryUC *WarehouseDeliveryUseCase,
	workOrderProductionUC *WorkOrderProductionUseCase,
) (*PackingListExcelExportUseCase, error) {
	if renderer == nil {
		return nil, errors.New("excel renderer is required")
	}
	if warehouseDeliveryUC == nil {
		return nil, errors.New("warehouse delivery usecase is required")
	}
	if workOrderProductionUC == nil {
		return nil, errors.New("work order production usecase is required")
	}

	return &PackingListExcelExportUseCase{
		renderer:              renderer,
		warehouseDeliveryUC:   warehouseDeliveryUC,
		workOrderProductionUC: workOrderProductionUC,
	}, nil
}

func (u *PackingListExcelExportUseCase) ExportByID(ctx context.Context, id int32, idMitra *int32) (*model.ExportedFile, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	detail, err := u.warehouseDeliveryUC.GetPackingListDetail(ctx, id, idMitra)
	if err != nil {
		return nil, err
	}

	workOrder, err := u.workOrderProductionUC.GetWorkOrderDetail(ctx, detail.IDWO, idMitra)
	if err != nil {
		return nil, err
	}

	workbook, err := u.renderer.OpenTemplate(packingListExportTemplateName)
	if err != nil {
		return nil, fmt.Errorf("open packing list export template: %w", err)
	}
	defer func() {
		_ = workbook.Close()
	}()

	sheetName := workbook.GetSheetName(0)
	if sheetName == "" {
		return nil, fmt.Errorf("open packing list export template: empty sheet name")
	}

	layout, err := preparePackingListExportLayout(workbook, sheetName, len(detail.Items))
	if err != nil {
		return nil, fmt.Errorf("prepare packing list export layout: %w", err)
	}

	sizes := collectPackingListSizes(detail)
	if err := writePackingListExportHeader(workbook, sheetName, detail, workOrder, sizes); err != nil {
		return nil, fmt.Errorf("write packing list export header: %w", err)
	}
	if err := writePackingListExportItems(workbook, sheetName, detail, sizes); err != nil {
		return nil, fmt.Errorf("write packing list export items: %w", err)
	}
	if err := writePackingListExportSummary(workbook, sheetName, layout, detail, sizes); err != nil {
		return nil, fmt.Errorf("write packing list export summary: %w", err)
	}
	if err := writePackingListExportFooter(workbook, sheetName, layout, detail); err != nil {
		return nil, fmt.Errorf("write packing list export footer: %w", err)
	}

	var buffer bytes.Buffer
	if err := workbook.Write(&buffer); err != nil {
		return nil, fmt.Errorf("write packing list export workbook: %w", err)
	}

	return &model.ExportedFile{
		FileName:    buildPackingListExportFileName(detail, workOrder),
		ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		Content:     buffer.Bytes(),
	}, nil
}

func preparePackingListExportLayout(workbook *excelize.File, sheetName string, itemCount int) (packingListExcelLayout, error) {
	itemExtraRows := max(0, itemCount-packingListItemBaseCapacity)
	for range itemExtraRows {
		if err := workbook.DuplicateRowTo(sheetName, packingListItemStartRow, packingListSummaryStartRow); err != nil {
			return packingListExcelLayout{}, err
		}
	}

	layout := packingListExcelLayout{
		itemExtraRows: itemExtraRows,
		summaryRow:    packingListSummaryStartRow + itemExtraRows,
		footerDateRow: 14 + itemExtraRows,
	}

	if err := normalizePackingListSizeColumns(workbook, sheetName, layout); err != nil {
		return packingListExcelLayout{}, err
	}

	return layout, nil
}

func normalizePackingListSizeColumns(workbook *excelize.File, sheetName string, layout packingListExcelLayout) error {
	rows := []int{10, layout.summaryRow}
	for row := packingListItemStartRow; row < layout.summaryRow; row++ {
		rows = append(rows, row)
	}

	for _, row := range rows {
		leftCell := fmt.Sprintf("I%d", row)
		rightCell := fmt.Sprintf("J%d", row)

		styleID, err := workbook.GetCellStyle(sheetName, leftCell)
		if err != nil {
			return err
		}

		if err := workbook.UnmergeCell(sheetName, leftCell, rightCell); err != nil {
			return err
		}
		if err := workbook.SetCellStyle(sheetName, leftCell, rightCell, styleID); err != nil {
			return err
		}
	}

	return nil
}

func collectPackingListSizes(detail *model.PackingListDetailResponse) []packingListExportSize {
	type sizeEntry struct {
		key   string
		label string
		rank  int
	}

	seen := make(map[string]sizeEntry)

	addSize := func(size string) {
		label := strings.TrimSpace(size)
		if label == "" {
			return
		}
		key := strings.ToUpper(label)
		if _, exists := seen[key]; exists {
			return
		}
		rank, ok := packingListSizeRank[key]
		if !ok {
			rank = 100 + len(seen)
		}
		seen[key] = sizeEntry{key: key, label: label, rank: rank}
	}

	for _, item := range detail.Items {
		for _, size := range item.Sizes {
			addSize(size.Size)
		}
	}
	for _, size := range detail.RejectSizes {
		addSize(size.Size)
	}

	ordered := make([]sizeEntry, 0, len(seen))
	for _, entry := range seen {
		ordered = append(ordered, entry)
	}

	for i := 0; i < len(ordered); i++ {
		for j := i + 1; j < len(ordered); j++ {
			if ordered[j].rank < ordered[i].rank || (ordered[j].rank == ordered[i].rank && ordered[j].label < ordered[i].label) {
				ordered[i], ordered[j] = ordered[j], ordered[i]
			}
		}
	}

	result := make([]packingListExportSize, 0, min(len(ordered), len(packingListSizeColumnOrder)))
	for i := 0; i < len(ordered) && i < len(packingListSizeColumnOrder); i++ {
		result = append(result, packingListExportSize{
			Key:   ordered[i].key,
			Label: ordered[i].label,
		})
	}
	return result
}

func writePackingListExportHeader(
	workbook *excelize.File,
	sheetName string,
	detail *model.PackingListDetailResponse,
	workOrder *model.WorkOrderDetailResponse,
	sizes []packingListExportSize,
) error {
	totalBoxes := int32(0)
	totalGarments := int32(0)
	for _, item := range detail.Items {
		totalBoxes += item.QtyBox
		totalGarments += item.QtyBox * item.QtyPerBox
	}

	values := map[string]any{
		"C3": detail.Buyer,
		"C4": detail.Model,
		"C5": strings.TrimSpace(workOrder.PONumber),
		"C6": totalBoxes,
		"C7": detail.TotalGarmentPerBox,
		"C8": totalGarments,
	}

	for cell, value := range values {
		if err := workbook.SetCellValue(sheetName, cell, value); err != nil {
			return err
		}
	}

	for index, column := range packingListSizeColumnOrder {
		label := ""
		if index < len(sizes) {
			label = sizes[index].Label
		}
		if err := workbook.SetCellValue(sheetName, column+"10", label); err != nil {
			return err
		}
	}

	return nil
}

func writePackingListExportItems(
	workbook *excelize.File,
	sheetName string,
	detail *model.PackingListDetailResponse,
	sizes []packingListExportSize,
) error {
	capacity := max(packingListItemBaseCapacity, len(detail.Items))
	for i := 0; i < capacity; i++ {
		rowIndex := packingListItemStartRow + i
		assignments := map[string]any{
			"A": "",
			"B": "",
			"D": "",
			"E": "",
			"L": "",
			"M": "",
		}

		for _, column := range packingListSizeColumnOrder {
			assignments[column] = ""
		}

		if i < len(detail.Items) {
			item := detail.Items[i]
			assignments["A"] = strings.TrimSpace(item.Color)
			assignments["B"] = item.QtyBox
			assignments["D"] = item.QtyPerBox
			assignments["E"] = buildPackingListBoxNumber(item.BoxNoStart, item.BoxNoEnd)
			assignments["L"] = item.QtyBox * item.QtyPerBox
			assignments["M"] = strings.TrimSpace(item.Note)

			sizeQtyMap := make(map[string]int32)
			for _, size := range item.Sizes {
				sizeQtyMap[strings.ToUpper(strings.TrimSpace(size.Size))] = size.Qty
			}

			for index, size := range sizes {
				assignments[packingListSizeColumnOrder[index]] = sizeQtyMap[size.Key]
			}
		}

		for column, value := range assignments {
			if err := workbook.SetCellValue(sheetName, column+fmt.Sprintf("%d", rowIndex), value); err != nil {
				return err
			}
		}
	}

	return nil
}

func writePackingListExportSummary(
	workbook *excelize.File,
	sheetName string,
	layout packingListExcelLayout,
	detail *model.PackingListDetailResponse,
	sizes []packingListExportSize,
) error {
	totalBoxes := int32(0)
	totalGarments := int32(0)
	sizeTotals := make(map[string]int32)

	for _, item := range detail.Items {
		totalBoxes += item.QtyBox
		totalGarments += item.QtyBox * item.QtyPerBox
		for _, size := range item.Sizes {
			key := strings.ToUpper(strings.TrimSpace(size.Size))
			sizeTotals[key] += size.Qty
		}
	}

	values := map[string]any{
		"B": totalBoxes,
		"D": detail.TotalGarmentPerBox,
		"L": totalGarments,
	}
	for index, size := range sizes {
		values[packingListSizeColumnOrder[index]] = sizeTotals[size.Key]
	}

	for column, value := range values {
		if err := workbook.SetCellValue(sheetName, column+fmt.Sprintf("%d", layout.summaryRow), value); err != nil {
			return err
		}
	}

	return nil
}

func writePackingListExportFooter(
	workbook *excelize.File,
	sheetName string,
	layout packingListExcelLayout,
	detail *model.PackingListDetailResponse,
) error {
	return workbook.SetCellValue(sheetName, "C"+fmt.Sprintf("%d", layout.footerDateRow), "Boyolali, "+formatPOInternalExportDate(detail.CreatedAt))
}

func buildPackingListBoxNumber(start, end int32) string {
	if start == end {
		return fmt.Sprintf("%d", start)
	}
	return fmt.Sprintf("%d-%d", start, end)
}

func buildPackingListExportFileName(detail *model.PackingListDetailResponse, workOrder *model.WorkOrderDetailResponse) string {
	baseName := fmt.Sprintf(
		"PACKING_LIST_%s_%d",
		sanitizeExportSegment(strings.TrimSpace(workOrder.Model)),
		detail.ID,
	)
	return ensureExportExtension(baseName, ".xlsx")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
