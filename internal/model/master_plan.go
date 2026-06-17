package model

import "permatatex-inventory/pkg/response"

// Request types

type CreateMasterPlanRequest struct {
	IDDepartemen     int32                    `json:"id_departemen" binding:"required,gt=0"`
	IDProductionLine int32                    `json:"id_production_line" binding:"required,gt=0"`
	Nama             string                   `json:"nama" binding:"omitempty"`
	Items            []AddMasterPlanItemEntry `json:"items" binding:"omitempty,dive"`
}

type AddMasterPlanItemEntry struct {
	IDWoShell int32 `json:"id_wo_shell" binding:"required,gt=0"`
	NoUrut    int32 `json:"no_urut" binding:"gte=0"`
}

type AddMasterPlanItemRequest struct {
	IDWoShell int32 `json:"id_wo_shell" binding:"required,gt=0"`
	NoUrut    int32 `json:"no_urut" binding:"gte=0"`
}

type UpdateMasterPlanRequest struct {
	Nama string `json:"nama"`
}

type TargetHarianEntry struct {
	Tanggal string `json:"tanggal" binding:"required,datetime=2006-01-02"`
	Target  int32  `json:"target" binding:"gte=0"`
}

type UpsertTargetHarianRequest struct {
	Entries []TargetHarianEntry `json:"entries" binding:"required,min=1,dive"`
}

type OutputHarianEntry struct {
	Tanggal string `json:"tanggal" binding:"required,datetime=2006-01-02"`
	Output  int32  `json:"output" binding:"gte=0"`
}

type UpsertOutputHarianRequest struct {
	Entries []OutputHarianEntry `json:"entries" binding:"required,min=1,dive"`
}

type UpsertTargetProsesRequest struct {
	Tanggal    string `json:"tanggal" binding:"required,datetime=2006-01-02"`
	NamaProses string `json:"nama_proses" binding:"required"`
}

// Response types

type TargetHarianResponse struct {
	Tanggal string `json:"tanggal"`
	Target  int32  `json:"target"`
}

type OutputHarianResponse struct {
	Tanggal string `json:"tanggal"`
	Output  int32  `json:"output"`
}

type TargetProsesResponse struct {
	Tanggal    string `json:"tanggal"`
	NamaProses string `json:"nama_proses"`
}

type MasterPlanItemResponse struct {
	IDMasterPlanItem int32                  `json:"id_master_plan_item"`
	IDMasterPlan     int32                  `json:"id_master_plan"`
	IDWoShell        int32                  `json:"id_wo_shell"`
	IDWo             int32                  `json:"id_wo"`
	NoUrut           int32                  `json:"no_urut"`
	Buyer            string                 `json:"buyer"`
	Style            string                 `json:"style"`
	Qty              int32                  `json:"qty"`
	Color            string                 `json:"color"`
	Deskripsi        string                 `json:"deskripsi"`
	CreatedAt        string                 `json:"created_at"`
	TargetHarian     []TargetHarianResponse `json:"target_harian"`
	OutputHarian     []OutputHarianResponse `json:"output_harian"`
	TargetProses     []TargetProsesResponse `json:"target_proses"`
}

type MasterPlanResponse struct {
	IDMasterPlan     int32                    `json:"id_master_plan"`
	IDDepartemen     int32                    `json:"id_departemen"`
	NamaDepartemen   string                   `json:"nama_departemen"`
	IDProductionLine int32                    `json:"id_production_line"`
	NamaLine         string                   `json:"nama_line"`
	Nama             string                   `json:"nama"`
	CreatedAt        string                   `json:"created_at"`
	Items            []MasterPlanItemResponse `json:"items"`
}

type MasterPlanListItem struct {
	IDMasterPlan     int32  `json:"id_master_plan"`
	IDDepartemen     int32  `json:"id_departemen"`
	NamaDepartemen   string `json:"nama_departemen"`
	IDProductionLine int32  `json:"id_production_line"`
	NamaLine         string `json:"nama_line"`
	Nama             string `json:"nama"`
	CreatedAt        string `json:"created_at"`
}

type MasterPlanListResponse struct {
	Items      []MasterPlanListItem `json:"items"`
	Pagination PaginationMeta       `json:"pagination"`
}

// Swagger doc helpers

type MasterPlanSuccessDoc struct {
	Status  string             `json:"status" example:"success"`
	Message string             `json:"message" example:"master plan created"`
	Data    MasterPlanResponse `json:"data"`
}

type MasterPlanListSuccessDoc struct {
	Status  string                 `json:"status" example:"success"`
	Message string                 `json:"message" example:"master plans retrieved"`
	Data    MasterPlanListResponse `json:"data"`
}

type MasterPlanErrorDetail struct {
	Code string `json:"code" example:"master_plan_not_found"`
}

type MasterPlanErrorDoc struct {
	Status  string                `json:"status" example:"error"`
	Message string                `json:"message" example:"master plan not found"`
	Error   MasterPlanErrorDetail `json:"error"`
}

type MasterPlanValidationErrorDoc struct {
	Status  string                         `json:"status" example:"error"`
	Message string                         `json:"message" example:"bad request"`
	Error   []response.ValidationErrorItem `json:"error"`
}
