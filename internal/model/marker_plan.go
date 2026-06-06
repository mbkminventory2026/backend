package model

import "permatatex-inventory/pkg/response"

type CreateRatioSizeMarkerRequest struct {
	IDWoShellSize int32 `json:"id_wo_shell_size" binding:"required,gt=0"`
	QtyPlan       int32 `json:"qty_plan" binding:"required,gte=0"`
}

type CreateRatioMarkerRequest struct {
	IDWoShell            int32                          `json:"id_wo_shell" binding:"required,gt=0"`
	Cons                 float64                        `json:"cons" binding:"required,gte=0"`
	PlanSpreadingGelaran float64                        `json:"plan_spreading_gelaran" binding:"required,gte=0"`
	PanjangMarker        float64                        `json:"panjang_marker" binding:"required,gte=0"`
	EfficiencyMarker     float64                        `json:"efficiency_marker" binding:"required,gte=0"`
	Allowance            float64                        `json:"allowance" binding:"required,gte=0"`
	ConsBuyer            *float64                       `json:"cons_buyer" binding:"omitempty,gte=0"`
	RollQty              int32                          `json:"roll_qty" binding:"required,gte=0"`
	SambunganRoll        int32                          `json:"sambungan_roll" binding:"required,gte=0"`
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
	QtyPlan           int32  `json:"qty_plan"`
	Size              string `json:"size,omitempty"`
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
	RollQty              int32                     `json:"roll_qty"`
	SambunganRoll        int32                     `json:"sambungan_roll"`
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
