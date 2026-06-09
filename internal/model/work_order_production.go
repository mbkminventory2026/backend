package model

import "permatatex-inventory/pkg/response"

type CreateWorkOrderShellSizeRequest struct {
	Size  string `json:"size" binding:"required"`
	Qty   int32  `json:"qty" binding:"required,gt=0"`
	Ratio int32  `json:"ratio" binding:"required,gte=0"`
}

type CreateWorkOrderShellRequest struct {
	Deskripsi    string                            `json:"deskripsi" binding:"required"`
	Cons         float64                           `json:"cons" binding:"required,gte=0"`
	Color        string                            `json:"color" binding:"required"`
	Allow        int32                             `json:"allow" binding:"required,gte=0"`
	Berat1Yd     float64                           `json:"berat_1_yd" binding:"required,gte=0"`
	ProvidedBy   string                            `json:"provided_by" binding:"required,oneof=client permata permatatex Client Permatatex"`
	MaterialType string                            `json:"material_type" binding:"required,oneof=fabric interlining Fabric Interlining"`
	Sizes        []CreateWorkOrderShellSizeRequest `json:"sizes" binding:"required,min=1,dive"`
}

type CreateWorkOrderTrimRequest struct {
	Item        string  `json:"item" binding:"required"`
	Description string  `json:"description"`
	Color       string  `json:"color"`
	Code        string  `json:"code"`
	Cons        float64 `json:"cons" binding:"required,gte=0"`
	Qty         int32   `json:"qty" binding:"required,gt=0"`
	UOM         string  `json:"uom" binding:"required"`
	Position    string  `json:"position"`
	CreatedBy   string  `json:"created_by"`
	Allow       int32   `json:"allow" binding:"required,gte=0"`
	ProvidedBy  string  `json:"provided_by" binding:"required,oneof=client permata permatatex Client Permatatex"`
}

type CreateMaterialListItemRequest struct {
	Item        string  `json:"item"`
	Description string  `json:"description"`
	Qty         int32   `json:"qty"`
	Unit        string  `json:"unit" binding:"required"`
	EstPrice    float64 `json:"est_price"`
	ShellIndex  *int    `json:"shell_index,omitempty"` // 0-based index into shells array
	TrimIndex   *int    `json:"trim_index,omitempty"`  // 0-based index into trims array
}

type CreateWorkOrderRequest struct {
	Buyer             string                          `json:"buyer" binding:"required"`
	Model             string                          `json:"model" binding:"required"`
	Qty               int32                           `json:"qty" binding:"required,gt=0"`
	FOBCMT            bool                            `json:"fob_cmt"`
	Delivery          string                          `json:"delivery" binding:"required,datetime=2006-01-02"`
	IDPOClientItem    int32                           `json:"id_po_client_item" binding:"required,gt=0"`
	Shells            []CreateWorkOrderShellRequest   `json:"shells" binding:"required,min=1,dive"`
	Trims             []CreateWorkOrderTrimRequest    `json:"trims" binding:"required,min=1,dive"`
	MaterialListItems []CreateMaterialListItemRequest `json:"material_list_items" binding:"omitempty,dive"`
}

type WorkOrderShellSizeResponse struct {
	ID        int32  `json:"id_wo_shell_size"`
	Size      string `json:"size"`
	Qty       int32  `json:"qty"`
	Ratio     int32  `json:"ratio"`
	CreatedAt string `json:"created_at"`
}

type WorkOrderShellResponse struct {
	ID           int32                        `json:"id_wo_shell"`
	Deskripsi    string                       `json:"deskripsi"`
	Cons         float64                      `json:"cons"`
	Color        string                       `json:"color"`
	Allow        int32                        `json:"allow"`
	Berat1Yd     float64                      `json:"berat_1_yd"`
	CreatedAt    string                       `json:"created_at"`
	ProvidedBy   string                       `json:"provided_by"`
	MaterialType string                       `json:"material_type"`
	Sizes        []WorkOrderShellSizeResponse `json:"sizes"`
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
	ProvidedBy  string  `json:"provided_by"`
}

type MaterialListItemResponse struct {
	ID            int32   `json:"id_material_list_item"`
	Item          string  `json:"item"`
	Description   string  `json:"description"`
	Qty           int32   `json:"qty"`
	Unit          string  `json:"unit"`
	EstPrice      float64 `json:"est_price"`
	IDWoShell     *int32  `json:"id_wo_shell,omitempty"`
	IDWoTrim      *int32  `json:"id_wo_trim,omitempty"`
	CreatedAt     string  `json:"created_at"`
	QtySuratJalan int32   `json:"qty_surat_jalan"`
	QtyReceived   int32   `json:"qty_received"`
}

type MaterialListResponse struct {
	ID        int32                      `json:"id_material_list"`
	IDWo      int32                      `json:"id_wo"`
	Name      string                     `json:"name"`
	IsLocked  bool                       `json:"is_locked"`
	CreatedAt string                     `json:"created_at"`
	Items     []MaterialListItemResponse `json:"items"`
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

type WorkOrderShellTotalQtyResponse struct {
	IDWoShell int32 `json:"id_wo_shell"`
	TotalQty  int64 `json:"total_qty"`
}

type WorkOrderShellTotalQtySuccessDoc struct {
	Status  string                         `json:"status" example:"success"`
	Message string                         `json:"message" example:"total qty retrieved"`
	Data    WorkOrderShellTotalQtyResponse `json:"data"`
}

type ReturClientResponse struct {
	IDReturClient int32  `json:"id_retur_client" example:"1"`
	IDWo          int32  `json:"id_wo" example:"10"`
	File          string `json:"file" example:"uploads/wo_10_retur_12345.pdf"`
	Deskripsi     string `json:"deskripsi" example:"Barang reject pada jahitan lengan"`
	CreatedAt     string `json:"created_at" example:"2026-06-06T15:00:00Z"`
}

type ReturClientSuccessDoc struct {
	Status  string              `json:"status" example:"success"`
	Message string              `json:"message" example:"client return submitted"`
	Data    ReturClientResponse `json:"data"`
}

type ReturClientListItem struct {
	IDReturClient int32  `json:"id_retur_client"`
	IDWo          int32  `json:"id_wo"`
	File          string `json:"file"`
	Deskripsi     string `json:"deskripsi"`
	CreatedAt     string `json:"created_at"`
	Buyer         string `json:"buyer"`
	Model         string `json:"model"`
	WoQty         int32  `json:"wo_qty"`
	PoNumber      string `json:"po_number"`
	IDMitra       int32  `json:"id_mitra"`
	MitraName     string `json:"mitra_name"`
	IDPOClient    int32  `json:"id_po_client"`
}

type ReturClientListResponse struct {
	Items      []ReturClientListItem `json:"items"`
	Pagination PaginationMeta        `json:"pagination"`
}

type ReturClientListSuccessDoc struct {
	Status  string                  `json:"status" example:"success"`
	Message string                  `json:"message" example:"retur client list retrieved"`
	Data    ReturClientListResponse `json:"data"`
}
