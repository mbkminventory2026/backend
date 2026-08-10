package usecase

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/xuri/excelize/v2"

	"permatatex-inventory/internal/entity"
	"permatatex-inventory/internal/model"
)

const materialListImportContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

// MaterialListExcelImportTemplateUseCase creates the machine-input workbook.
// Reference values are deliberately informational; Import remains authoritative.
type MaterialListExcelImportTemplateUseCase struct{ repo entity.Querier }

func NewMaterialListExcelImportTemplateUseCase(repo entity.Querier) (*MaterialListExcelImportTemplateUseCase, error) {
	if repo == nil {
		return nil, errors.New("material list import template repository is required")
	}
	return &MaterialListExcelImportTemplateUseCase{repo: repo}, nil
}

func (u *MaterialListExcelImportTemplateUseCase) Generate(ctx context.Context, id int32) (*model.ExportedFile, error) {
	ml, err := u.repo.GetMaterialList(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, importDomainError("material_list_not_found", "target material list was not found")
		}
		return nil, fmt.Errorf("%w: load target material list", ErrMaterialListUnavailable)
	}
	if ml.IsLocked {
		return nil, importDomainError("material_list_locked", "target material list is locked")
	}
	shells, err := u.repo.ListWorkOrderShellsByWorkOrderID(ctx, ml.IDWo)
	if err != nil {
		return nil, fmt.Errorf("%w: load work order shells", ErrMaterialListUnavailable)
	}
	trims, err := u.repo.ListWorkOrderTrimsByWorkOrderID(ctx, ml.IDWo)
	if err != nil {
		return nil, fmt.Errorf("%w: load work order trims", ErrMaterialListUnavailable)
	}
	sizes, err := u.repo.ListWorkOrderShellSizesByWorkOrderID(ctx, ml.IDWo)
	if err != nil {
		return nil, fmt.Errorf("%w: load work order sizes", ErrMaterialListUnavailable)
	}
	book := excelize.NewFile()
	defer book.Close()
	importSheet := book.GetSheetName(0)
	if err := book.SetSheetName(importSheet, "IMPORT"); err != nil {
		return nil, err
	}
	if _, err := book.NewSheet("REFERENSI"); err != nil {
		return nil, err
	}
	if _, err := book.NewSheet("_META"); err != nil {
		return nil, err
	}
	book.SetActiveSheet(0)
	if err := book.SetSheetVisible("_META", false); err != nil {
		return nil, err
	}
	if err := writeMaterialListImportTemplate(book, id, shells, trims, sizes); err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := book.Write(&out); err != nil {
		return nil, fmt.Errorf("write material list import template: %w", err)
	}
	fileName := buildMaterialListImportTemplateFileName(ml.Name)
	return &model.ExportedFile{FileName: fileName, ContentType: materialListImportContentType, Content: out.Bytes()}, nil
}

func buildMaterialListImportTemplateFileName(name string) string {
	segment := materialListExportFileNameSegment(name)
	if segment == "" {
		return "MATERIAL_LIST_IMPORT.xlsx"
	}
	return "MATERIAL_LIST_IMPORT_" + segment + ".xlsx"
}

func writeMaterialListImportTemplate(book *excelize.File, id int32, shells []entity.WorkOrderShell, trims []entity.WorkOrderTrim, sizes []entity.ListWorkOrderShellSizesByWorkOrderIDRow) error {
	headStyle, err := book.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}, Fill: excelize.Fill{Type: "pattern", Color: []string{"D9EAD3"}, Pattern: 1}})
	if err != nil {
		return err
	}
	for i, header := range materialListImportHeaders {
		if err := book.SetCellValue("IMPORT", materialListCell(i+1, 1), header); err != nil {
			return err
		}
	}
	if err := book.SetCellStyle("IMPORT", "A1", "L1", headStyle); err != nil {
		return err
	}
	if err := book.SetPanes("IMPORT", &excelize.Panes{Freeze: true, Split: true, XSplit: 0, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"}); err != nil {
		return err
	}
	for col, width := range map[string]float64{"A": 14, "B": 24, "C": 32, "D": 10, "E": 12, "F": 14, "G": 14, "H": 14, "I": 12, "J": 16, "K": 22, "L": 22} {
		if err := book.SetColWidth("IMPORT", col, col, width); err != nil {
			return err
		}
	}
	if err := book.SetColStyle("IMPORT", "F", mustNumberStyle(book, "0.00")); err != nil {
		return err
	}
	if err := book.SetColStyle("IMPORT", "G", mustNumberStyle(book, "0.000")); err != nil {
		return err
	}
	if err := book.SetCellValue("_META", "A1", materialListImportMarker); err != nil {
		return err
	}
	if err := book.SetCellValue("_META", "B1", id); err != nil {
		return err
	}
	guidance := []string{"REFERENSI (informational; backend validates all IDs)", "SOURCE_TYPE = SHELL -> SOURCE_ID gunakan ID_SHELL", "SOURCE_TYPE = TRIM -> SOURCE_ID gunakan ID_TRIM", "QTY_WO_SCOPE = SIZE -> APPLICABILITY_SIZE_ID gunakan ID_SIZE", "QTY_WO_SCOPE = COLOR -> APPLICABILITY_SHELL_ID gunakan ID_SHELL", "QTY_WO_SCOPE = COLOR_SIZE -> isi kedua ID sesuai pasangan Shell + Size pada referensi"}
	for r, text := range guidance {
		if err := book.SetCellValue("REFERENSI", materialListCell(1, r+1), text); err != nil {
			return err
		}
	}
	r := len(guidance) + 2
	for i, h := range []string{"SHELL", "ID_SHELL", "COLOR", "DESCRIPTION"} {
		if err := book.SetCellValue("REFERENSI", materialListCell(i+1, r), h); err != nil {
			return err
		}
	}
	for _, s := range shells {
		r++
		_ = book.SetCellValue("REFERENSI", materialListCell(2, r), s.IDWoShell)
		_ = book.SetCellValue("REFERENSI", materialListCell(3, r), s.Color)
		_ = book.SetCellValue("REFERENSI", materialListCell(4, r), s.Deskripsi)
	}
	r += 2
	for i, h := range []string{"TRIM", "ID_TRIM", "ITEM", "COLOR_OR_TYPE", "DESCRIPTION"} {
		if err := book.SetCellValue("REFERENSI", materialListCell(i+1, r), h); err != nil {
			return err
		}
	}
	for _, t := range trims {
		r++
		_ = book.SetCellValue("REFERENSI", materialListCell(2, r), t.IDWoTrim)
		_ = book.SetCellValue("REFERENSI", materialListCell(3, r), t.Item)
		_ = book.SetCellValue("REFERENSI", materialListCell(4, r), t.Color)
		_ = book.SetCellValue("REFERENSI", materialListCell(5, r), t.Description)
	}
	r += 2
	for i, h := range []string{"SHELL_SIZE", "ID_SHELL", "SHELL_COLOR", "ID_SIZE", "SIZE"} {
		if err := book.SetCellValue("REFERENSI", materialListCell(i+1, r), h); err != nil {
			return err
		}
	}
	colors := map[int32]string{}
	for _, s := range shells {
		colors[s.IDWoShell] = s.Color
	}
	for _, z := range sizes {
		r++
		_ = book.SetCellValue("REFERENSI", materialListCell(2, r), z.IDWoShell)
		_ = book.SetCellValue("REFERENSI", materialListCell(3, r), colors[z.IDWoShell])
		_ = book.SetCellValue("REFERENSI", materialListCell(4, r), z.IDSize)
		_ = book.SetCellValue("REFERENSI", materialListCell(5, r), z.Size)
	}
	if err := book.SetCellStyle("REFERENSI", "A1", "F1", headStyle); err != nil {
		return err
	}
	for _, col := range []string{"A", "B", "C", "D", "E", "F"} {
		if err := book.SetColWidth("REFERENSI", col, col, 24); err != nil {
			return err
		}
	}
	return nil
}

func mustNumberStyle(book *excelize.File, format string) int {
	style, _ := book.NewStyle(&excelize.Style{NumFmt: 2})
	return style
}
