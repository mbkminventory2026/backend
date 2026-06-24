package excel

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/xuri/excelize/v2"
)

var (
	ErrTemplateRootRequired = errors.New("excel template root is required")
	ErrTemplateNameRequired = errors.New("excel template name is required")
	ErrTemplateNotFound     = errors.New("excel template not found")
)

type CellMutation struct {
	Sheet string
	Cell  string
	Value any
}

type RowMutation struct {
	Sheet     string
	StartCell string
	Values    []any
}

type WorkbookRequest struct {
	TemplateName  string
	CellMutations []CellMutation
	RowMutations  []RowMutation
}

type Renderer struct {
	templateRoot string
}

func NewRenderer(templateRoot string) (*Renderer, error) {
	root := strings.TrimSpace(templateRoot)
	if root == "" {
		return nil, ErrTemplateRootRequired
	}

	return &Renderer{templateRoot: root}, nil
}

func (r *Renderer) ListTemplates() ([]string, error) {
	templates := make([]string, 0)

	err := filepath.WalkDir(r.templateRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if d.IsDir() {
			return nil
		}

		if !strings.EqualFold(filepath.Ext(d.Name()), ".xlsx") {
			return nil
		}

		relativePath, err := filepath.Rel(r.templateRoot, path)
		if err != nil {
			return err
		}

		templates = append(templates, filepath.ToSlash(relativePath))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk excel templates: %w", err)
	}

	slices.Sort(templates)
	return templates, nil
}

func (r *Renderer) OpenTemplate(templateName string) (*excelize.File, error) {
	templatePath, err := r.resolveTemplatePath(templateName)
	if err != nil {
		return nil, err
	}

	workbook, err := excelize.OpenFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("open excel template: %w", err)
	}

	return workbook, nil
}

func (r *Renderer) Render(request WorkbookRequest) ([]byte, error) {
	workbook, err := r.OpenTemplate(request.TemplateName)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = workbook.Close()
	}()

	for _, mutation := range request.CellMutations {
		if err := workbook.SetCellValue(mutation.Sheet, mutation.Cell, mutation.Value); err != nil {
			return nil, fmt.Errorf(
				"set excel cell value for sheet %s cell %s: %w",
				mutation.Sheet,
				mutation.Cell,
				err,
			)
		}
	}

	for _, mutation := range request.RowMutations {
		values := append([]any(nil), mutation.Values...)
		if err := workbook.SetSheetRow(mutation.Sheet, mutation.StartCell, &values); err != nil {
			return nil, fmt.Errorf(
				"set excel row value for sheet %s start cell %s: %w",
				mutation.Sheet,
				mutation.StartCell,
				err,
			)
		}
	}

	var buffer bytes.Buffer
	if err := workbook.Write(&buffer); err != nil {
		return nil, fmt.Errorf("write excel workbook: %w", err)
	}

	return buffer.Bytes(), nil
}

func (r *Renderer) resolveTemplatePath(templateName string) (string, error) {
	name := strings.TrimSpace(templateName)
	if name == "" {
		return "", ErrTemplateNameRequired
	}

	cleanName := filepath.Clean(name)
	if cleanName == "." || strings.HasPrefix(cleanName, "..") {
		return "", fmt.Errorf("%w: %s", ErrTemplateNotFound, templateName)
	}

	candidatePath := filepath.Join(r.templateRoot, cleanName)
	relativePath, err := filepath.Rel(r.templateRoot, candidatePath)
	if err != nil {
		return "", fmt.Errorf("resolve excel template path: %w", err)
	}

	if strings.HasPrefix(relativePath, "..") {
		return "", fmt.Errorf("%w: %s", ErrTemplateNotFound, templateName)
	}

	fileInfo, err := os.Stat(candidatePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("%w: %s", ErrTemplateNotFound, templateName)
		}
		return "", fmt.Errorf("stat excel template: %w", err)
	}

	if fileInfo.IsDir() {
		return "", fmt.Errorf("%w: %s", ErrTemplateNotFound, templateName)
	}

	if !strings.EqualFold(filepath.Ext(fileInfo.Name()), ".xlsx") {
		return "", fmt.Errorf("%w: %s", ErrTemplateNotFound, templateName)
	}

	return candidatePath, nil
}
