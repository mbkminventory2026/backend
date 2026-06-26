package usecase

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"permatatex-inventory/internal/model"
	"permatatex-inventory/pkg/exporter/excel"

	"github.com/xuri/excelize/v2"
)

const (
	prInternalExportTemplateName = "xlsx/template_pr.xlsx"
	prInternalItemBaseCapacity   = 8
	prInternalItemStartRow       = 8
	prInternalSummaryStartRow    = 16
	prInternalFooterStartRow     = 17
	prInternalFooterEndRow       = 29
)

type prInternalExcelLayout struct {
	itemExtraRows int
	summaryRow    int
	footerShift   int
}

type PRInternalExcelExportUseCase struct {
	renderer              *excel.Renderer
	transactionDocumentUC *TransactionDocumentUseCase
}

func NewPRInternalExcelExportUseCase(
	renderer *excel.Renderer,
	transactionDocumentUC *TransactionDocumentUseCase,
) (*PRInternalExcelExportUseCase, error) {
	if renderer == nil {
		return nil, errors.New("excel renderer is required")
	}
	if transactionDocumentUC == nil {
		return nil, errors.New("transaction document usecase is required")
	}

	return &PRInternalExcelExportUseCase{
		renderer:              renderer,
		transactionDocumentUC: transactionDocumentUC,
	}, nil
}

func (u *PRInternalExcelExportUseCase) ExportByID(ctx context.Context, id int32) (*model.ExportedFile, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	detail, err := u.transactionDocumentUC.GetPRInternalDetail(ctx, id)
	if err != nil {
		return nil, err
	}

	workbook, err := u.renderer.OpenTemplate(prInternalExportTemplateName)
	if err != nil {
		return nil, fmt.Errorf("open pr internal export template: %w", err)
	}
	defer func() {
		_ = workbook.Close()
	}()

	sheetName := workbook.GetSheetName(0)
	if sheetName == "" {
		return nil, fmt.Errorf("open pr internal export template: empty sheet name")
	}

	layout, err := preparePRInternalExportLayout(workbook, sheetName, len(detail.Items))
	if err != nil {
		return nil, fmt.Errorf("prepare pr internal export layout: %w", err)
	}

	if err := writePRInternalExportHeader(workbook, sheetName, detail); err != nil {
		return nil, fmt.Errorf("write pr internal export header: %w", err)
	}
	if err := writePRInternalExportItems(workbook, sheetName, detail.Items); err != nil {
		return nil, fmt.Errorf("write pr internal export items: %w", err)
	}
	if err := writePRInternalExportSummary(workbook, sheetName, layout, detail.Items); err != nil {
		return nil, fmt.Errorf("write pr internal export summary: %w", err)
	}
	if err := writePRInternalExportFooter(workbook, sheetName, layout, detail); err != nil {
		return nil, fmt.Errorf("write pr internal export footer: %w", err)
	}

	var buffer bytes.Buffer
	if err := workbook.Write(&buffer); err != nil {
		return nil, fmt.Errorf("write pr internal export workbook: %w", err)
	}

	return &model.ExportedFile{
		FileName:    buildPRInternalExportFileName(detail),
		ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		Content:     buffer.Bytes(),
	}, nil
}

func preparePRInternalExportLayout(workbook *excelize.File, sheetName string, itemCount int) (prInternalExcelLayout, error) {
	itemExtraRows := max(0, itemCount-prInternalItemBaseCapacity)
	for range itemExtraRows {
		if err := workbook.DuplicateRowTo(sheetName, prInternalItemStartRow, prInternalSummaryStartRow); err != nil {
			return prInternalExcelLayout{}, err
		}
	}

	return prInternalExcelLayout{
		itemExtraRows: itemExtraRows,
		summaryRow:    prInternalSummaryStartRow + itemExtraRows,
		footerShift:   itemExtraRows,
	}, nil
}

func writePRInternalExportHeader(workbook *excelize.File, sheetName string, detail *model.PRInternalResponse) error {
	values := map[string]any{
		"C3": formatPOInternalExportDate(detail.Tanggal),
		"C4": "",
		"C5": strings.TrimSpace(detail.Departemen),
		"C6": strings.TrimSpace(detail.Projek),
		"H3": strings.TrimSpace(detail.VendorName),
		"H4": strings.TrimSpace(detail.VendorAddress),
		"H5": strings.TrimSpace(detail.VendorTelp),
	}

	for cell, value := range values {
		if err := workbook.SetCellValue(sheetName, cell, value); err != nil {
			return err
		}
	}

	return nil
}

func writePRInternalExportItems(
	workbook *excelize.File,
	sheetName string,
	items []model.PRInternalItemResponse,
) error {
	capacity := max(prInternalItemBaseCapacity, len(items))
	for i := 0; i < capacity; i++ {
		rowIndex := prInternalItemStartRow + i
		assignments := map[string]any{
			"A": "",
			"C": "",
			"D": "",
			"E": "",
			"I": "",
			"K": "",
		}

		if i < len(items) {
			item := items[i]
			assignments = map[string]any{
				"A": i + 1,
				"C": item.Qty,
				"D": strings.TrimSpace(item.Unit),
				"E": buildPRInternalItemDescription(item),
				"I": formatPOInternalCurrency(item.EstPrice),
				"K": formatPOInternalCurrency(float64(item.Qty) * item.EstPrice),
			}
		}

		for cell, value := range assignments {
			if err := workbook.SetCellValue(sheetName, cell+fmt.Sprintf("%d", rowIndex), value); err != nil {
				return err
			}
		}
	}

	return nil
}

func writePRInternalExportSummary(
	workbook *excelize.File,
	sheetName string,
	layout prInternalExcelLayout,
	items []model.PRInternalItemResponse,
) error {
	var grandTotal float64
	for _, item := range items {
		grandTotal += float64(item.Qty) * item.EstPrice
	}

	return workbook.SetCellValue(sheetName, "K"+fmt.Sprintf("%d", layout.summaryRow), formatPOInternalCurrency(grandTotal))
}

func writePRInternalExportFooter(
	workbook *excelize.File,
	sheetName string,
	layout prInternalExcelLayout,
	detail *model.PRInternalResponse,
) error {
	if layout.footerShift == 0 {
		return setPRInternalSignatureDates(workbook, sheetName, detail.Tanggal, 29)
	}

	for row := prInternalFooterEndRow; row >= prInternalFooterStartRow; row-- {
		if err := workbook.DuplicateRowTo(sheetName, row, row+layout.footerShift); err != nil {
			return err
		}
	}

	return setPRInternalSignatureDates(workbook, sheetName, detail.Tanggal, 29+layout.footerShift)
}

func setPRInternalSignatureDates(workbook *excelize.File, sheetName, rawDate string, row int) error {
	dateLabel := "Tanggal : " + formatPOInternalExportDate(rawDate)
	for _, cell := range []string{"A", "C", "F", "I"} {
		if err := workbook.SetCellValue(sheetName, cell+fmt.Sprintf("%d", row), dateLabel); err != nil {
			return err
		}
	}
	return nil
}

func buildPRInternalItemDescription(item model.PRInternalItemResponse) string {
	parts := []string{strings.TrimSpace(item.Item)}
	if description := strings.TrimSpace(item.Description); description != "" {
		parts = append(parts, description)
	}
	return strings.Join(parts, " - ")
}

func buildPRInternalExportFileName(detail *model.PRInternalResponse) string {
	baseName := fmt.Sprintf("PR_INTERNAL_%s_%d", sanitizeExportSegment(detail.Nama), detail.ID)
	return ensureExportExtension(baseName, ".xlsx")
}

func sanitizeExportSegment(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "DOCUMENT"
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
	sanitized := replacer.Replace(trimmed)
	sanitized = strings.Trim(sanitized, "._")
	if sanitized == "" {
		return "DOCUMENT"
	}
	return sanitized
}

func ensureExportExtension(fileName, extension string) string {
	if strings.EqualFold(filepath.Ext(fileName), extension) {
		return fileName
	}
	return fileName + extension
}
