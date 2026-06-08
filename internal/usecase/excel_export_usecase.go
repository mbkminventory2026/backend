package usecase

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"permatatex-inventory/internal/model"
	excelexporter "permatatex-inventory/pkg/exporter/excel"
)

const excelContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

type ExcelRenderer interface {
	ListTemplates() ([]string, error)
	Render(request excelexporter.WorkbookRequest) ([]byte, error)
}

type ExcelExportUseCase struct {
	renderer ExcelRenderer
}

func NewExcelExportUseCase(renderer ExcelRenderer) (*ExcelExportUseCase, error) {
	if renderer == nil {
		return nil, errors.New("excel renderer is required")
	}

	return &ExcelExportUseCase{renderer: renderer}, nil
}

func (u *ExcelExportUseCase) ListTemplates(ctx context.Context) ([]model.ExcelTemplateInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	templates, err := u.renderer.ListTemplates()
	if err != nil {
		return nil, fmt.Errorf("list excel templates: %w", err)
	}

	result := make([]model.ExcelTemplateInfo, 0, len(templates))
	for _, templateName := range templates {
		result = append(result, model.ExcelTemplateInfo{Name: templateName})
	}

	return result, nil
}

func (u *ExcelExportUseCase) RenderTemplate(ctx context.Context, request model.ExcelRenderRequest) (*model.ExportedFile, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	workbookRequest := excelexporter.WorkbookRequest{
		TemplateName:  request.TemplateName,
		CellMutations: make([]excelexporter.CellMutation, 0, len(request.CellMutations)),
		RowMutations:  make([]excelexporter.RowMutation, 0, len(request.RowMutations)),
	}

	for _, mutation := range request.CellMutations {
		workbookRequest.CellMutations = append(workbookRequest.CellMutations, excelexporter.CellMutation{
			Sheet: mutation.Sheet,
			Cell:  mutation.Cell,
			Value: mutation.Value,
		})
	}

	for _, mutation := range request.RowMutations {
		workbookRequest.RowMutations = append(workbookRequest.RowMutations, excelexporter.RowMutation{
			Sheet:     mutation.Sheet,
			StartCell: mutation.StartCell,
			Values:    mutation.Values,
		})
	}

	content, err := u.renderer.Render(workbookRequest)
	if err != nil {
		return nil, fmt.Errorf("render excel template: %w", err)
	}

	return &model.ExportedFile{
		FileName:    buildOutputFileName(request.OutputFileName, request.TemplateName),
		ContentType: excelContentType,
		Content:     content,
	}, nil
}

func buildOutputFileName(outputFileName, templateName string) string {
	trimmedOutputName := strings.TrimSpace(outputFileName)
	if trimmedOutputName != "" {
		if strings.EqualFold(filepath.Ext(trimmedOutputName), ".xlsx") {
			return trimmedOutputName
		}

		return trimmedOutputName + ".xlsx"
	}

	templateBaseName := filepath.Base(templateName)
	if templateBaseName == "" || templateBaseName == "." {
		return "export.xlsx"
	}

	if strings.EqualFold(filepath.Ext(templateBaseName), ".xlsx") {
		return templateBaseName
	}

	return templateBaseName + ".xlsx"
}
