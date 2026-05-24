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

// DTO untuk endpoint AI Estimation
type AIEstimationResponse struct {
	TotalDataHistoris int     `json:"total_data_historis"`
	BaseDurationDays  float64 `json:"base_duration_days"` // Nilai 'a' (Intercept)
	DaysPerItem       float64 `json:"days_per_item"`      // Nilai 'b' (Slope)
	RumusPrediksi     string  `json:"rumus_prediksi"`
}

// ListLogsSuccessDoc merepresentasikan bentuk response sukses untuk list logs
type ListLogsSuccessDoc struct {
	Status  string                 `json:"status" example:"success"`
	Message string                 `json:"message" example:"Logs berhasil diambil"`
	Data    []AktivitasLogResponse `json:"data"`
}

// AIEstimationSuccessDoc merepresentasikan bentuk response sukses untuk AI estimation
type AIEstimationSuccessDoc struct {
	Status  string               `json:"status" example:"success"`
	Message string               `json:"message" example:"Estimasi AI berhasil dihitung"`
	Data    AIEstimationResponse `json:"data"`
}

// Tambahkan di internal/model/dashboard.go

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

type AIPredictionResponseData struct {
	EstimasiWaktuTotalHari   float64 `json:"estimasi_waktu_total_hari"`
	EstimasiTahapCuttingHari float64 `json:"estimasi_tahap_cutting_hari"`
	EstimasiTahapSewingHari  float64 `json:"estimasi_tahap_sewing_hari"`
	EstimasiTahapQCHari      float64 `json:"estimasi_tahap_qc_hari"`
}

type AIPredictionResponse struct {
	Status  string                   `json:"status"`
	Message string                   `json:"message"`
	Data    AIPredictionResponseData `json:"data"`
}
