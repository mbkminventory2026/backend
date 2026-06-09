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
