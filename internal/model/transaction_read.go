package model

import "permatatex-inventory/pkg/response"

type TransactionListFilter struct {
	ListQueryFilter
	IDMitra *int32
}

type PaginationMeta struct {
	Page       int32 `json:"page"`
	Limit      int32 `json:"limit"`
	TotalItems int64 `json:"total_items"`
	TotalPages int64 `json:"total_pages"`
}

type WorkOrderListItem struct {
	ID                int32  `json:"id_wo"`
	Buyer             string `json:"buyer"`
	Model             string `json:"model"`
	Qty               int32  `json:"qty"`
	FOBCMT            bool   `json:"fob_cmt"`
	Delivery          string `json:"delivery"`
	IDPOClientItem    int32  `json:"id_po_client_item"`
	Status            string `json:"status"`
	ClosedAt          string `json:"closed_at,omitempty"`
	PONumber          string `json:"po_number"`
	POClientItemStyle string `json:"po_client_item_style"`
	CreatedAt         string `json:"created_at"`
}

type WorkOrderDetailResponse struct {
	ID                int32                    `json:"id_wo"`
	Buyer             string                   `json:"buyer"`
	Model             string                   `json:"model"`
	Qty               int32                    `json:"qty"`
	FOBCMT            bool                     `json:"fob_cmt"`
	Delivery          string                   `json:"delivery"`
	IDPOClientItem    int32                    `json:"id_po_client_item"`
	Status            string                   `json:"status"`
	ClosedByUserID    *int32                   `json:"closed_by_user_id,omitempty"`
	ClosedAt          string                   `json:"closed_at,omitempty"`
	PONumber          string                   `json:"po_number"`
	POClientItemStyle string                   `json:"po_client_item_style"`
	CreatedAt         string                   `json:"created_at"`
	Shells            []WorkOrderShellResponse `json:"shells"`
	Trims             []WorkOrderTrimResponse  `json:"trims"`
	MaterialLists     []MaterialListResponse   `json:"material_lists"`
	Retur             *ReturClientResponse     `json:"retur,omitempty"`
}

type WorkOrderListResponse struct {
	Items      []WorkOrderListItem `json:"items"`
	Pagination PaginationMeta      `json:"pagination"`
}

type POClientListItem struct {
	ID        int32  `json:"id_po_client"`
	PONumber  string `json:"po_number"`
	Tanggal   string `json:"tanggal"`
	Season    string `json:"season"`
	Delivery  string `json:"delivery"`
	IDMitra   int32  `json:"id_mitra"`
	MitraName string `json:"mitra_name"`
	CreatedAt string `json:"created_at"`
}

type POClientDetailResponse struct {
	ID              int32                     `json:"id_po_client"`
	PONumber        string                    `json:"po_number"`
	Tanggal         string                    `json:"tanggal"`
	Season          string                    `json:"season"`
	Delivery        string                    `json:"delivery"`
	PaymentTerm     string                    `json:"payment_term"`
	File            string                    `json:"file"`
	IDMitra         int32                     `json:"id_mitra"`
	MitraName       string                    `json:"mitra_name"`
	CreatedAt       string                    `json:"created_at"`
	Items           []POClientItemResponse    `json:"items"`
	PenanggungJawab []PenanggungJawabResponse `json:"penanggung_jawab"`
}

type POClientListResponse struct {
	Items      []POClientListItem `json:"items"`
	Pagination PaginationMeta     `json:"pagination"`
}

type PRInternalListItem struct {
	ID         int32  `json:"id_pr_internal"`
	Tanggal    string `json:"tanggal"`
	Nama       string `json:"nama"`
	Departemen string `json:"departemen"`
	VendorName string `json:"vendor_name"`
	Projek     string `json:"projek"`
	IDWO       int32  `json:"id_wo"`
	IDUser     int32  `json:"id_user"`
	Status     string `json:"status"`
	ApprovedAt string `json:"approved_at,omitempty"`
	CreatedAt  string `json:"created_at"`
}

type PRInternalListResponse struct {
	Items      []PRInternalListItem `json:"items"`
	Pagination PaginationMeta       `json:"pagination"`
}

type POInternalListItem struct {
	ID           int32  `json:"id_po_internal"`
	Tanggal      string `json:"tanggal"`
	NamaPO       string `json:"nama_po"`
	SupplierName string `json:"supplier_name"`
	Currency     string `json:"currency"`
	CPO          string `json:"cpo"`
	ShipDate     string `json:"ship_date"`
	IDPRInternal int32  `json:"id_pr_internal"`
	CreatedAt    string `json:"created_at"`
}

type POInternalListResponse struct {
	Items      []POInternalListItem `json:"items"`
	Pagination PaginationMeta       `json:"pagination"`
}

type PackingListListItem struct {
	ID                   int32  `json:"id_packing_list"`
	TotalGarmentPerBox   int32  `json:"total_garment_per_box"`
	TotalReject          int32  `json:"total_reject"`
	IDWO                 int32  `json:"id_wo"`
	IDSuratJalanInternal *int32 `json:"id_surat_jalan_internal,omitempty"`
	Buyer                string `json:"buyer"`
	Model                string `json:"model"`
	CreatedAt            string `json:"created_at"`
}

type PackingListDetailResponse struct {
	ID                   int32                     `json:"id_packing_list"`
	TotalGarmentPerBox   int32                     `json:"total_garment_per_box"`
	TotalReject          int32                     `json:"total_reject"`
	IDWO                 int32                     `json:"id_wo"`
	IDSuratJalanInternal *int32                    `json:"id_surat_jalan_internal,omitempty"`
	Buyer                string                    `json:"buyer"`
	Model                string                    `json:"model"`
	CreatedAt            string                    `json:"created_at"`
	Items                []PackingListItemResponse `json:"items"`
}

type PackingListListResponse struct {
	Items      []PackingListListItem `json:"items"`
	Pagination PaginationMeta        `json:"pagination"`
}

type SuratJalanClientListItem struct {
	ID                  int32  `json:"id_surat_jalan_client"`
	Tanggal             string `json:"tanggal"`
	Qty                 int32  `json:"qty"`
	Keterangan          string `json:"keterangan"`
	IDMaterialList      int32  `json:"id_material_list"`
	MaterialDescription string `json:"material_description"`
	IDWO                int32  `json:"id_wo"`
	CreatedAt           string `json:"created_at"`
}

type SuratJalanClientDetailResponse struct {
	ID                  int32  `json:"id_surat_jalan_client"`
	Tanggal             string `json:"tanggal"`
	Qty                 int32  `json:"qty"`
	Keterangan          string `json:"keterangan"`
	IDMaterialList      int32  `json:"id_material_list"`
	MaterialDescription string `json:"material_description"`
	IDWO                int32  `json:"id_wo"`
	CreatedAt           string `json:"created_at"`
}

type SuratJalanClientListResponse struct {
	Items      []SuratJalanClientListItem `json:"items"`
	Pagination PaginationMeta             `json:"pagination"`
}

type SuratJalanInternalListItem struct {
	ID        int32  `json:"id_surat_jalan_internal"`
	CreatedAt string `json:"created_at"`
}

type SuratJalanInternalDetailResponse struct {
	ID        int32  `json:"id_surat_jalan_internal"`
	CreatedAt string `json:"created_at"`
}

type SuratJalanInternalListResponse struct {
	Items      []SuratJalanInternalListItem `json:"items"`
	Pagination PaginationMeta               `json:"pagination"`
}

type WorkOrderListSuccessDoc struct {
	Status  string                `json:"status" example:"success"`
	Message string                `json:"message" example:"work orders retrieved"`
	Data    WorkOrderListResponse `json:"data"`
}

type WorkOrderDetailSuccessDoc struct {
	Status  string                  `json:"status" example:"success"`
	Message string                  `json:"message" example:"work order retrieved"`
	Data    WorkOrderDetailResponse `json:"data"`
}

type POClientListSuccessDoc struct {
	Status  string               `json:"status" example:"success"`
	Message string               `json:"message" example:"po clients retrieved"`
	Data    POClientListResponse `json:"data"`
}

type POClientDetailSuccessDoc struct {
	Status  string                 `json:"status" example:"success"`
	Message string                 `json:"message" example:"po client retrieved"`
	Data    POClientDetailResponse `json:"data"`
}

type PRInternalListSuccessDoc struct {
	Status  string                 `json:"status" example:"success"`
	Message string                 `json:"message" example:"pr internals retrieved"`
	Data    PRInternalListResponse `json:"data"`
}

type PRInternalDetailSuccessDoc struct {
	Status  string             `json:"status" example:"success"`
	Message string             `json:"message" example:"pr internal retrieved"`
	Data    PRInternalResponse `json:"data"`
}

type POInternalListSuccessDoc struct {
	Status  string                 `json:"status" example:"success"`
	Message string                 `json:"message" example:"po internals retrieved"`
	Data    POInternalListResponse `json:"data"`
}

type POInternalDetailSuccessDoc struct {
	Status  string             `json:"status" example:"success"`
	Message string             `json:"message" example:"po internal retrieved"`
	Data    POInternalResponse `json:"data"`
}

type PackingListListSuccessDoc struct {
	Status  string                  `json:"status" example:"success"`
	Message string                  `json:"message" example:"packing lists retrieved"`
	Data    PackingListListResponse `json:"data"`
}

type PackingListDetailSuccessDoc struct {
	Status  string                    `json:"status" example:"success"`
	Message string                    `json:"message" example:"packing list retrieved"`
	Data    PackingListDetailResponse `json:"data"`
}

type SuratJalanClientListSuccessDoc struct {
	Status  string                       `json:"status" example:"success"`
	Message string                       `json:"message" example:"surat jalan clients retrieved"`
	Data    SuratJalanClientListResponse `json:"data"`
}

type SuratJalanClientDetailSuccessDoc struct {
	Status  string                         `json:"status" example:"success"`
	Message string                         `json:"message" example:"surat jalan client retrieved"`
	Data    SuratJalanClientDetailResponse `json:"data"`
}

type SuratJalanInternalListSuccessDoc struct {
	Status  string                         `json:"status" example:"success"`
	Message string                         `json:"message" example:"surat jalan internals retrieved"`
	Data    SuratJalanInternalListResponse `json:"data"`
}

type SuratJalanInternalDetailSuccessDoc struct {
	Status  string                           `json:"status" example:"success"`
	Message string                           `json:"message" example:"surat jalan internal retrieved"`
	Data    SuratJalanInternalDetailResponse `json:"data"`
}

type TransactionReadValidationErrorDoc struct {
	Status  string                         `json:"status" example:"error"`
	Message string                         `json:"message" example:"bad request"`
	Error   []response.ValidationErrorItem `json:"error"`
}

type TransactionReadErrorDetail struct {
	Code string `json:"code" example:"transaction_not_found"`
}

type TransactionReadErrorDoc struct {
	Status  string                     `json:"status" example:"error"`
	Message string                     `json:"message" example:"transaction not found"`
	Error   TransactionReadErrorDetail `json:"error"`
}
