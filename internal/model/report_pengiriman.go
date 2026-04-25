package model

type CreateReportPengirimanRequest struct {
	Date          string `json:"date" binding:"required"`
	Quantity      int32  `json:"quantity" binding:"required,gt=0"`
	IDWOShellSize int32  `json:"id_wo_shell_size" binding:"required,gt=0"`
}

type ListReportPengirimanFilter struct {
	DateFrom      string
	DateTo        string
	IDWOShellSize int32
	Limit         int32
	Offset        int32
}

type ReportPengirimanResponse struct {
	IDReportPengiriman int32  `json:"id_report_pengiriman"`
	Date               string `json:"date"`
	Quantity           int32  `json:"quantity"`
	IDWOShellSize      int32  `json:"id_wo_shell_size"`
	CreatedAt          string `json:"created_at"`
}

type DeleteReportPengirimanResponse struct {
	IDReportPengiriman int32 `json:"id_report_pengiriman"`
}

type ReportPengirimanSuccessDoc struct {
	Status  string                   `json:"status" example:"success"`
	Message string                   `json:"message" example:"report pengiriman created"`
	Data    ReportPengirimanResponse `json:"data"`
}

type ReportPengirimanListSuccessDoc struct {
	Status  string                     `json:"status" example:"success"`
	Message string                     `json:"message" example:"report pengiriman list retrieved"`
	Data    []ReportPengirimanResponse `json:"data"`
}

type ReportPengirimanDeleteSuccessDoc struct {
	Status  string                         `json:"status" example:"success"`
	Message string                         `json:"message" example:"report pengiriman deleted"`
	Data    DeleteReportPengirimanResponse `json:"data"`
}

type ReportPengirimanBadRequestDoc struct {
	Status  string `json:"status" example:"error"`
	Message string `json:"message" example:"bad request"`
	Error   any    `json:"error"`
}

type ReportPengirimanUnauthorizedDoc struct {
	Status  string `json:"status" example:"error"`
	Message string `json:"message" example:"unauthorized"`
	Error   any    `json:"error,omitempty"`
}

type ReportPengirimanNotFoundDoc struct {
	Status  string `json:"status" example:"error"`
	Message string `json:"message" example:"report pengiriman not found"`
	Error   any    `json:"error,omitempty"`
}

type ReportPengirimanServiceUnavailableDoc struct {
	Status  string `json:"status" example:"error"`
	Message string `json:"message" example:"report pengiriman service unavailable"`
	Error   any    `json:"error,omitempty"`
}
