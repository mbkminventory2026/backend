package model

import "permatatex-inventory/pkg/response"

type ReceiveInventoryRequest struct {
	Tanggal                string `json:"tanggal" binding:"required,datetime=2006-01-02"`
	Qty                    int32  `json:"qty" binding:"required,gt=0"`
	Keterangan             string `json:"keterangan"`
	IDMaterialList         int32  `json:"id_material_list" binding:"required,gt=0"`
	IDRekonsiliasiMaterial int32  `json:"id_rekonsiliasi_material" binding:"required,gt=0"`
}

type IssueInventoryRequest struct {
	Qty                    int32 `json:"qty" binding:"required,gt=0"`
	IDRekonsiliasiMaterial int32 `json:"id_rekonsiliasi_material" binding:"required,gt=0"`
}

type ReceiveInventoryResponse struct {
	IDReceived                   int32  `json:"id_received"`
	Tanggal                      string `json:"tanggal"`
	Qty                          int32  `json:"qty"`
	Keterangan                   string `json:"keterangan"`
	IDMaterialList               int32  `json:"id_material_list"`
	IDRekonsiliasiMaterialTerima int32  `json:"id_rekonsiliasi_material_terima"`
	IDRekonsiliasiMaterial       int32  `json:"id_rekonsiliasi_material"`
	ActualKirim                  int32  `json:"actual_kirim"`
	Balance                      int32  `json:"balance"`
	CreatedAt                    string `json:"created_at"`
}

type IssueInventoryResponse struct {
	IDRekonsiliasiMaterial int32 `json:"id_rekonsiliasi_material"`
	QtyIssued              int32 `json:"qty_issued"`
	PreviousBalance        int32 `json:"previous_balance"`
	Balance                int32 `json:"balance"`
}

type CreatePackingListItemSizeRequest struct {
	Qty int32 `json:"qty" binding:"required,gt=0"`
}

type CreatePackingListItemRequest struct {
	Color      string                             `json:"color" binding:"required"`
	QtyBox     int32                              `json:"qty_box" binding:"required,gt=0"`
	QtyPerBox  int32                              `json:"qty_per_box" binding:"required,gt=0"`
	BoxNoStart int32                              `json:"box_no_start" binding:"required,gte=0"`
	BoxNoEnd   int32                              `json:"box_no_end" binding:"required,gte=0"`
	Note       string                             `json:"note"`
	Sizes      []CreatePackingListItemSizeRequest `json:"sizes" binding:"required,min=1,dive"`
}

type CreatePackingListRequest struct {
	TotalGarmentPerBox   int32                          `json:"total_garment_per_box" binding:"required,gte=0"`
	TotalReject          int32                          `json:"total_reject" binding:"required,gte=0"`
	IDWO                 int32                          `json:"id_wo" binding:"required,gt=0"`
	IDSuratJalanInternal *int32                         `json:"id_surat_jalan_internal"`
	Items                []CreatePackingListItemRequest `json:"items" binding:"required,min=1,dive"`
}

type PackingListItemSizeResponse struct {
	ID        int32  `json:"id_packing_list_item_size"`
	Qty       int32  `json:"qty"`
	CreatedAt string `json:"created_at"`
}

type PackingListItemResponse struct {
	ID         int32                         `json:"id_packing_list_item"`
	Color      string                        `json:"color"`
	QtyBox     int32                         `json:"qty_box"`
	QtyPerBox  int32                         `json:"qty_per_box"`
	BoxNoStart int32                         `json:"box_no_start"`
	BoxNoEnd   int32                         `json:"box_no_end"`
	Note       string                        `json:"note"`
	CreatedAt  string                        `json:"created_at"`
	Sizes      []PackingListItemSizeResponse `json:"sizes"`
}

type PackingListResponse struct {
	ID                   int32                     `json:"id_packing_list"`
	TotalGarmentPerBox   int32                     `json:"total_garment_per_box"`
	TotalReject          int32                     `json:"total_reject"`
	IDWO                 int32                     `json:"id_wo"`
	IDSuratJalanInternal *int32                    `json:"id_surat_jalan_internal,omitempty"`
	CreatedAt            string                    `json:"created_at"`
	Items                []PackingListItemResponse `json:"items"`
}

type CreateSuratJalanClientRequest struct {
	Tanggal        string `json:"tanggal" binding:"required,datetime=2006-01-02"`
	Qty            int32  `json:"qty" binding:"required,gt=0"`
	Keterangan     string `json:"keterangan"`
	IDMaterialList int32  `json:"id_material_list" binding:"required,gt=0"`
}

type SuratJalanResponse struct {
	Type           string `json:"type"`
	IDSuratJalan   int32  `json:"id_surat_jalan"`
	Tanggal        string `json:"tanggal,omitempty"`
	Qty            int32  `json:"qty,omitempty"`
	Keterangan     string `json:"keterangan,omitempty"`
	IDMaterialList int32  `json:"id_material_list,omitempty"`
	CreatedAt      string `json:"created_at"`
}

type ReceiveInventorySuccessDoc struct {
	Status  string                   `json:"status" example:"success"`
	Message string                   `json:"message" example:"inventory received"`
	Data    ReceiveInventoryResponse `json:"data"`
}

type IssueInventorySuccessDoc struct {
	Status  string                 `json:"status" example:"success"`
	Message string                 `json:"message" example:"inventory issued"`
	Data    IssueInventoryResponse `json:"data"`
}

type PackingListSuccessDoc struct {
	Status  string              `json:"status" example:"success"`
	Message string              `json:"message" example:"packing list created"`
	Data    PackingListResponse `json:"data"`
}

type SuratJalanSuccessDoc struct {
	Status  string             `json:"status" example:"success"`
	Message string             `json:"message" example:"surat jalan created"`
	Data    SuratJalanResponse `json:"data"`
}

type WarehouseValidationErrorDoc struct {
	Status  string                         `json:"status" example:"error"`
	Message string                         `json:"message" example:"bad request"`
	Error   []response.ValidationErrorItem `json:"error"`
}

type WarehouseErrorDetail struct {
	Code string `json:"code" example:"related_data_not_found"`
}

type WarehouseErrorDoc struct {
	Status  string               `json:"status" example:"error"`
	Message string               `json:"message" example:"related data not found"`
	Error   WarehouseErrorDetail `json:"error"`
}
