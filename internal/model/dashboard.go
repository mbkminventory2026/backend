package model

import "time"

// DTO untuk endpoint Logs
type AktivitasLogResponse struct {
	IDLog           int32     `json:"id_log"`
	Aksi            string    `json:"aksi"`
	Waktu           time.Time `json:"waktu"`
	DetailNama      *string   `json:"detail_nama,omitempty"`
	DetailTable     *string   `json:"detail_table,omitempty"`
	DetailDeskripsi *string   `json:"detail_deskripsi,omitempty"`
}

type ListLogsFilter struct {
	Limit  int32 `form:"limit" binding:"gte=1"`
	Offset int32 `form:"offset" binding:"gte=0"`
}

// ListLogsSuccessDoc merepresentasikan bentuk response sukses untuk list logs
type ListLogsSuccessDoc struct {
	Status  string                 `json:"status" example:"success"`
	Message string                 `json:"message" example:"Logs berhasil diambil"`
	Data    []AktivitasLogResponse `json:"data"`
}

// AIEstimationSuccessDoc merepresentasikan bentuk response sukses untuk AI estimation di Swagger
type AIEstimationSuccessDoc struct {
	Status  string                   `json:"status" example:"success"`
	Message string                   `json:"message" example:"Estimasi AI berhasil dihitung"`
	Data    AIPredictionResponseData `json:"data"` // <-- Sekarang merujuk ke data hasil Python!
}

// AIPredictionRequest adalah DTO yang dikirim dari Golang ke Python
type AIPredictionRequest struct {
	QtyS               float64 `json:"qty_s"`
	QtyM               float64 `json:"qty_m"`
	QtyL               float64 `json:"qty_l"`
	QtyXL              float64 `json:"qty_xl"`
	QtyXXL             float64 `json:"qty_xxl"`
	QtyTotal           float64 `json:"qty_total"`
	JumlahSize         float64 `json:"jumlah_size"`
	RasioS             float64 `json:"rasio_s"`
	RasioM             float64 `json:"rasio_m"`
	RasioL             float64 `json:"rasio_l"`
	RasioXL            float64 `json:"rasio_xl"`
	RasioXXL           float64 `json:"rasio_xxl"`
	Jenis              float64 `json:"jenis"`
	MenWomen           float64 `json:"men_women"`
	Panjang01          float64 `json:"panjang_01"`
	Embro              float64 `json:"embro"`
	Furing             float64 `json:"furing"`
	CuttingInHouse     float64 `json:"cutting_in_house"`
	KonsumsiKainPerPcs float64 `json:"konsumsi_kain_per_pcs"`
	JenisKain          float64 `json:"jenis_kain"`
}

// AIPredictionResponseData adalah struktur data kembalian dari Python
type AIPredictionResponseData struct {
	EstimasiWaktuTotalHari   float64 `json:"estimasi_waktu_total_hari"`
	EstimasiTahapCuttingHari float64 `json:"estimasi_tahap_cutting_hari"`
	EstimasiTahapSewingHari  float64 `json:"estimasi_tahap_sewing_hari"`
	EstimasiTahapQCHari      float64 `json:"estimasi_tahap_qc_hari"`
}

// AIPredictionResponse adalah struktur utuh dari API Python
type AIPredictionResponse struct {
	Status  string                   `json:"status"`
	Message string                   `json:"message"`
	Data    AIPredictionResponseData `json:"data"`
}

// AIEstimationRequest adalah payload yang dikirim oleh Frontend ke Golang
type AIEstimationRequest struct {
	QtyS               float64 `json:"qty_s" binding:"min=0"`
	QtyM               float64 `json:"qty_m" binding:"min=0"`
	QtyL               float64 `json:"qty_l" binding:"min=0"`
	QtyXL              float64 `json:"qty_xl" binding:"min=0"`
	QtyXXL             float64 `json:"qty_xxl" binding:"min=0"`
	Jenis              float64 `json:"jenis"`
	MenWomen           float64 `json:"men_women"`
	Panjang01          float64 `json:"panjang_01"`
	Embro              float64 `json:"embro"`
	Furing             float64 `json:"furing"`
	CuttingInHouse     float64 `json:"cutting_in_house"`
	KonsumsiKainPerPcs float64 `json:"konsumsi_kain_per_pcs"`
	JenisKain          float64 `json:"jenis_kain"`
}

// OngoingWorkOrder representasi progress WO berjalan
type OngoingWorkOrder struct {
	IDWO        int32  `json:"id_wo"`
	Buyer       string `json:"buyer"`
	Model       string `json:"model"`
	Qty         int32  `json:"qty"`
	TotalOutput int32  `json:"total_output"`
}

// AdminSistemDashboardMetrics representasi seluruh metrik untuk Admin Sistem
type AdminSistemDashboardMetrics struct {
	ActiveWorkOrders  int64              `json:"active_work_orders"`
	TargetProduksiPcs int32              `json:"target_produksi_pcs"`
	OutputHariIni     int32              `json:"output_hari_ini"`
	RasioReject       float64            `json:"rasio_reject"`
	OngoingWorkOrders []OngoingWorkOrder `json:"ongoing_work_orders"`
}

// RecentPOClient representasi untuk daftar recent PO Client
type RecentPOClient struct {
	IDPoClient int32  `json:"id_po_client"`
	PoNumber   string `json:"po_number"`
	Tanggal    string `json:"tanggal"`
	MitraName  string `json:"mitra_name"`
}

// RecentPOInternal representasi untuk daftar recent PO Internal
type RecentPOInternal struct {
	IDPoInternal int32  `json:"id_po_internal"`
	NamaPo       string `json:"nama_po"`
	Tanggal      string `json:"tanggal"`
	SupplierName string `json:"supplier_name"`
}

// FinanceDashboardMetrics representasi seluruh metrik untuk Admin Keuangan
type FinanceDashboardMetrics struct {
	TotalPOClientThisMonth   int64              `json:"total_po_client_this_month"`
	TotalPOInternalThisMonth int64              `json:"total_po_internal_this_month"`
	TotalPRInternalThisMonth int64              `json:"total_pr_internal_this_month"`
	RecentPOClients          []RecentPOClient   `json:"recent_po_clients"`
	RecentPOInternals        []RecentPOInternal `json:"recent_po_internals"`
}

type RecentTimeline struct {
	IDTimeline     int32  `json:"id_timeline"`
	TanggalDisusun string `json:"tanggal_disusun"`
	Notes          string `json:"notes"`
	PoNumber       string `json:"po_number"`
}

type RecentMarkerPlan struct {
	IDMarkerPlan   int32  `json:"id_marker_plan"`
	NoDokumen      string `json:"no_dokumen"`
	TanggalEfektif string `json:"tanggal_efektif"`
	Color          string `json:"color"`
	Model          string `json:"model"`
}

type RecentSpreadingCuttingPlan struct {
	IDSpreadingCuttingPlan int32  `json:"id_spreading_cutting_plan"`
	NoDokumen              string `json:"no_dokumen"`
	TanggalEfektif         string `json:"tanggal_efektif"`
	Model                  string `json:"model"`
}

type ProductionDashboardMetrics struct {
	TargetProduksiPcs                  int32                        `json:"target_produksi_pcs"`
	TotalTimelineThisMonth             int64                        `json:"total_timeline_this_month"`
	TotalMarkerPlanThisMonth           int64                        `json:"total_marker_plan_this_month"`
	TotalSpreadingCuttingPlanThisMonth int64                        `json:"total_spreading_cutting_plan_this_month"`
	RecentTimelines                    []RecentTimeline             `json:"recent_timelines"`
	RecentMarkerPlans                  []RecentMarkerPlan           `json:"recent_marker_plans"`
	RecentSpreadingCuttingPlans        []RecentSpreadingCuttingPlan `json:"recent_spreading_cutting_plans"`
}

type RecentWarehouseSuratJalanClient struct {
	IDSuratJalanClient  int32  `json:"id_surat_jalan_client"`
	Tanggal             string `json:"tanggal"`
	Keterangan          string `json:"keterangan"`
	MaterialDescription string `json:"material_description"`
}

type RecentWarehouseSuratJalanInternal struct {
	IDSuratJalanInternal int32  `json:"id_surat_jalan_internal"`
	CreatedAt            string `json:"created_at"`
}

type RecentWarehouseBarang struct {
	IDBarang    int32  `json:"id_barang"`
	NamaBarang  string `json:"nama_barang"`
	Kode        string `json:"kode"`
	StokMinimum int32  `json:"stok_minimum"`
	CreatedAt   string `json:"created_at"`
}

type LowStockAlert struct {
	IDRekonsiliasiMaterial int32  `json:"id_rekonsiliasi_material"`
	Description            string `json:"description"`
	Size                   string `json:"size"`
	Balance                int32  `json:"balance"`
	LastBalance            int32  `json:"last_balance"`
	Satuan                 string `json:"satuan"`
	MinStock               int32  `json:"min_stock"`
}

type WarehouseDashboardMetrics struct {
	TotalItems                       int64                               `json:"total_items"`
	TotalSuratJalanClientThisMonth   int64                               `json:"total_surat_jalan_client_this_month"`
	TotalSuratJalanInternalThisMonth int64                               `json:"total_surat_jalan_internal_this_month"`
	LowStockAlertsCount              int64                               `json:"low_stock_alerts_count"`
	RecentSuratJalanClients          []RecentWarehouseSuratJalanClient   `json:"recent_surat_jalan_clients"`
	RecentSuratJalanInternals        []RecentWarehouseSuratJalanInternal `json:"recent_surat_jalan_internals"`
	RecentBarangs                    []RecentWarehouseBarang             `json:"recent_barangs"`
	LowStockAlerts                   []LowStockAlert                     `json:"low_stock_alerts"`
}
