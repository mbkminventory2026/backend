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
	suratJalanInternalExportTemplateName = "xlsx/template_surat_jalan.xlsx"
	suratJalanInternalItemStartRow       = 11
	suratJalanInternalItemBaseCapacity   = 9
	suratJalanInternalItemTemplateRow    = 19
	suratJalanInternalSubtotalBaseRow    = 20
)

type suratJalanInternalExcelLayout struct {
	itemExtraRows int
	subtotalRow   int
	footerDateRow int
}

type SuratJalanInternalExcelExportUseCase struct {
	renderer              *excel.Renderer
	warehouseDeliveryUC   *WarehouseDeliveryUseCase
	workOrderProductionUC *WorkOrderProductionUseCase
}

func NewSuratJalanInternalExcelExportUseCase(
	renderer *excel.Renderer,
	warehouseDeliveryUC *WarehouseDeliveryUseCase,
	workOrderProductionUC *WorkOrderProductionUseCase,
) (*SuratJalanInternalExcelExportUseCase, error) {
	if renderer == nil {
		return nil, errors.New("excel renderer is required")
	}
	if warehouseDeliveryUC == nil {
		return nil, errors.New("warehouse delivery usecase is required")
	}
	if workOrderProductionUC == nil {
		return nil, errors.New("work order production usecase is required")
	}

	return &SuratJalanInternalExcelExportUseCase{
		renderer:              renderer,
		warehouseDeliveryUC:   warehouseDeliveryUC,
		workOrderProductionUC: workOrderProductionUC,
	}, nil
}

func (u *SuratJalanInternalExcelExportUseCase) ExportByID(ctx context.Context, id int32) (*model.ExportedFile, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	detail, err := u.warehouseDeliveryUC.GetSuratJalanInternalDetail(ctx, id)
	if err != nil {
		return nil, err
	}

	var workOrder *model.WorkOrderDetailResponse
	if detail.IDWO != nil && *detail.IDWO > 0 {
		workOrder, err = u.workOrderProductionUC.GetWorkOrderDetail(ctx, *detail.IDWO, nil)
		if err != nil {
			return nil, err
		}
	}

	workbook, err := u.renderer.OpenTemplate(suratJalanInternalExportTemplateName)
	if err != nil {
		return nil, fmt.Errorf("open surat jalan internal export template: %w", err)
	}
	defer func() {
		_ = workbook.Close()
	}()

	sheetName := workbook.GetSheetName(0)
	if sheetName == "" {
		return nil, fmt.Errorf("open surat jalan internal export template: empty sheet name")
	}

	layout, err := prepareSuratJalanInternalExportLayout(workbook, sheetName, len(detail.WOShells))
	if err != nil {
		return nil, fmt.Errorf("prepare surat jalan internal export layout: %w", err)
	}

	if err := writeSuratJalanInternalExportHeader(workbook, sheetName, detail, workOrder); err != nil {
		return nil, fmt.Errorf("write surat jalan internal export header: %w", err)
	}
	if err := writeSuratJalanInternalExportItems(workbook, sheetName, layout, detail.WOShells); err != nil {
		return nil, fmt.Errorf("write surat jalan internal export items: %w", err)
	}
	if err := writeSuratJalanInternalExportFooter(workbook, sheetName, layout, detail); err != nil {
		return nil, fmt.Errorf("write surat jalan internal export footer: %w", err)
	}

	var buffer bytes.Buffer
	if err := workbook.Write(&buffer); err != nil {
		return nil, fmt.Errorf("write surat jalan internal export workbook: %w", err)
	}

	return &model.ExportedFile{
		FileName:    buildSuratJalanInternalExportFileName(detail),
		ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		Content:     buffer.Bytes(),
	}, nil
}

func prepareSuratJalanInternalExportLayout(workbook *excelize.File, sheetName string, itemCount int) (suratJalanInternalExcelLayout, error) {
	itemExtraRows := max(0, itemCount-suratJalanInternalItemBaseCapacity)
	for range itemExtraRows {
		if err := workbook.DuplicateRowTo(sheetName, suratJalanInternalItemTemplateRow, suratJalanInternalSubtotalBaseRow); err != nil {
			return suratJalanInternalExcelLayout{}, err
		}
	}

	return suratJalanInternalExcelLayout{
		itemExtraRows: itemExtraRows,
		subtotalRow:   suratJalanInternalSubtotalBaseRow + itemExtraRows,
		footerDateRow: 21 + itemExtraRows,
	}, nil
}

func writeSuratJalanInternalExportHeader(
	workbook *excelize.File,
	sheetName string,
	detail *model.SuratJalanInternalDetailResponse,
	workOrder *model.WorkOrderDetailResponse,
) error {
	buyer := strings.TrimSpace(detail.Buyer)
	style := strings.TrimSpace(detail.Model)
	poNumber := ""
	totalQty := detail.WOQty

	if workOrder != nil {
		if buyer == "" {
			buyer = strings.TrimSpace(workOrder.Buyer)
		}
		if strings.TrimSpace(workOrder.POClientItemStyle) != "" {
			style = strings.TrimSpace(workOrder.POClientItemStyle)
		}
		poNumber = strings.TrimSpace(workOrder.PONumber)
		if totalQty == 0 {
			totalQty = workOrder.Qty
		}
	}

	if totalQty == 0 {
		for _, item := range detail.WOShells {
			totalQty += item.Qty
		}
	}

	values := map[string]any{
		"C3": buyer,
		"C4": style,
		"C5": poNumber,
		"C6": totalQty,
	}

	for cell, value := range values {
		if err := workbook.SetCellValue(sheetName, cell, value); err != nil {
			return err
		}
	}

	return nil
}

func writeSuratJalanInternalExportItems(
	workbook *excelize.File,
	sheetName string,
	layout suratJalanInternalExcelLayout,
	items []model.SuratJalanInternalShellRow,
) error {
	for row := suratJalanInternalItemStartRow; row < layout.subtotalRow; row++ {
		if err := clearSuratJalanInternalExportRow(workbook, sheetName, row); err != nil {
			return err
		}
	}
	if err := clearSuratJalanInternalExportRow(workbook, sheetName, layout.subtotalRow); err != nil {
		return err
	}

	totalQty := int32(0)
	for index, item := range items {
		row := suratJalanInternalItemStartRow + index
		totalQty += item.Qty

		if err := disableSuratJalanInternalDescriptionShrink(workbook, sheetName, row); err != nil {
			return err
		}

		values := map[string]any{
			fmt.Sprintf("A%d", row): item.No,
			fmt.Sprintf("B%d", row): strings.TrimSpace(item.Deskripsi),
			fmt.Sprintf("F%d", row): item.Qty,
			fmt.Sprintf("L%d", row): strings.TrimSpace(item.Note),
		}

		for cell, value := range values {
			if err := workbook.SetCellValue(sheetName, cell, value); err != nil {
				return err
			}
		}
	}

	if err := workbook.SetCellValue(sheetName, fmt.Sprintf("A%d", layout.subtotalRow), "SUB TOTAL"); err != nil {
		return err
	}
	if err := workbook.SetCellValue(sheetName, fmt.Sprintf("F%d", layout.subtotalRow), totalQty); err != nil {
		return err
	}
	if err := workbook.SetCellValue(sheetName, fmt.Sprintf("L%d", layout.subtotalRow), ""); err != nil {
		return err
	}

	return nil
}

func disableSuratJalanInternalDescriptionShrink(workbook *excelize.File, sheetName string, row int) error {
	styleID, err := workbook.GetCellStyle(sheetName, fmt.Sprintf("B%d", row))
	if err != nil {
		return err
	}

	style, err := workbook.GetStyle(styleID)
	if err != nil {
		return err
	}
	if style == nil {
		return nil
	}

	if style.Alignment == nil {
		style.Alignment = &excelize.Alignment{}
	}
	style.Alignment.ShrinkToFit = false

	updatedStyleID, err := workbook.NewStyle(style)
	if err != nil {
		return err
	}

	return workbook.SetCellStyle(sheetName, fmt.Sprintf("B%d", row), fmt.Sprintf("E%d", row), updatedStyleID)
}

func writeSuratJalanInternalExportFooter(
	workbook *excelize.File,
	sheetName string,
	layout suratJalanInternalExcelLayout,
	detail *model.SuratJalanInternalDetailResponse,
) error {
	dateLabel := "Boyolali, " + formatPOInternalExportDate(detail.CreatedAt)
	leftDateCell := fmt.Sprintf("A%d", layout.footerDateRow)
	rightDateCell := fmt.Sprintf("J%d", layout.footerDateRow)

	if err := workbook.SetCellValue(sheetName, leftDateCell, dateLabel); err != nil {
		return err
	}
	if err := workbook.SetCellValue(sheetName, rightDateCell, dateLabel); err != nil {
		return err
	}

	return nil
}

func clearSuratJalanInternalExportRow(workbook *excelize.File, sheetName string, row int) error {
	for col := 1; col <= 14; col++ {
		cell, err := excelize.CoordinatesToCellName(col, row)
		if err != nil {
			return err
		}
		if err := workbook.SetCellValue(sheetName, cell, ""); err != nil {
			return err
		}
	}
	return nil
}

func buildSuratJalanInternalExportFileName(detail *model.SuratJalanInternalDetailResponse) string {
	parts := []string{"SURAT_JALAN_INTERNAL"}
	if detail != nil {
		if noDokumen := sanitizeExportSegment(detail.NoDokumen); noDokumen != "DOCUMENT" {
			parts = append(parts, noDokumen)
		}
		parts = append(parts, fmt.Sprintf("%d", detail.ID))
	}

	return ensureExportExtension(strings.Join(parts, "_"), ".xlsx")
}
