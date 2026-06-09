package model

type CreateMaterialListRequest struct {
	Name string `json:"name" binding:"required"`
}

type UpdateMaterialListRequest struct {
	Name string `json:"name" binding:"required"`
}

type CreateMaterialListItemBody struct {
	Item        string  `json:"item" binding:"required"`
	Description string  `json:"description"`
	Qty         int32   `json:"qty"`
	Unit        string  `json:"unit" binding:"required"`
	EstPrice    float64 `json:"est_price"`
	IDWoShell   *int32  `json:"id_wo_shell,omitempty"`
	IDWoTrim    *int32  `json:"id_wo_trim,omitempty"`
}

type UpdateMaterialListItemBody struct {
	Item        string  `json:"item" binding:"required"`
	Description string  `json:"description"`
	Qty         int32   `json:"qty"`
	Unit        string  `json:"unit" binding:"required"`
	EstPrice    float64 `json:"est_price"`
	IDWoShell   *int32  `json:"id_wo_shell,omitempty"`
	IDWoTrim    *int32  `json:"id_wo_trim,omitempty"`
}

type MaterialListListResponse struct {
	Items []MaterialListResponse `json:"items"`
}

type MaterialListPageItem struct {
	IDMaterialList   int32  `json:"id_material_list"`
	IDWo             int32  `json:"id_wo"`
	Name             string `json:"name"`
	IsLocked         bool   `json:"is_locked"`
	CreatedAt        string `json:"created_at"`
	Buyer            string `json:"buyer"`
	Model            string `json:"model"`
	WoQty            int32  `json:"wo_qty"`
	ItemCount        int32  `json:"item_count"`
	TotalQtySj       int32  `json:"total_qty_sj"`
	TotalQtyReceived int32  `json:"total_qty_received"`
}

type MaterialListPageResponse struct {
	Items      []MaterialListPageItem `json:"items"`
	Pagination PaginationMeta         `json:"pagination"`
}

type MaterialListItemDetailResponse struct {
	IDMaterialListItem int32   `json:"id_material_list_item"`
	IDMaterialList     int32   `json:"id_material_list"`
	Item               string  `json:"item"`
	Description        string  `json:"description"`
	Qty                int32   `json:"qty"`
	Unit               string  `json:"unit"`
	EstPrice           float64 `json:"est_price"`
	CreatedAt          string  `json:"created_at"`
	QtySuratJalan      int32   `json:"qty_surat_jalan"`
	QtyReceived        int32   `json:"qty_received"`
	MlName             string  `json:"ml_name"`
	MlIsLocked         bool    `json:"ml_is_locked"`
	IDWo               int32   `json:"id_wo"`
	Buyer              string  `json:"buyer"`
	Model              string  `json:"model"`
}
