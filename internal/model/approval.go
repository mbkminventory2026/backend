package model

import "time"

type ApprovalPendingResponse struct {
	IDIDDetail       int32     `json:"id_otoritas_detail"`
	IDHeader         int32     `json:"id_otoritas"`
	NamaTabelDokumen string    `json:"nama_tabel_dokumen"`
	IDDokumen        int32     `json:"id_dokumen"`
	TipePeran        string    `json:"tipe_peran"`
	DocSummary       string    `json:"doc_summary,omitempty"`
	RequestedBy      string    `json:"requested_by,omitempty"`
	RequestedAt      time.Time `json:"requested_at"`
}

type ApprovalActionRequest struct {
	IDDetail int32  `json:"id_otoritas_detail" binding:"required"`
	Action   string `json:"action" binding:"required,oneof=approve reject"`
	Catatan  string `json:"catatan"`
}

type AuditTrailStep struct {
	IDDetail  int32      `json:"id_otoritas_detail"`
	IDUser    int32      `json:"id_user"`
	NamaUser  string     `json:"nama_user"`
	TipePeran string     `json:"tipe_peran"`
	Done      bool       `json:"done"`
	WaktuAksi *time.Time `json:"waktu_aksi,omitempty"`
	Catatan   string     `json:"catatan,omitempty"`
}

type DocumentAuditTrailResponse struct {
	IDHeader     int32            `json:"id_otoritas"`
	NamaTabel    string           `json:"nama_tabel"`
	IDDokumen    int32            `json:"id_dokumen"`
	StatusGlobal string           `json:"status_global"`
	Steps        []AuditTrailStep `json:"steps"`
}

type ApprovalHistoryListItem struct {
	IDHeader         int32     `json:"id_otoritas"`
	NamaTabelDokumen string    `json:"nama_tabel_dokumen"`
	IDDokumen        int32     `json:"id_dokumen"`
	StatusGlobal     string    `json:"status_global"`
	DocSummary       string    `json:"doc_summary,omitempty"`
	RequestedBy      string    `json:"requested_by,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

type ApprovalHistoryResponse struct {
	Items      []ApprovalHistoryListItem `json:"items"`
	TotalItems int64                     `json:"total_items"`
}
