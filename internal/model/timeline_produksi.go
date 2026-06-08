package model

import "permatatex-inventory/pkg/response"

type CreateWOShellPlanRequest struct {
	IDWoShell              int32   `json:"id_wo_shell" binding:"required,gt=0"`
	InLine                 string  `json:"in_line" binding:"required"`
	TglGelarCutting        *string `json:"tgl_gelar_cutting" binding:"omitempty,datetime=2006-01-02"`
	StatusGelarCutting     string  `json:"status_gelar_cutting" binding:"omitempty"`
	TglEmbroo              *string `json:"tgl_embroo" binding:"omitempty,datetime=2006-01-02"`
	StatusEmbroo           string  `json:"status_embroo" binding:"omitempty"`
	TglLoadingSewing       *string `json:"tgl_loading_sewing" binding:"omitempty,datetime=2006-01-02"`
	StatusLoadingSewing    string  `json:"status_loading_sewing" binding:"omitempty"`
	TglFinishingPacking    *string `json:"tgl_finishing_packing" binding:"omitempty,datetime=2006-01-02"`
	StatusFinishingPacking string  `json:"status_finishing_packing" binding:"omitempty"`
}

type CreateTimelinePlanRequest struct {
	IDPoClient     int32                      `json:"id_po_client" binding:"required,gt=0"`
	TanggalDisusun string                     `json:"tanggal_disusun" binding:"required,datetime=2006-01-02"`
	Notes          string                     `json:"notes"`
	ShellPlans     []CreateWOShellPlanRequest `json:"shell_plans" binding:"required,min=1,dive"`
}

type WOShellPlanResponse struct {
	IDWoShellPlan          int32  `json:"id_wo_shell_plan"`
	IDTimeline             int32  `json:"id_timeline"`
	IDWoShell              int32  `json:"id_wo_shell"`
	InLine                 string `json:"in_line"`
	TglGelarCutting        string `json:"tgl_gelar_cutting,omitempty"`
	StatusGelarCutting     string `json:"status_gelar_cutting"`
	TglEmbroo              string `json:"tgl_embroo,omitempty"`
	StatusEmbroo           string `json:"status_embroo"`
	TglLoadingSewing       string `json:"tgl_loading_sewing,omitempty"`
	StatusLoadingSewing    string `json:"status_loading_sewing"`
	TglFinishingPacking    string `json:"tgl_finishing_packing,omitempty"`
	StatusFinishingPacking string `json:"status_finishing_packing"`
	Fabric                 string `json:"fabric,omitempty"`
	Color                  string `json:"color,omitempty"`
}

type TimelinePlanResponse struct {
	IDTimeline     int32                 `json:"id_timeline"`
	IDPoClient     int32                 `json:"id_po_client"`
	TanggalDisusun string                `json:"tanggal_disusun"`
	Notes          string                `json:"notes"`
	CreatedAt      string                `json:"created_at"`
	ShellPlans     []WOShellPlanResponse `json:"shell_plans"`
}

type UpdateWOShellPlanStatusRequest struct {
	StatusGelarCutting     string `json:"status_gelar_cutting" binding:"omitempty"`
	StatusEmbroo           string `json:"status_embroo" binding:"omitempty"`
	StatusLoadingSewing    string `json:"status_loading_sewing" binding:"omitempty"`
	StatusFinishingPacking string `json:"status_finishing_packing" binding:"omitempty"`
}

// Swagger documentation support types

type TimelinePlanSuccessDoc struct {
	Status  string               `json:"status" example:"success"`
	Message string               `json:"message" example:"timeline plan created"`
	Data    TimelinePlanResponse `json:"data"`
}

type TimelinePlanErrorDetail struct {
	Code string `json:"code" example:"related_data_not_found"`
}

type TimelinePlanErrorDoc struct {
	Status  string                  `json:"status" example:"error"`
	Message string                  `json:"message" example:"related data not found"`
	Error   TimelinePlanErrorDetail `json:"error"`
}

type TimelinePlanValidationErrorDoc struct {
	Status  string                         `json:"status" example:"error"`
	Message string                         `json:"message" example:"bad request"`
	Error   []response.ValidationErrorItem `json:"error"`
}

type UpdateWOShellPlanStatusSuccessDoc struct {
	Status  string `json:"status" example:"success"`
	Message string `json:"message" example:"wo shell plan status updated"`
}

type TimelinePlanListItem struct {
	IDTimeline       int32  `json:"id_timeline"`
	IDPoClient       int32  `json:"id_po_client"`
	ClientName       string `json:"client_name"`
	PoInternalNumber string `json:"po_number"`
	TanggalDisusun   string `json:"tanggal_disusun"`
	Notes            string `json:"notes"`
	CreatedAt        string `json:"created_at"`
}

type TimelinePlanListResponse struct {
	Items      []TimelinePlanListItem `json:"items"`
	Pagination PaginationMeta         `json:"pagination"`
}

type TimelinePlanListSuccessDoc struct {
	Status  string                            `json:"status" example:"success"`
	Message string                            `json:"message" example:"timeline plans retrieved"`
	Data    TimelinePlanListResponse          `json:"data"`
}
