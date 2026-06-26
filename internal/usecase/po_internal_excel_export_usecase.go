package usecase

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"permatatex-inventory/internal/model"
	"permatatex-inventory/pkg/exporter/excel"
	"github.com/xuri/excelize/v2"
)

const (
	poInternalExportTemplateName = "xlsx/template_po_internal.xlsx"
	poInternalItemBaseCapacity   = 6
	poInternalItemStartRow       = 18
	poInternalSummaryStartRow    = 24
)

var indonesianMonths = []string{
	"Januari",
	"Februari",
	"Maret",
	"April",
	"Mei",
	"Juni",
	"Juli",
	"Agustus",
	"September",
	"Oktober",
	"November",
	"Desember",
}

type poInternalExcelLayout struct {
	itemExtraRows   int
	summaryRow      int
	ppnRow          int
	balanceRow      int
	transferRow     int
	regardsRow      int
	companyNameRow  int
	leftSignNameRow int
	leftSignRoleRow int
	centerNoteRow   int
	centerNameRow   int
	centerRoleRow   int
	rightSignRow    int
}

type POInternalExcelExportUseCase struct {
	renderer              *excel.Renderer
	transactionDocumentUC *TransactionDocumentUseCase
	profilPerusahaanUC    *ProfilPerusahaanUseCase
}

func NewPOInternalExcelExportUseCase(
	renderer *excel.Renderer,
	transactionDocumentUC *TransactionDocumentUseCase,
	profilPerusahaanUC *ProfilPerusahaanUseCase,
) (*POInternalExcelExportUseCase, error) {
	if renderer == nil {
		return nil, errors.New("excel renderer is required")
	}
	if transactionDocumentUC == nil {
		return nil, errors.New("transaction document usecase is required")
	}
	if profilPerusahaanUC == nil {
		return nil, errors.New("profil perusahaan usecase is required")
	}

	return &POInternalExcelExportUseCase{
		renderer:              renderer,
		transactionDocumentUC: transactionDocumentUC,
		profilPerusahaanUC:    profilPerusahaanUC,
	}, nil
}

func (u *POInternalExcelExportUseCase) ExportByID(ctx context.Context, id int32) (*model.ExportedFile, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	detail, err := u.transactionDocumentUC.GetPOInternalDetail(ctx, id)
	if err != nil {
		return nil, err
	}

	profile, err := u.profilPerusahaanUC.GetProfilPerusahaan(ctx)
	if err != nil && !errors.Is(err, ErrProfilPerusahaanNotFound) {
		return nil, fmt.Errorf("get profil perusahaan: %w", err)
	}
	if errors.Is(err, ErrProfilPerusahaanNotFound) {
		profile = model.ProfilPerusahaanResponse{}
	}

	workbook, err := u.renderer.OpenTemplate(poInternalExportTemplateName)
	if err != nil {
		return nil, fmt.Errorf("open po internal export template: %w", err)
	}
	defer func() {
		_ = workbook.Close()
	}()

	sheetName := workbook.GetSheetName(0)
	if sheetName == "" {
		return nil, fmt.Errorf("open po internal export template: empty sheet name")
	}

	layout, err := preparePOInternalExportLayout(workbook, sheetName, len(detail.Items))
	if err != nil {
		return nil, fmt.Errorf("prepare po internal export layout: %w", err)
	}

	if err := writePOInternalExportHeader(workbook, sheetName, detail, profile); err != nil {
		return nil, fmt.Errorf("write po internal export header: %w", err)
	}
	if err := writePOInternalExportItems(workbook, sheetName, detail.Items); err != nil {
		return nil, fmt.Errorf("write po internal export items: %w", err)
	}
	if err := writePOInternalExportSummary(workbook, sheetName, layout, detail.Items); err != nil {
		return nil, fmt.Errorf("write po internal export summary: %w", err)
	}
	if err := writePOInternalExportFooter(workbook, sheetName, layout); err != nil {
		return nil, fmt.Errorf("write po internal export footer: %w", err)
	}

	var buffer bytes.Buffer
	if err := workbook.Write(&buffer); err != nil {
		return nil, fmt.Errorf("write po internal export workbook: %w", err)
	}

	return &model.ExportedFile{
		FileName:    buildPOInternalExportFileName(detail),
		ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		Content:     buffer.Bytes(),
	}, nil
}

func preparePOInternalExportLayout(workbook *excelize.File, sheetName string, itemCount int) (poInternalExcelLayout, error) {
	itemExtraRows := max(0, itemCount-poInternalItemBaseCapacity)
	for range itemExtraRows {
		if err := workbook.DuplicateRowTo(sheetName, poInternalItemStartRow, poInternalSummaryStartRow); err != nil {
			return poInternalExcelLayout{}, err
		}
	}

	return poInternalExcelLayout{
		itemExtraRows:   itemExtraRows,
		summaryRow:      poInternalSummaryStartRow + itemExtraRows,
		ppnRow:          poInternalSummaryStartRow + itemExtraRows + 1,
		balanceRow:      poInternalSummaryStartRow + itemExtraRows + 2,
		transferRow:     poInternalSummaryStartRow + itemExtraRows + 4,
		regardsRow:      poInternalSummaryStartRow + itemExtraRows + 7,
		companyNameRow:  poInternalSummaryStartRow + itemExtraRows + 8,
		leftSignNameRow: poInternalSummaryStartRow + itemExtraRows + 15,
		leftSignRoleRow: poInternalSummaryStartRow + itemExtraRows + 16,
		centerNoteRow:   poInternalSummaryStartRow + itemExtraRows + 17,
		centerNameRow:   poInternalSummaryStartRow + itemExtraRows + 24,
		centerRoleRow:   poInternalSummaryStartRow + itemExtraRows + 25,
		rightSignRow:    poInternalSummaryStartRow + itemExtraRows + 15,
	}, nil
}

func writePOInternalExportHeader(
	workbook *excelize.File,
	sheetName string,
	detail *model.POInternalResponse,
	profile model.ProfilPerusahaanResponse,
) error {
	companyContact := profile.Nama
	if strings.TrimSpace(companyContact) == "" {
		companyContact = "Contact"
	}

	values := map[string]any{
		"A5": "Supplier :",
		"A6": strings.TrimSpace(detail.SupplierName),
		"A7": strings.TrimSpace(detail.SupplierAddr),
		"C10": strings.TrimSpace(detail.SupplierContact),
		"E10": strings.TrimSpace(detail.SupplierTelp),
		"C11": strings.TrimSpace(detail.SupplierEmail),
		"E11": strings.TrimSpace(detail.SupplierFax),
		"F6": strings.TrimSpace(profile.Nama),
		"F7": strings.TrimSpace(profile.Alamat),
		"G10": buildPOInternalPrefixedValue(companyContact),
		"J10": strings.TrimSpace(profile.NoTelp),
		"G11": buildPOInternalPrefixedValue(profile.Email),
		"C12": buildPOInternalDocumentNumber(detail),
		"C13": formatPOInternalExportDate(detail.Tanggal),
		"C14": strings.TrimSpace(detail.NamaPO),
		"H12": strings.TrimSpace(detail.Currency),
		"H13": strings.TrimSpace(detail.CPO),
		"H14": strings.TrimSpace(detail.Term),
		"H15": formatPOInternalExportDate(detail.ShipDate),
	}

	for cell, value := range values {
		if err := workbook.SetCellValue(sheetName, cell, value); err != nil {
			return err
		}
	}

	return nil
}

func writePOInternalExportItems(
	workbook *excelize.File,
	sheetName string,
	items []model.POInternalItemResponse,
) error {
	capacity := max(poInternalItemBaseCapacity, len(items))
	for i := 0; i < capacity; i++ {
		rowIndex := poInternalItemStartRow + i
		if i >= len(items) {
			for cell, value := range map[string]any{
				"A": "",
				"B": "",
				"C": "",
				"E": "",
				"F": "",
				"G": "",
				"I": "",
			} {
				if err := workbook.SetCellValue(sheetName, cell+fmt.Sprintf("%d", rowIndex), value); err != nil {
					return err
				}
			}
			continue
		}

		item := items[i]
		assignments := map[string]any{
			"A": i + 1,
			"B": strings.TrimSpace(item.Item),
			"C": strings.TrimSpace(item.Description),
			"E": item.Qty,
			"F": strings.TrimSpace(item.Unit),
			"G": formatPOInternalCurrency(item.UnitPrice),
			"I": formatPOInternalCurrency(float64(item.Qty) * item.UnitPrice),
		}

		for cell, value := range assignments {
			if err := workbook.SetCellValue(sheetName, cell+fmt.Sprintf("%d", rowIndex), value); err != nil {
				return err
			}
		}
	}

	return nil
}

func writePOInternalExportSummary(
	workbook *excelize.File,
	sheetName string,
	layout poInternalExcelLayout,
	items []model.POInternalItemResponse,
) error {
	var totalQty int32
	var subtotal float64
	for _, item := range items {
		totalQty += item.Qty
		subtotal += float64(item.Qty) * item.UnitPrice
	}

	values := map[string]any{
		"E": totalQty,
		"I": formatPOInternalCurrency(subtotal),
	}
	for cell, value := range values {
		if err := workbook.SetCellValue(sheetName, cell+fmt.Sprintf("%d", layout.summaryRow), value); err != nil {
			return err
		}
	}

	if err := workbook.SetCellValue(sheetName, "I"+fmt.Sprintf("%d", layout.ppnRow), "Rp -"); err != nil {
		return err
	}
	if err := workbook.SetCellValue(sheetName, "I"+fmt.Sprintf("%d", layout.balanceRow), formatPOInternalCurrency(subtotal)); err != nil {
		return err
	}
	if err := workbook.SetCellValue(sheetName, "A"+fmt.Sprintf("%d", layout.ppnRow), "Terbilang:"); err != nil {
		return err
	}

	return nil
}

func writePOInternalExportFooter(workbook *excelize.File, sheetName string, layout poInternalExcelLayout) error {
	values := map[string]any{
		"A" + fmt.Sprintf("%d", layout.transferRow):    "Transfer:",
		"B" + fmt.Sprintf("%d", layout.regardsRow):     "Regards",
		"B" + fmt.Sprintf("%d", layout.companyNameRow): "PT PERMATA ANUGRAH KUSUMA",
		"B" + fmt.Sprintf("%d", layout.leftSignNameRow): "Ari Putra Andita",
		"B" + fmt.Sprintf("%d", layout.leftSignRoleRow): "Tim Purchasing",
		"E" + fmt.Sprintf("%d", layout.centerNoteRow):   "Mengetahui,",
		"E" + fmt.Sprintf("%d", layout.centerNameRow):   "Sukaria",
		"E" + fmt.Sprintf("%d", layout.centerRoleRow):   "Operational Director",
		"H" + fmt.Sprintf("%d", layout.regardsRow):      "Konfirmasi Terima Order ",
		"H" + fmt.Sprintf("%d", layout.rightSignRow):    "______________________________",
	}

	for cell, value := range values {
		if err := workbook.SetCellValue(sheetName, cell, value); err != nil {
			return err
		}
	}

	return nil
}

func buildPOInternalPrefixedValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ":"
	}
	return ": " + trimmed
}

func buildPOInternalDocumentNumber(detail *model.POInternalResponse) string {
	if detail == nil {
		return ""
	}

	documentID := fmt.Sprintf("#%d", detail.ID)
	dateValue := parsePOInternalExportDate(detail.Tanggal)
	if dateValue == nil {
		return documentID
	}

	return fmt.Sprintf("POI/%03d/%s/%d", detail.ID, formatRomanMonth(int(dateValue.Month())), dateValue.Year())
}

func formatPOInternalExportDate(raw string) string {
	parsed := parsePOInternalExportDate(raw)
	if parsed == nil {
		return strings.TrimSpace(raw)
	}

	monthIndex := int(parsed.Month()) - 1
	if monthIndex < 0 || monthIndex >= len(indonesianMonths) {
		return parsed.Format("02-01-2006")
	}

	return fmt.Sprintf("%02d %s %d", parsed.Day(), indonesianMonths[monthIndex], parsed.Year())
}

func parsePOInternalExportDate(raw string) *time.Time {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}

	for _, layout := range []string{time.RFC3339, "2006-01-02"} {
		parsed, err := time.Parse(layout, trimmed)
		if err == nil {
			return &parsed
		}
	}

	return nil
}

func formatRomanMonth(month int) string {
	values := []string{"I", "II", "III", "IV", "V", "VI", "VII", "VIII", "IX", "X", "XI", "XII"}
	if month < 1 || month > len(values) {
		return "NA"
	}
	return values[month-1]
}

func formatPOInternalCurrency(value float64) string {
	negative := value < 0
	if negative {
		value = -value
	}

	formatted := fmt.Sprintf("%.2f", value)
	parts := strings.SplitN(formatted, ".", 2)
	integerPart := addThousandsSeparator(parts[0])
	decimalPart := "00"
	if len(parts) == 2 {
		decimalPart = parts[1]
	}

	result := "Rp " + integerPart + "," + decimalPart
	if negative {
		result = "-" + result
	}

	return result
}

func addThousandsSeparator(raw string) string {
	if len(raw) <= 3 {
		return raw
	}

	var parts []string
	for len(raw) > 3 {
		parts = append([]string{raw[len(raw)-3:]}, parts...)
		raw = raw[:len(raw)-3]
	}
	parts = append([]string{raw}, parts...)
	return strings.Join(parts, ".")
}

func buildPOInternalExportFileName(detail *model.POInternalResponse) string {
	parts := []string{"PO_INTERNAL"}
	if detail != nil {
		if sanitized := sanitizePOInternalExportFilePart(detail.NamaPO); sanitized != "" {
			parts = append(parts, sanitized)
		}
		parts = append(parts, fmt.Sprintf("%d", detail.ID))
	}

	fileName := strings.Join(parts, "_")
	return fileName + filepath.Ext(poInternalExportTemplateName)
}

func sanitizePOInternalExportFilePart(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}

	var builder strings.Builder
	lastUnderscore := false
	for _, char := range trimmed {
		switch {
		case char >= 'a' && char <= 'z', char >= 'A' && char <= 'Z', char >= '0' && char <= '9':
			builder.WriteRune(char)
			lastUnderscore = false
		default:
			if lastUnderscore {
				continue
			}
			builder.WriteByte('_')
			lastUnderscore = true
		}
	}

	return strings.Trim(builder.String(), "_")
}

func filterNonEmpty(values ...string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}
	return result
}
