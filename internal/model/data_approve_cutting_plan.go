package model

import "permatatex-inventory/pkg/response"

// ─── Request ─────────────────────────────────────────────────────────────────

type CreateDataApproveCuttingPlanRequest struct {
	NoDokumen string `json:"no_dokumen" binding:"required"`
	Tanggal   string `json:"tanggal" binding:"required,datetime=2006-01-02"`
	IDWo      int32  `json:"id_wo" binding:"required,gt=0"`
}

// ─── Response ─────────────────────────────────────────────────────────────────

type DataApproveCuttingPlanRow struct {
	Size             string `json:"size"`
	QtyOrder         int32  `json:"qty_order"`
	QtyCuttingPlan   int64  `json:"qty_cutting_plan"`
	QtyCuttingActual int64  `json:"qty_cutting_actual"`
	CuttingReport    int64  `json:"cutting_report"`
	BalanceAllowance int64  `json:"balance_allowance"`
}

type DataApproveCuttingPlanResponse struct {
	IDDacp    int32                       `json:"id_dacp"`
	NoDokumen string                      `json:"no_dokumen"`
	Tanggal   string                      `json:"tanggal"`
	IDWo      int32                       `json:"id_wo"`
	Buyer     string                      `json:"buyer"`
	Model     string                      `json:"model"`
	Style     string                      `json:"style"`
	CreatedAt string                      `json:"created_at"`
	Rows      []DataApproveCuttingPlanRow `json:"rows"`
}

type DataApproveCuttingPlanListItem struct {
	IDDacp    int32  `json:"id_dacp"`
	NoDokumen string `json:"no_dokumen"`
	Tanggal   string `json:"tanggal"`
	IDWo      int32  `json:"id_wo"`
	Buyer     string `json:"buyer"`
	Model     string `json:"model"`
	CreatedAt string `json:"created_at"`
}

type DataApproveCuttingPlanListResponse struct {
	Items      []DataApproveCuttingPlanListItem `json:"items"`
	Pagination PaginationMeta                   `json:"pagination"`
}

// ─── Swagger doc types ────────────────────────────────────────────────────────

type DataApproveCuttingPlanSuccessDoc struct {
	Status  string                         `json:"status" example:"success"`
	Message string                         `json:"message" example:"data approve cutting plan created"`
	Data    DataApproveCuttingPlanResponse `json:"data"`
}

type DataApproveCuttingPlanErrorDetail struct {
	Code string `json:"code" example:"invalid_payload"`
}

type DataApproveCuttingPlanErrorDoc struct {
	Status  string                            `json:"status" example:"error"`
	Message string                            `json:"message" example:"bad request"`
	Error   DataApproveCuttingPlanErrorDetail `json:"error"`
}

type DataApproveCuttingPlanValidationErrorDoc struct {
	Status  string                         `json:"status" example:"error"`
	Message string                         `json:"message" example:"validation error"`
	Error   []response.ValidationErrorItem `json:"error"`
}

type DataApproveCuttingPlanListSuccessDoc struct {
	Status  string                             `json:"status" example:"success"`
	Message string                             `json:"message" example:"data approve cutting plans retrieved"`
	Data    DataApproveCuttingPlanListResponse `json:"data"`
}
