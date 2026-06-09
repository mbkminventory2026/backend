package model

import "permatatex-inventory/pkg/response"

type CreateRatioSizeSpreadingRequest struct {
	IDWoShellSize int32 `json:"id_wo_shell_size" binding:"required,gt=0"`
	RatioPlan     int32 `json:"ratio_plan" binding:"gte=0"`
}

type CreateRatioSpreadingRequest struct {
	IDWoShell            int32                             `json:"id_wo_shell" binding:"required,gt=0"`
	Cons                 float64                           `json:"cons" binding:"gte=0"`
	PlanSpreadingGelaran float64                           `json:"plan_spreading_gelaran" binding:"gte=0"`
	Allowance            float64                           `json:"allowance" binding:"gte=0"`
	RollQty              int32                             `json:"roll_qty" binding:"gte=0"`
	SambunganRoll        int32                             `json:"sambungan_roll" binding:"gte=0"`
	Reject               float64                           `json:"reject" binding:"gte=0"`
	LebarKain            float64                           `json:"lebar_kain" binding:"gte=0"`
	Ket                  string                            `json:"ket" binding:"omitempty"`
	Sizes                []CreateRatioSizeSpreadingRequest `json:"sizes" binding:"required,min=1,dive"`
}

type CreateKomponenSpreadingRequest struct {
	NamaKomponen string                        `json:"nama_komponen" binding:"required"`
	Ratios       []CreateRatioSpreadingRequest `json:"ratios" binding:"required,min=1,dive"`
}

type CreateSpreadingCuttingPlanRequest struct {
	NoDokumen      string                           `json:"no_dokumen" binding:"required"`
	TanggalEfektif string                           `json:"tanggal_efektif" binding:"required,datetime=2006-01-02"`
	IDWo           int32                            `json:"id_wo" binding:"required,gt=0"`
	Components     []CreateKomponenSpreadingRequest `json:"components" binding:"required,min=1,dive"`
}

type RatioSizeSpreadingResponse struct {
	IDRatioSizeSpreading int32  `json:"id_ratio_size_spreading"`
	IDRatioSpreading     int32  `json:"id_ratio_spreading"`
	IDWoShellSize        int32  `json:"id_wo_shell_size"`
	RatioPlan            int32  `json:"ratio_plan"`
	Size                 string `json:"size,omitempty"`
	SizeQty              int32  `json:"size_qty"`
}

type RatioSpreadingResponse struct {
	IDRatioSpreading     int32                        `json:"id_ratio_spreading"`
	IDKomponenSpreading  int32                        `json:"id_komponen_spreading"`
	IDWoShell            int32                        `json:"id_wo_shell"`
	Cons                 float64                      `json:"cons"`
	PlanSpreadingGelaran float64                      `json:"plan_spreading_gelaran"`
	Allowance            float64                      `json:"allowance"`
	RollQty              int32                        `json:"roll_qty"`
	SambunganRoll        int32                        `json:"sambungan_roll"`
	Reject               float64                      `json:"reject"`
	LebarKain            float64                      `json:"lebar_kain"`
	Ket                  string                       `json:"ket"`
	CreatedAt            string                       `json:"created_at"`
	Sizes                []RatioSizeSpreadingResponse `json:"sizes"`
}

type KomponenSpreadingResponse struct {
	IDKomponenSpreading    int32                    `json:"id_komponen_spreading"`
	IDSpreadingCuttingPlan int32                    `json:"id_spreading_cutting_plan"`
	NamaKomponen           string                   `json:"nama_komponen"`
	CreatedAt              string                   `json:"created_at"`
	Ratios                 []RatioSpreadingResponse `json:"ratios"`
}

type SpreadingCuttingPlanResponse struct {
	IDSpreadingCuttingPlan int32                       `json:"id_spreading_cutting_plan"`
	NoDokumen              string                      `json:"no_dokumen"`
	TanggalEfektif         string                      `json:"tanggal_efektif"`
	IDWo                   int32                       `json:"id_wo"`
	CreatedAt              string                      `json:"created_at"`
	Components             []KomponenSpreadingResponse `json:"components"`
}

// Swagger documentation support types

type SpreadingCuttingPlanSuccessDoc struct {
	Status  string                       `json:"status" example:"success"`
	Message string                       `json:"message" example:"spreading cutting plan created"`
	Data    SpreadingCuttingPlanResponse `json:"data"`
}

type SpreadingCuttingPlanErrorDetail struct {
	Code string `json:"code" example:"related_data_not_found"`
}

type SpreadingCuttingPlanErrorDoc struct {
	Status  string                          `json:"status" example:"error"`
	Message string                          `json:"message" example:"related data not found"`
	Error   SpreadingCuttingPlanErrorDetail `json:"error"`
}

type SpreadingCuttingPlanValidationErrorDoc struct {
	Status  string                         `json:"status" example:"error"`
	Message string                         `json:"message" example:"bad request"`
	Error   []response.ValidationErrorItem `json:"error"`
}

type SpreadingCuttingPlanListItem struct {
	IDSpreadingCuttingPlan int32  `json:"id_spreading_cutting_plan"`
	NoDokumen              string `json:"no_dokumen"`
	TanggalEfektif         string `json:"tanggal_efektif"`
	IDWo                   int32  `json:"id_wo"`
	Buyer                  string `json:"buyer"`
	Model                  string `json:"model"`
	CreatedAt              string `json:"created_at"`
}

type SpreadingCuttingPlanListResponse struct {
	Items      []SpreadingCuttingPlanListItem `json:"items"`
	Pagination PaginationMeta                 `json:"pagination"`
}

type SpreadingCuttingPlanListSuccessDoc struct {
	Status  string                           `json:"status" example:"success"`
	Message string                           `json:"message" example:"spreading cutting plans retrieved"`
	Data    SpreadingCuttingPlanListResponse `json:"data"`
}
