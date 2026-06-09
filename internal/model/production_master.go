package model

import "time"

type ProductionLineResponse struct {
	IDProductionLine int32     `json:"id_production_line"`
	Name             string    `json:"name"`
	CreatedAt        time.Time `json:"created_at"`
}

type ProductionStatusPlanResponse struct {
	IDProductionStatusPlan int32     `json:"id_production_status_plan"`
	Name                   string    `json:"name"`
	CreatedAt              time.Time `json:"created_at"`
}

type MasterDataResponse struct {
	ProductionLines       []ProductionLineResponse       `json:"production_lines"`
	ProductionStatusPlans []ProductionStatusPlanResponse `json:"production_status_plans"`
}

type CreateProductionLineRequest struct {
	Name string `json:"name" binding:"required"`
}

type UpdateProductionLineRequest struct {
	Name string `json:"name" binding:"required"`
}

type CreateProductionStatusPlanRequest struct {
	Name string `json:"name" binding:"required"`
}

type UpdateProductionStatusPlanRequest struct {
	Name string `json:"name" binding:"required"`
}
