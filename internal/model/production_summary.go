package model

type ProductionSummaryFilter struct {
	IDWO          int32
	IDWOShellSize int32
	IDMitra       *int32
	ListQueryFilter
}

type ProductionStats struct {
	Cutting int32 `json:"cutting"`
	Sewing  int32 `json:"sewing"`
	QCPass  int32 `json:"qc_pass"`
	Packing int32 `json:"packing"`
	Shipped int32 `json:"shipped"`
}

type ProductionAggregateResponse struct {
	IDWOShellSize int32           `json:"id_wo_shell_size"`
	IDSize        *int32          `json:"id_size,omitempty"`
	ModelName     string          `json:"model_name"`
	Size          string          `json:"size"`
	TargetQty     int32           `json:"target_qty"`
	Production    ProductionStats `json:"production"`
	LastUpdated   string          `json:"last_updated"`
	Status        string          `json:"status"`
}

type ProductionSummaryListResponse struct {
	Items      []ProductionAggregateResponse `json:"items"`
	Pagination PaginationMeta                `json:"pagination"`
}

type ProductionSummaryListSuccessDoc struct {
	Status  string                        `json:"status" example:"success"`
	Message string                        `json:"message" example:"production summary retrieved"`
	Data    ProductionSummaryListResponse `json:"data"`
}

type DailyReportListItem struct {
	Division      string `json:"division"`
	Tanggal       string `json:"tanggal"`
	Qty           int32  `json:"qty"`
	IDWOShellSize int32  `json:"id_wo_shell_size"`
}

type DailyReportListResponse struct {
	Items []DailyReportListItem `json:"items"`
}

type DailyReportListSuccessDoc struct {
	Status  string                  `json:"status" example:"success"`
	Message string                  `json:"message" example:"daily reports retrieved"`
	Data    DailyReportListResponse `json:"data"`
}
