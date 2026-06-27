package model

import "permatatex-inventory/pkg/response"

type ReceiveInventoryRequest struct {
	Tanggal                string `json:"tanggal" binding:"required,datetime=2006-01-02"`
	Qty                    int32  `json:"qty" binding:"required,gt=0"`
	Keterangan             string `json:"keterangan"`
	IDMaterialListItem     int32  `json:"id_material_list_item" binding:"required,gt=0"`
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
	IDMaterialListItem           int32  `json:"id_material_list_item"`
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
	IDWOShellSize int32 `json:"id_wo_shell_size" binding:"required,gt=0"`
	Qty           int32 `json:"qty" binding:"gte=0"`
}

type CreatePackingListRejectSizeRequest struct {
	IDWOShellSize int32 `json:"id_wo_shell_size" binding:"required,gt=0"`
	Qty           int32 `json:"qty" binding:"gte=0"`
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
	TotalGarmentPerBox   int32                                `json:"total_garment_per_box" binding:"required,gte=0"`
	TotalReject          int32                                `json:"total_reject" binding:"required,gte=0"`
	IDWO                 int32                                `json:"id_wo" binding:"required,gt=0"`
	IDSuratJalanInternal *int32                               `json:"id_surat_jalan_internal"`
	Items                []CreatePackingListItemRequest       `json:"items" binding:"required,min=1,dive"`
	RejectSizes          []CreatePackingListRejectSizeRequest `json:"reject_sizes" binding:"dive"`
}

type PackingListItemSizeResponse struct {
	ID            int32  `json:"id_packing_list_item_size"`
	IDWOShellSize int32  `json:"id_wo_shell_size"`
	IDSize        *int32 `json:"id_size,omitempty"`
	Size          string `json:"size"`
	Qty           int32  `json:"qty"`
	CreatedAt     string `json:"created_at"`
}

type PackingListRejectSizeResponse struct {
	ID            int32  `json:"id_packing_list_reject_size"`
	IDWOShellSize int32  `json:"id_wo_shell_size"`
	IDSize        *int32 `json:"id_size,omitempty"`
	Size          string `json:"size"`
	Qty           int32  `json:"qty"`
	CreatedAt     string `json:"created_at"`
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
	ID                   int32                           `json:"id_packing_list"`
	TotalGarmentPerBox   int32                           `json:"total_garment_per_box"`
	TotalReject          int32                           `json:"total_reject"`
	IDWO                 int32                           `json:"id_wo"`
	IDSuratJalanInternal *int32                          `json:"id_surat_jalan_internal,omitempty"`
	CreatedAt            string                          `json:"created_at"`
	Items                []PackingListItemResponse       `json:"items"`
	RejectSizes          []PackingListRejectSizeResponse `json:"reject_sizes"`
}

type CreateSuratJalanClientRequest struct {
	Tanggal            string `json:"tanggal" binding:"required,datetime=2006-01-02"`
	Qty                int32  `json:"qty" binding:"required,gt=0"`
	Keterangan         string `json:"keterangan"`
	IDMaterialListItem int32  `json:"id_material_list_item" binding:"required,gt=0"`
}

type SuratJalanResponse struct {
	Type               string `json:"type"`
	IDSuratJalan       int32  `json:"id_surat_jalan"`
	Tanggal            string `json:"tanggal,omitempty"`
	Qty                int32  `json:"qty,omitempty"`
	Keterangan         string `json:"keterangan,omitempty"`
	IDMaterialListItem int32  `json:"id_material_list_item,omitempty"`
	CreatedAt          string `json:"created_at"`
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

type CreateSimpleReceivedRequest struct {
	Tanggal            string `json:"tanggal" binding:"required,datetime=2006-01-02"`
	Qty                int32  `json:"qty" binding:"required,gt=0"`
	Keterangan         string `json:"keterangan"`
	IDMaterialListItem int32  `json:"id_material_list_item" binding:"required,gt=0"`
}

type UpdateSimpleReceivedRequest struct {
	Tanggal    string `json:"tanggal" binding:"required,datetime=2006-01-02"`
	Qty        int32  `json:"qty" binding:"required,gt=0"`
	Keterangan string `json:"keterangan"`
}

type SimpleReceivedResponse struct {
	IDReceived         int32  `json:"id_received"`
	Tanggal            string `json:"tanggal"`
	Qty                int32  `json:"qty"`
	Keterangan         string `json:"keterangan"`
	IDMaterialListItem int32  `json:"id_material_list_item"`
	CreatedAt          string `json:"created_at"`
}

type SimpleReceivedDetailResponse struct {
	IDReceived          int32  `json:"id_received"`
	Tanggal             string `json:"tanggal"`
	Qty                 int32  `json:"qty"`
	Keterangan          string `json:"keterangan"`
	IDMaterialListItem  int32  `json:"id_material_list_item"`
	MaterialItem        string `json:"material_item"`
	MaterialDescription string `json:"material_description"`
	IDWO                int32  `json:"id_wo"`
	CreatedAt           string `json:"created_at"`
}

type SimpleReceivedListItem struct {
	IDReceived          int32  `json:"id_received"`
	Tanggal             string `json:"tanggal"`
	Qty                 int32  `json:"qty"`
	Keterangan          string `json:"keterangan"`
	IDMaterialListItem  int32  `json:"id_material_list_item"`
	MaterialItem        string `json:"material_item"`
	MaterialDescription string `json:"material_description"`
	IDWO                int32  `json:"id_wo"`
	CreatedAt           string `json:"created_at"`
}

type SimpleReceivedListResponse struct {
	Items      []SimpleReceivedListItem `json:"items"`
	Pagination PaginationMeta           `json:"pagination"`
}

type MLIHistoryEntry struct {
	ID         int32  `json:"id"`
	Tanggal    string `json:"tanggal"`
	Qty        int32  `json:"qty"`
	Keterangan string `json:"keterangan"`
	CreatedAt  string `json:"created_at"`
}

type MLIHistoryResponse struct {
	SuratJalan []MLIHistoryEntry `json:"surat_jalan"`
	Received   []MLIHistoryEntry `json:"received"`
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

// Surat Jalan Internal types
type CreateSuratJalanInternalItemRequest struct {
	No        int    `json:"no"`
	Deskripsi string `json:"deskripsi" binding:"required"`
	Qty       int32  `json:"qty"`
	Note      string `json:"note"`
}

type CreateSuratJalanInternalRequest struct {
	IDWO           int32                                  `json:"id_wo" binding:"required,gt=0"`
	NoDokumen      string                                 `json:"no_dokumen" binding:"required"`
	Deskripsi      string                                 `json:"deskripsi"`
	Items          []CreateSuratJalanInternalItemRequest `json:"items"`
	IDPackingLists []int32                                `json:"id_packing_lists"`
}

type AssignPackingListRequest struct {
	IDPackingList int32 `json:"id_packing_list" binding:"required,gt=0"`
}

type SuratJalanInternalShellRow struct {
	No        int    `json:"no"`
	Deskripsi string `json:"deskripsi"`
	Color     string `json:"color,omitempty"`
	Qty       int32  `json:"qty"`
	Note      string `json:"note"`
}

type SuratJalanInternalPackingListRow struct {
	IDPackingList      int32  `json:"id_packing_list"`
	TotalGarmentPerBox int32  `json:"total_garment_per_box"`
	TotalReject        int32  `json:"total_reject"`
	IDWO               int32  `json:"id_wo"`
	CreatedAt          string `json:"created_at"`
}

type SuratJalanInternalListItem struct {
	ID               int32  `json:"id_surat_jalan_internal"`
	IDWO             *int32 `json:"id_wo,omitempty"`
	NoDokumen        string `json:"no_dokumen"`
	Deskripsi        string `json:"deskripsi"`
	Buyer            string `json:"buyer,omitempty"`
	Model            string `json:"model,omitempty"`
	PackingListCount int32  `json:"packing_list_count"`
	CreatedAt        string `json:"created_at"`
}

type SuratJalanInternalListResponse struct {
	Items      []SuratJalanInternalListItem `json:"items"`
	Pagination PaginationMeta               `json:"pagination"`
}

type SuratJalanInternalDetailResponse struct {
	ID           int32                              `json:"id_surat_jalan_internal"`
	IDWO         *int32                             `json:"id_wo,omitempty"`
	NoDokumen    string                             `json:"no_dokumen"`
	Deskripsi    string                             `json:"deskripsi"`
	Buyer        string                             `json:"buyer,omitempty"`
	Model        string                             `json:"model,omitempty"`
	WOQty        int32                              `json:"wo_qty,omitempty"`
	CreatedAt    string                             `json:"created_at"`
	PackingLists []SuratJalanInternalPackingListRow `json:"packing_lists"`
	WOShells     []SuratJalanInternalShellRow       `json:"wo_shells"`
}

type SuratJalanInternalCreateResponse struct {
	ID        int32  `json:"id_surat_jalan_internal"`
	IDWO      int32  `json:"id_wo"`
	NoDokumen string `json:"no_dokumen"`
	Deskripsi string `json:"deskripsi"`
	CreatedAt string `json:"created_at"`
}

type SuratJalanInternalCreateSuccessDoc struct {
	Status  string                            `json:"status" example:"success"`
	Message string                            `json:"message" example:"surat jalan internal created"`
	Data    SuratJalanInternalCreateResponse  `json:"data"`
}

type SuratJalanInternalDetailSuccessDoc struct {
	Status  string                            `json:"status" example:"success"`
	Message string                            `json:"message" example:"surat jalan internal retrieved"`
	Data    SuratJalanInternalDetailResponse  `json:"data"`
}

type SuratJalanInternalListSuccessDoc struct {
	Status  string                          `json:"status" example:"success"`
	Message string                          `json:"message" example:"surat jalan internals retrieved"`
	Data    SuratJalanInternalListResponse  `json:"data"`
}
