package usecase

import (
	"context"
	"fmt"

	"permatatex-inventory/internal/entity"
	"permatatex-inventory/internal/model"
)

type DashboardUseCase struct {
	queries *entity.Queries
}

func NewDashboardUseCase(queries *entity.Queries) (*DashboardUseCase, error) {
	if queries == nil {
		return nil, fmt.Errorf("queries is required")
	}
	return &DashboardUseCase{queries: queries}, nil
}

// Logic untuk mengambil Logs dengan Paginasi
func (u *DashboardUseCase) GetLogs(ctx context.Context, filter model.ListLogsFilter) ([]model.AktivitasLogResponse, error) {
	arg := entity.GetAktivitasLogsParams{
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}

	logs, err := u.queries.GetAktivitasLogs(ctx, arg)
	if err != nil {
		return nil, err
	}

	var result []model.AktivitasLogResponse
	for _, l := range logs {
		// Konversi tipe data pointer dari sqlc ke DTO
		var detailNama, detailTable, detailDeskripsi *string
		if l.DetailNama.Valid {
			detailNama = &l.DetailNama.String
		}
		if l.DetailTable.Valid {
			detailTable = &l.DetailTable.String
		}
		if l.DetailDeskripsi.Valid {
			detailDeskripsi = &l.DetailDeskripsi.String
		}

		result = append(result, model.AktivitasLogResponse{
			IDLog:           l.IDLog,
			Aksi:            l.Aksi,
			Waktu:           l.Waktu.Time,
			DetailNama:      detailNama,
			DetailTable:     detailTable,
			DetailDeskripsi: detailDeskripsi,
		})
	}

	return result, nil
}

// Logic untuk AI Estimation menggunakan Regresi Linier Bawaan
func (u *DashboardUseCase) GetAIEstimation(ctx context.Context) (*model.AIEstimationResponse, error) {
	// 1. Tarik data historis yang sangat cepat via sqlc
	wos, err := u.queries.GetWorkOrderForAIEstimation(ctx)
	if err != nil {
		return nil, err
	}

	totalData := len(wos)
	if totalData == 0 {
		return &model.AIEstimationResponse{
			TotalDataHistoris: 0,
			BaseDurationDays:  0,
			DaysPerItem:       0,
			RumusPrediksi:     "Data historis tidak cukup untuk estimasi.",
		}, nil
	}

	// 2. Kalkulasi Regresi Linier: Y = a + bX
	// X = QTY barang, Y = Durasi pengerjaan (Hari)
	var sumX, sumY, sumXY, sumX2 float64
	n := float64(totalData)

	for _, wo := range wos {
		x := float64(wo.Qty)

		// Menghitung selisih hari antara Target Delivery dan Tanggal PR
		durationHours := wo.TargetDelivery.Time.Sub(wo.TanggalPr.Time).Hours()
		y := durationHours / 24.0

		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	// Mencegah division by zero jika data X (QTY) seragam semua
	denominator := (n * sumX2) - (sumX * sumX)
	var a, b float64
	if denominator != 0 {
		b = ((n * sumXY) - (sumX * sumY)) / denominator
		a = (sumY - (b * sumX)) / n
	}

	rumus := fmt.Sprintf("Estimasi Hari = %.2f + (%.4f * QTY)", a, b)

	return &model.AIEstimationResponse{
		TotalDataHistoris: totalData,
		BaseDurationDays:  a,
		DaysPerItem:       b,
		RumusPrediksi:     rumus,
	}, nil
}
