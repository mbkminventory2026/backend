package model

import "permatatex-inventory/pkg/response"

type CreatePOClientItemRequest struct {
	Style       string  `json:"style" binding:"required"`
	Colour      string  `json:"colour" binding:"required"`
	Description string  `json:"description"`
	Qty         int32   `json:"qty" binding:"required,gt=0"`
	Price       float64 `json:"price" binding:"required,gte=0"`
}

type CreatePenanggungJawabRequest struct {
	Nama   string `json:"nama" binding:"required"`
	NoTelp string `json:"no_telp" binding:"required"`
	Email  string `json:"email" binding:"omitempty,email"`
}

type CreatePOClientRequest struct {
	PONumber        string                         `json:"po_number" binding:"required"`
	Tanggal         string                         `json:"tanggal" binding:"required,datetime=2006-01-02"`
	Season          string                         `json:"season"`
	Delivery        string                         `json:"delivery" binding:"required,datetime=2006-01-02"`
	PaymentTerm     string                         `json:"payment_term"`
	File            string                         `json:"file"`
	IDMitra         int32                          `json:"id_mitra" binding:"required"`
	Items           []CreatePOClientItemRequest    `json:"items" binding:"required,min=1,dive"`
	PenanggungJawab []CreatePenanggungJawabRequest `json:"penanggung_jawab" binding:"required,min=1,dive"`
}

type CreatePRInternalItemRequest struct {
	Item        string  `json:"item" binding:"required"`
	Description string  `json:"description"`
	Qty         int32   `json:"qty" binding:"required,gt=0"`
	Unit        string  `json:"unit" binding:"required"`
	EstPrice    float64 `json:"est_price" binding:"required,gte=0"`
}

type CreatePRInternalRequest struct {
	Tanggal       string                        `json:"tanggal" binding:"required,datetime=2006-01-02"`
	Nama          string                        `json:"nama" binding:"required"`
	Departemen    string                        `json:"departemen" binding:"required"`
	VendorName    string                        `json:"vendor_name" binding:"required"`
	VendorAddress string                        `json:"vendor_address"`
	VendorTelp    string                        `json:"vendor_telp"`
	Projek        string                        `json:"projek" binding:"required"`
	IDWO          int32                         `json:"id_wo" binding:"required"`
	Items         []CreatePRInternalItemRequest `json:"items" binding:"required,min=1,dive"`
}

type CreatePOInternalItemRequest struct {
	Item        string  `json:"item" binding:"required"`
	Description string  `json:"description"`
	Qty         int32   `json:"qty" binding:"required,gt=0"`
	Unit        string  `json:"unit" binding:"required"`
	UnitPrice   float64 `json:"unit_price" binding:"required,gte=0"`
}

type CreatePOInternalRequest struct {
	Tanggal         string                        `json:"tanggal" binding:"required,datetime=2006-01-02"`
	NamaPO          string                        `json:"nama_po" binding:"required"`
	SupplierName    string                        `json:"supplier_name" binding:"required"`
	SupplierAddr    string                        `json:"supplier_addr"`
	SupplierContact string                        `json:"supplier_contact"`
	SupplierEmail   string                        `json:"supplier_email" binding:"omitempty,email"`
	SupplierTelp    string                        `json:"supplier_telp"`
	SupplierFax     string                        `json:"supplier_fax"`
	Currency        string                        `json:"currency" binding:"required"`
	CPO             string                        `json:"cpo"`
	Term            string                        `json:"term"`
	ShipDate        string                        `json:"ship_date" binding:"required,datetime=2006-01-02"`
	IDPRInternal    int32                         `json:"id_pr_internal" binding:"required"`
	Items           []CreatePOInternalItemRequest `json:"items" binding:"required,min=1,dive"`
}

type POClientItemResponse struct {
	ID          int32   `json:"id_po_client_item"`
	Style       string  `json:"style"`
	Colour      string  `json:"colour"`
	Description string  `json:"description"`
	Qty         int32   `json:"qty"`
	Price       float64 `json:"price"`
	CreatedAt   string  `json:"created_at"`
}

type PenanggungJawabResponse struct {
	ID        int32  `json:"id_penanggung_jawab"`
	Nama      string `json:"nama"`
	NoTelp    string `json:"no_telp"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
}

type POClientResponse struct {
	ID              int32                     `json:"id_po_client"`
	PONumber        string                    `json:"po_number"`
	Tanggal         string                    `json:"tanggal"`
	Season          string                    `json:"season"`
	Delivery        string                    `json:"delivery"`
	PaymentTerm     string                    `json:"payment_term"`
	File            string                    `json:"file"`
	IDMitra         int32                     `json:"id_mitra"`
	CreatedAt       string                    `json:"created_at"`
	Items           []POClientItemResponse    `json:"items"`
	PenanggungJawab []PenanggungJawabResponse `json:"penanggung_jawab"`
}

type PRInternalItemResponse struct {
	ID          int32   `json:"id_pr_internal_item"`
	Item        string  `json:"item"`
	Description string  `json:"description"`
	Qty         int32   `json:"qty"`
	Unit        string  `json:"unit"`
	EstPrice    float64 `json:"est_price"`
	CreatedAt   string  `json:"created_at"`
}

type PRInternalResponse struct {
	ID            int32                    `json:"id_pr_internal"`
	Tanggal       string                   `json:"tanggal"`
	Nama          string                   `json:"nama"`
	Departemen    string                   `json:"departemen"`
	VendorName    string                   `json:"vendor_name"`
	VendorAddress string                   `json:"vendor_address"`
	VendorTelp    string                   `json:"vendor_telp"`
	Projek        string                   `json:"projek"`
	IDWO          int32                    `json:"id_wo"`
	IDUser        int32                    `json:"id_user"`
	Status        string                   `json:"status"`
	ApprovedByID  *int32                   `json:"approved_by_user_id,omitempty"`
	ApprovedAt    string                   `json:"approved_at,omitempty"`
	CreatedAt     string                   `json:"created_at"`
	Items         []PRInternalItemResponse `json:"items"`
}

type POInternalItemResponse struct {
	ID          int32   `json:"id_po_internal_item"`
	Item        string  `json:"item"`
	Description string  `json:"description"`
	Qty         int32   `json:"qty"`
	Unit        string  `json:"unit"`
	UnitPrice   float64 `json:"unit_price"`
	CreatedAt   string  `json:"created_at"`
}

type POInternalResponse struct {
	ID              int32                    `json:"id_po_internal"`
	Tanggal         string                   `json:"tanggal"`
	NamaPO          string                   `json:"nama_po"`
	SupplierName    string                   `json:"supplier_name"`
	SupplierAddr    string                   `json:"supplier_addr"`
	SupplierContact string                   `json:"supplier_contact"`
	SupplierEmail   string                   `json:"supplier_email"`
	SupplierTelp    string                   `json:"supplier_telp"`
	SupplierFax     string                   `json:"supplier_fax"`
	Currency        string                   `json:"currency"`
	CPO             string                   `json:"cpo"`
	Term            string                   `json:"term"`
	ShipDate        string                   `json:"ship_date"`
	IDPRInternal    int32                    `json:"id_pr_internal"`
	CreatedAt       string                   `json:"created_at"`
	Items           []POInternalItemResponse `json:"items"`
}

type PRInternalStatusResponse struct {
	ID           int32  `json:"id_pr_internal"`
	Status       string `json:"status"`
	ApprovedByID *int32 `json:"approved_by_user_id,omitempty"`
	ApprovedAt   string `json:"approved_at,omitempty"`
}

type POClientSuccessDoc struct {
	Status  string           `json:"status" example:"success"`
	Message string           `json:"message" example:"po client created"`
	Data    POClientResponse `json:"data"`
}

type PRInternalSuccessDoc struct {
	Status  string             `json:"status" example:"success"`
	Message string             `json:"message" example:"pr internal created"`
	Data    PRInternalResponse `json:"data"`
}

type POInternalSuccessDoc struct {
	Status  string             `json:"status" example:"success"`
	Message string             `json:"message" example:"po internal created"`
	Data    POInternalResponse `json:"data"`
}

type PRInternalStatusSuccessDoc struct {
	Status  string                   `json:"status" example:"success"`
	Message string                   `json:"message" example:"pr internal approved"`
	Data    PRInternalStatusResponse `json:"data"`
}

type TransactionValidationErrorDoc struct {
	Status  string                         `json:"status" example:"error"`
	Message string                         `json:"message" example:"bad request"`
	Error   []response.ValidationErrorItem `json:"error"`
}

type TransactionErrorDetail struct {
	Code string `json:"code" example:"related_data_not_found"`
}

type TransactionErrorDoc struct {
	Status  string                 `json:"status" example:"error"`
	Message string                 `json:"message" example:"related data not found"`
	Error   TransactionErrorDetail `json:"error"`
}
