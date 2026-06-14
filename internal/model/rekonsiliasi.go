package model

import "permatatex-inventory/pkg/response"

type RekonsiliasiListFilter struct {
	ListQueryFilter
	IDWo *int32
}

type CreateRekonsiliasiRequest struct {
	IDWo int32 `json:"id_wo" binding:"required,gt=0"`
}

type UpdateRekonsiliasiTerimaEntryRequest struct {
	IDRekonsiliasiTerimaEntry *int32 `json:"id_rekonsiliasi_terima_entry,omitempty"`
	EntryType                 string `json:"entry_type" binding:"required,oneof=awal untuk ambil"`
	EntryLabel                string `json:"entry_label" binding:"required"`
	Qty                       int32  `json:"qty" binding:"gte=0"`
	Note                      string `json:"note"`
}

type UpdateRekonsiliasiMaterialRowRequest struct {
	IDRekonsiliasiMaterialRow int32                                  `json:"id_rekonsiliasi_material_row" binding:"required,gt=0"`
	RatioInput                float64                                `json:"ratio_input" binding:"gte=0"`
	QtyPerPcsInput            float64                                `json:"qty_per_pcs_input" binding:"gte=0"`
	QtyActualKirimManual      int32                                  `json:"qty_actual_kirim_manual" binding:"gte=0"`
	RejectQty                 int32                                  `json:"reject_qty" binding:"gte=0"`
	ReturQty                  int32                                  `json:"retur_qty" binding:"gte=0"`
	Keterangan                string                                 `json:"keterangan"`
	TerimaEntries             []UpdateRekonsiliasiTerimaEntryRequest `json:"terima_entries" binding:"omitempty,dive"`
}

type UpdateRekonsiliasiRequest struct {
	MaterialRows []UpdateRekonsiliasiMaterialRowRequest `json:"material_rows" binding:"required,min=1,dive"`
}

type RekonsiliasiListItem struct {
	IDRekonsiliasi    int32  `json:"id_rekonsiliasi"`
	IDWo              int32  `json:"id_wo"`
	NamaWo            string `json:"nama_wo"`
	Buyer             string `json:"buyer"`
	Brand             string `json:"brand"`
	Style             string `json:"style"`
	QtyPo             int32  `json:"qty_po"`
	PlanCutTotal      int64  `json:"plan_cut_total"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
	CreatedByUsername string `json:"created_by_username"`
	UpdatedByUsername string `json:"updated_by_username"`
}

type RekonsiliasiListResponse struct {
	Items      []RekonsiliasiListItem `json:"items"`
	Pagination PaginationMeta         `json:"pagination"`
}

type RekonsiliasiHeaderResponse struct {
	IDRekonsiliasi    int32    `json:"id_rekonsiliasi"`
	IDWo              int32    `json:"id_wo"`
	Jasa              string   `json:"jasa"`
	NoPO              string   `json:"no_po"`
	Delivery          string   `json:"delivery"`
	Buyer             string   `json:"buyer"`
	Brand             string   `json:"brand"`
	Style             string   `json:"style"`
	QtyPO             int32    `json:"qty_po"`
	PlanCutTotal      int64    `json:"plan_cut_total"`
	ConsBajuSummary   string   `json:"cons_baju_summary"`
	NamaBahan         string   `json:"nama_bahan"`
	WarnaKainSummary  []string `json:"warna_kain_summary"`
	CreatedAt         string   `json:"created_at"`
	UpdatedAt         string   `json:"updated_at"`
	CreatedByUsername string   `json:"created_by_username"`
	UpdatedByUsername string   `json:"updated_by_username"`
}

type RekonsiliasiColorSizeSummaryResponse struct {
	Size     string `json:"size"`
	QtyOrder int32  `json:"qty_order"`
	QtyKirim int32  `json:"qty_kirim"`
	Balance  int32  `json:"balance"`
}

type RekonsiliasiColorSummaryResponse struct {
	Color         string                                 `json:"color"`
	QtyOrder      int32                                  `json:"qty_order"`
	QtyKirim      int32                                  `json:"qty_kirim"`
	Balance       int32                                  `json:"balance"`
	SizeBreakdown []RekonsiliasiColorSizeSummaryResponse `json:"size_breakdown"`
}

type RekonsiliasiTerimaEntryResponse struct {
	IDRekonsiliasiTerimaEntry int32  `json:"id_rekonsiliasi_terima_entry"`
	EntryType                 string `json:"entry_type"`
	EntryLabel                string `json:"entry_label"`
	Qty                       int32  `json:"qty"`
	Note                      string `json:"note"`
	CreatedAt                 string `json:"created_at"`
	UpdatedAt                 string `json:"updated_at"`
}

type RekonsiliasiMaterialRowResponse struct {
	IDRekonsiliasiMaterialRow int32                             `json:"id_rekonsiliasi_material_row"`
	RowNo                     int32                             `json:"row_no"`
	Kategori                  string                            `json:"kategori"`
	Description               string                            `json:"description"`
	SizeLabel                 string                            `json:"size_label"`
	RatioSource               float64                           `json:"ratio_source"`
	RatioInput                float64                           `json:"ratio_input"`
	QtyPerPcsInput            float64                           `json:"qty_per_pcs_input"`
	QtyWO                     int32                             `json:"qty_wo"`
	Toleransi                 int32                             `json:"toleransi"`
	Satuan                    string                            `json:"satuan"`
	QtyActualKirimSource      int32                             `json:"qty_actual_kirim_source"`
	QtyActualKirimManual      int32                             `json:"qty_actual_kirim_manual"`
	QtyActualKirim            int32                             `json:"qty_actual_kirim"`
	TotalTerima               int32                             `json:"total_terima"`
	ConsActual                float64                           `json:"cons_actual"`
	Balance                   float64                           `json:"balance"`
	RejectQty                 int32                             `json:"reject_qty"`
	ReturQty                  int32                             `json:"retur_qty"`
	LastBalance               float64                           `json:"last_balance"`
	Keterangan                string                            `json:"keterangan"`
	IDMaterialListItem        *int32                            `json:"id_material_list_item,omitempty"`
	IDWoShell                 *int32                            `json:"id_wo_shell,omitempty"`
	IDWoTrim                  *int32                            `json:"id_wo_trim,omitempty"`
	TerimaEntries             []RekonsiliasiTerimaEntryResponse `json:"terima_entries"`
}

type RekonsiliasiDetailResponse struct {
	Header         RekonsiliasiHeaderResponse         `json:"header"`
	ColorSummaries []RekonsiliasiColorSummaryResponse `json:"color_summaries"`
	MaterialRows   []RekonsiliasiMaterialRowResponse  `json:"material_rows"`
}

type RekonsiliasiSuccessDoc struct {
	Status  string                     `json:"status" example:"success"`
	Message string                     `json:"message" example:"rekonsiliasi created"`
	Data    RekonsiliasiDetailResponse `json:"data"`
}

type RekonsiliasiListSuccessDoc struct {
	Status  string                   `json:"status" example:"success"`
	Message string                   `json:"message" example:"rekonsiliasi retrieved"`
	Data    RekonsiliasiListResponse `json:"data"`
}

type RekonsiliasiErrorDetail struct {
	Code string `json:"code" example:"rekonsiliasi_not_found"`
}

type RekonsiliasiErrorDoc struct {
	Status  string                  `json:"status" example:"error"`
	Message string                  `json:"message" example:"rekonsiliasi not found"`
	Error   RekonsiliasiErrorDetail `json:"error"`
}

type RekonsiliasiValidationErrorDoc struct {
	Status  string                         `json:"status" example:"error"`
	Message string                         `json:"message" example:"validation error"`
	Error   []response.ValidationErrorItem `json:"error"`
}
