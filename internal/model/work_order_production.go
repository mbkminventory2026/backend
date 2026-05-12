package model

import "permatatex-inventory/pkg/response"

type CreateWorkOrderShellSizeRequest struct {
	Size  string `json:"size" binding:"required"`
	Qty   int32  `json:"qty" binding:"required,gt=0"`
	Ratio int32  `json:"ratio" binding:"required,gte=0"`
}

type CreateWorkOrderShellRequest struct {
	Fabric   string                            `json:"fabric" binding:"required"`
	Cons     float64                           `json:"cons" binding:"required,gte=0"`
	Color    string                            `json:"color" binding:"required"`
	Allow    int32                             `json:"allow" binding:"required,gte=0"`
	Berat1Yd float64                           `json:"berat_1_yd" binding:"required,gte=0"`
	Sizes    []CreateWorkOrderShellSizeRequest `json:"sizes" binding:"required,min=1,dive"`
}

type CreateWorkOrderTrimRequest struct {
	Item        string  `json:"item" binding:"required"`
	Description string  `json:"description"`
	Color       string  `json:"color" binding:"required"`
	Code        string  `json:"code" binding:"required"`
	Cons        float64 `json:"cons" binding:"required,gte=0"`
	Qty         int32   `json:"qty" binding:"required,gt=0"`
	UOM         string  `json:"uom" binding:"required"`
	Position    string  `json:"position"`
	CreatedBy   string  `json:"created_by"`
	Allow       int32   `json:"allow" binding:"required,gte=0"`
}

type CreateMaterialListRequest struct {
	Description string `json:"description"`
	Size        string `json:"size" binding:"required"`
	Color       string `json:"color" binding:"required"`
	UOM         string `json:"uom" binding:"required"`
}

type CreateWorkOrderRequest struct {
	Buyer          string                        `json:"buyer" binding:"required"`
	Model          string                        `json:"model" binding:"required"`
	Qty            int32                         `json:"qty" binding:"required,gt=0"`
	FOBCMT         bool                          `json:"fob_cmt"`
	Delivery       string                        `json:"delivery" binding:"required,datetime=2006-01-02"`
	IDPOClientItem int32                         `json:"id_po_client_item" binding:"required,gt=0"`
	Shells         []CreateWorkOrderShellRequest `json:"shells" binding:"required,min=1,dive"`
	Trims          []CreateWorkOrderTrimRequest  `json:"trims" binding:"required,min=1,dive"`
	MaterialLists  []CreateMaterialListRequest   `json:"material_lists" binding:"omitempty,dive"`
}

type WorkOrderShellSizeResponse struct {
	ID        int32  `json:"id_wo_shell_size"`
	Size      string `json:"size"`
	Qty       int32  `json:"qty"`
	Ratio     int32  `json:"ratio"`
	CreatedAt string `json:"created_at"`
}

type WorkOrderShellResponse struct {
	ID        int32                        `json:"id_wo_shell"`
	Fabric    string                       `json:"fabric"`
	Cons      float64                      `json:"cons"`
	Color     string                       `json:"color"`
	Allow     int32                        `json:"allow"`
	Berat1Yd  float64                      `json:"berat_1_yd"`
	CreatedAt string                       `json:"created_at"`
	Sizes     []WorkOrderShellSizeResponse `json:"sizes"`
}

type WorkOrderTrimResponse struct {
	ID          int32   `json:"id_wo_trim"`
	Item        string  `json:"item"`
	Description string  `json:"description"`
	Color       string  `json:"color"`
	Code        string  `json:"code"`
	Cons        float64 `json:"cons"`
	Qty         int32   `json:"qty"`
	UOM         string  `json:"uom"`
	Position    string  `json:"position"`
	CreatedBy   string  `json:"created_by"`
	Allow       int32   `json:"allow"`
	CreatedAt   string  `json:"created_at"`
}

type MaterialListResponse struct {
	ID          int32  `json:"id_material_list"`
	Description string `json:"description"`
	Size        string `json:"size"`
	Color       string `json:"color"`
	UOM         string `json:"uom"`
	CreatedAt   string `json:"created_at"`
}

type WorkOrderResponse struct {
	ID             int32                    `json:"id_wo"`
	Buyer          string                   `json:"buyer"`
	Model          string                   `json:"model"`
	Qty            int32                    `json:"qty"`
	FOBCMT         bool                     `json:"fob_cmt"`
	Delivery       string                   `json:"delivery"`
	IDPOClientItem int32                    `json:"id_po_client_item"`
	Status         string                   `json:"status"`
	ClosedByUserID *int32                   `json:"closed_by_user_id,omitempty"`
	ClosedAt       string                   `json:"closed_at,omitempty"`
	CreatedAt      string                   `json:"created_at"`
	Shells         []WorkOrderShellResponse `json:"shells"`
	Trims          []WorkOrderTrimResponse  `json:"trims"`
	MaterialLists  []MaterialListResponse   `json:"material_lists"`
}

type CreateFactoryReportRequest struct {
	IDWOShellSize int32  `json:"id_wo_shell_size" binding:"required,gt=0"`
	Tanggal       string `json:"tanggal" binding:"required,datetime=2006-01-02"`
	Qty           int32  `json:"qty" binding:"required,gt=0"`
}

type FactoryReportResponse struct {
	Division      string `json:"division"`
	ReportID      int32  `json:"report_id"`
	Tanggal       string `json:"tanggal"`
	Qty           int32  `json:"qty"`
	IDWOShellSize int32  `json:"id_wo_shell_size"`
	CreatedAt     string `json:"created_at"`
}

type WorkOrderStatusResponse struct {
	ID             int32  `json:"id_wo"`
	Status         string `json:"status"`
	ClosedByUserID *int32 `json:"closed_by_user_id,omitempty"`
	ClosedAt       string `json:"closed_at,omitempty"`
}

type WorkOrderSuccessDoc struct {
	Status  string            `json:"status" example:"success"`
	Message string            `json:"message" example:"work order created"`
	Data    WorkOrderResponse `json:"data"`
}

type FactoryReportSuccessDoc struct {
	Status  string                `json:"status" example:"success"`
	Message string                `json:"message" example:"factory report created"`
	Data    FactoryReportResponse `json:"data"`
}

type WorkOrderStatusSuccessDoc struct {
	Status  string                  `json:"status" example:"success"`
	Message string                  `json:"message" example:"work order closed"`
	Data    WorkOrderStatusResponse `json:"data"`
}

type WorkOrderValidationErrorDoc struct {
	Status  string                         `json:"status" example:"error"`
	Message string                         `json:"message" example:"bad request"`
	Error   []response.ValidationErrorItem `json:"error"`
}

type WorkOrderErrorDetail struct {
	Code string `json:"code" example:"related_data_not_found"`
}

type WorkOrderErrorDoc struct {
	Status  string               `json:"status" example:"error"`
	Message string               `json:"message" example:"related data not found"`
	Error   WorkOrderErrorDetail `json:"error"`
}
