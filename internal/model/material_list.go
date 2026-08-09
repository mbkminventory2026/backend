package model

import (
	"bytes"
	"encoding/json"
)

// OptionalString distinguishes an omitted PATCH field from an explicit JSON null.
type OptionalString struct {
	Set   bool
	Value *string
}

func (o *OptionalString) UnmarshalJSON(data []byte) error {
	o.Set = true
	if bytes.Equal(data, []byte("null")) {
		o.Value = nil
		return nil
	}
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	o.Value = &value
	return nil
}

// OptionalFloat64 distinguishes an omitted PATCH field from an explicit JSON null.
type OptionalFloat64 struct {
	Set   bool
	Value *float64
}

func (o *OptionalFloat64) UnmarshalJSON(data []byte) error {
	o.Set = true
	if bytes.Equal(data, []byte("null")) {
		o.Value = nil
		return nil
	}
	var value float64
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	o.Value = &value
	return nil
}

// OptionalInt32 distinguishes an omitted PATCH field from an explicit JSON null.
type OptionalInt32 struct {
	Set   bool
	Value *int32
}

func (o *OptionalInt32) UnmarshalJSON(data []byte) error {
	o.Set = true
	if bytes.Equal(data, []byte("null")) {
		o.Value = nil
		return nil
	}
	var value int32
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	o.Value = &value
	return nil
}

type CreateMaterialListRequest struct {
	Name string `json:"name" binding:"required"`
}

type UpdateMaterialListRequest struct {
	Name string `json:"name" binding:"required"`
}

type CreateMaterialListItemBody struct {
	Item         string   `json:"item" binding:"required"`
	Description  string   `json:"description"`
	Qty          int32    `json:"qty"`
	Unit         string   `json:"unit" binding:"required"`
	EstPrice     float64  `json:"est_price"`
	IDWoShell    *int32   `json:"id_wo_shell,omitempty"`
	IDWoTrim     *int32   `json:"id_wo_trim,omitempty"`
	Category     *string  `json:"category"`
	ConsPerPC    *float64 `json:"cons_per_pc"`
	QtyWoScope   *string  `json:"qty_wo_scope"`
	IDQtyWoShell *int32   `json:"id_qty_wo_shell"`
	IDQtyWoSize  *int32   `json:"id_qty_wo_size"`
}

type UpdateMaterialListItemBody struct {
	Item         string          `json:"item" binding:"required"`
	Description  string          `json:"description"`
	Qty          int32           `json:"qty"`
	Unit         string          `json:"unit" binding:"required"`
	EstPrice     float64         `json:"est_price"`
	IDWoShell    *int32          `json:"id_wo_shell,omitempty"`
	IDWoTrim     *int32          `json:"id_wo_trim,omitempty"`
	Category     OptionalString  `json:"category" swaggertype:"string"`
	ConsPerPC    OptionalFloat64 `json:"cons_per_pc" swaggertype:"number"`
	QtyWoScope   OptionalString  `json:"qty_wo_scope" swaggertype:"string"`
	IDQtyWoShell OptionalInt32   `json:"id_qty_wo_shell" swaggertype:"integer"`
	IDQtyWoSize  OptionalInt32   `json:"id_qty_wo_size" swaggertype:"integer"`
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
	IDMaterialListItem int32    `json:"id_material_list_item"`
	IDMaterialList     int32    `json:"id_material_list"`
	Item               string   `json:"item"`
	Description        string   `json:"description"`
	Qty                int32    `json:"qty"`
	Unit               string   `json:"unit"`
	EstPrice           float64  `json:"est_price"`
	Category           *string  `json:"category"`
	ConsPerPC          *float64 `json:"cons_per_pc"`
	QtyWoScope         *string  `json:"qty_wo_scope"`
	IDQtyWoShell       *int32   `json:"id_qty_wo_shell"`
	IDQtyWoSize        *int32   `json:"id_qty_wo_size"`
	CreatedAt          string   `json:"created_at"`
	QtySuratJalan      int32    `json:"qty_surat_jalan"`
	QtyReceived        int32    `json:"qty_received"`
	MlName             string   `json:"ml_name"`
	MlIsLocked         bool     `json:"ml_is_locked"`
	IDWo               int32    `json:"id_wo"`
	Buyer              string   `json:"buyer"`
	Model              string   `json:"model"`
}
