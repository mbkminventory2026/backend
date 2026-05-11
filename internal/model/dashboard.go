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
