package model

type ExcelCellMutation struct {
	Sheet string `json:"sheet" binding:"required"`
	Cell  string `json:"cell" binding:"required"`
	Value any    `json:"value"`
}

type ExcelRowMutation struct {
	Sheet     string `json:"sheet" binding:"required"`
	StartCell string `json:"start_cell" binding:"required"`
	Values    []any  `json:"values" binding:"required"`
}

type ExcelRenderRequest struct {
	TemplateName   string              `json:"template_name" binding:"required"`
	OutputFileName string              `json:"output_file_name"`
	CellMutations  []ExcelCellMutation `json:"cell_mutations"`
	RowMutations   []ExcelRowMutation  `json:"row_mutations"`
}

type ExportedFile struct {
	FileName    string
	ContentType string
	Content     []byte
}

type ExcelTemplateInfo struct {
	Name string `json:"name"`
}
