package model

import "permatatex-inventory/pkg/response"

type CreateRatioSizeMarkerRequest struct {
	IDWoShellSize int32 `json:"id_wo_shell_size" binding:"required,gt=0"`
	RatioPlan     int32 `json:"ratio_plan" binding:"gte=0"`
}

type CreateRatioMarkerRequest struct {
	IDWoShell            int32                          `json:"id_wo_shell" binding:"required,gt=0"`
	Cons                 float64                        `json:"cons" binding:"gte=0"`
	PlanSpreadingGelaran float64                        `json:"plan_spreading_gelaran" binding:"gte=0"`
	PanjangMarker        float64                        `json:"panjang_marker" binding:"gte=0"`
	EfficiencyMarker     float64                        `json:"efficiency_marker" binding:"gte=0"`
	Allowance            float64                        `json:"allowance" binding:"gte=0"`
	ConsBuyer            *float64                       `json:"cons_buyer" binding:"omitempty,gte=0"`
	Plot                 int32                          `json:"plot" binding:"required,gte=1"`
	LebarKain            float64                        `json:"lebar_kain" binding:"gte=0"`
	PanjangMarkerUnit    string                         `json:"panjang_marker_unit" binding:"required"`
	Ket                  string                         `json:"ket" binding:"omitempty"`
	Sizes                []CreateRatioSizeMarkerRequest `json:"sizes" binding:"required,min=1,dive"`
}

type CreateKomponenMarkerPlanRequest struct {
	NamaKomponen string                     `json:"nama_komponen" binding:"required"`
	Ratios       []CreateRatioMarkerRequest `json:"ratios" binding:"required,min=1,dive"`
}

type CreateMarkerPlanRequest struct {
	NoDokumen      string                            `json:"no_dokumen" binding:"required"`
	TanggalEfektif string                            `json:"tanggal_efektif" binding:"required,datetime=2006-01-02"`
	IDWoShell      int32                             `json:"id_wo_shell" binding:"required,gt=0"`
	Components     []CreateKomponenMarkerPlanRequest `json:"components" binding:"required,min=1,dive"`
}

type RatioSizeMarkerResponse struct {
	IDRatioSizeMarker int32  `json:"id_ratio_size_marker"`
	IDRatioMarker     int32  `json:"id_ratio_marker"`
	IDWoShellSize     int32  `json:"id_wo_shell_size"`
	RatioPlan         int32  `json:"ratio_plan"`
	Size              string `json:"size,omitempty"`
	SizeQty           int32  `json:"size_qty"`
}

type RatioMarkerResponse struct {
	IDRatioMarker        int32                     `json:"id_ratio_marker"`
	IDKomponenMarker     int32                     `json:"id_komponen_marker"`
	IDWoShell            int32                     `json:"id_wo_shell"`
	Cons                 float64                   `json:"cons"`
	PlanSpreadingGelaran float64                   `json:"plan_spreading_gelaran"`
	PanjangMarker        float64                   `json:"panjang_marker"`
	EfficiencyMarker     float64                   `json:"efficiency_marker"`
	Allowance            float64                   `json:"allowance"`
	ConsBuyer            *float64                  `json:"cons_buyer,omitempty"`
	Plot                 int32                     `json:"plot"`
	LebarKain            float64                   `json:"lebar_kain"`
	PanjangMarkerUnit    string                    `json:"panjang_marker_unit"`
	Ket                  string                    `json:"ket"`
	CreatedAt            string                    `json:"created_at"`
	Sizes                []RatioSizeMarkerResponse `json:"sizes"`
}

type KomponenMarkerPlanResponse struct {
	IDKomponenMarker int32                 `json:"id_komponen_marker"`
	IDMarkerPlan     int32                 `json:"id_marker_plan"`
	NamaKomponen     string                `json:"nama_komponen"`
	CreatedAt        string                `json:"created_at"`
	Ratios           []RatioMarkerResponse `json:"ratios"`
}

type MarkerPlanResponse struct {
	IDMarkerPlan   int32                        `json:"id_marker_plan"`
	NoDokumen      string                       `json:"no_dokumen"`
	TanggalEfektif string                       `json:"tanggal_efektif"`
	IDWoShell      int32                        `json:"id_wo_shell"`
	CreatedAt      string                       `json:"created_at"`
	Components     []KomponenMarkerPlanResponse `json:"components"`
}

// Swagger documentation support types

type MarkerPlanSuccessDoc struct {
	Status  string             `json:"status" example:"success"`
	Message string             `json:"message" example:"marker plan created"`
	Data    MarkerPlanResponse `json:"data"`
}

type MarkerPlanErrorDetail struct {
	Code string `json:"code" example:"related_data_not_found"`
}

type MarkerPlanErrorDoc struct {
	Status  string                `json:"status" example:"error"`
	Message string                `json:"message" example:"related data not found"`
	Error   MarkerPlanErrorDetail `json:"error"`
}

type MarkerPlanValidationErrorDoc struct {
	Status  string                         `json:"status" example:"error"`
	Message string                         `json:"message" example:"bad request"`
	Error   []response.ValidationErrorItem `json:"error"`
}

type MarkerPlanListItem struct {
	IDMarkerPlan   int32  `json:"id_marker_plan"`
	NoDokumen      string `json:"no_dokumen"`
	TanggalEfektif string `json:"tanggal_efektif"`
	IDWoShell      int32  `json:"id_wo_shell"`
	Deskripsi      string `json:"deskripsi"`
	Color          string `json:"color"`
	IDWo           int32  `json:"id_wo"`
	Buyer          string `json:"buyer"`
	Model          string `json:"model"`
	CreatedAt      string `json:"created_at"`
}

type MarkerPlanListResponse struct {
	Items      []MarkerPlanListItem `json:"items"`
	Pagination PaginationMeta       `json:"pagination"`
}

type MarkerPlanListSuccessDoc struct {
	Status  string                 `json:"status" example:"success"`
	Message string                 `json:"message" example:"marker plans retrieved"`
	Data    MarkerPlanListResponse `json:"data"`
}
