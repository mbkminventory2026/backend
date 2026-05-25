package usecase

import (
	"context"
	"fmt"

	"permatatex-inventory/internal/entity"
	"permatatex-inventory/internal/gateway/ai"
	"permatatex-inventory/internal/model"
)

type DashboardUseCase struct {
	queries   *entity.Queries
	aiGateway *ai.Gateway
}

// Hanya ada SATU NewDashboardUseCase di sini (Gabungan Queries & AI Gateway)
func NewDashboardUseCase(queries *entity.Queries, aiGateway *ai.Gateway) (*DashboardUseCase, error) {
	if queries == nil {
		return nil, fmt.Errorf("queries is required")
	}
	return &DashboardUseCase{
		queries:   queries,
		aiGateway: aiGateway,
	}, nil
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

// PredictNewOrder melakukan kalkulasi rasio otomatis lalu menembak Python
func (u *DashboardUseCase) PredictNewOrder(ctx context.Context, req model.AIEstimationRequest) (*model.AIPredictionResponseData, error) {
	// 1. Kalkulasi Qty Total
	totalQty := req.QtyS + req.QtyM + req.QtyL + req.QtyXL + req.QtyXXL

	// 2. Kalkulasi Jumlah Size (Berapa banyak size yang jumlahnya > 0)
	var jumlahSize float64
	if req.QtyS > 0 { jumlahSize++ }
	if req.QtyM > 0 { jumlahSize++ }
	if req.QtyL > 0 { jumlahSize++ }
	if req.QtyXL > 0 { jumlahSize++ }
	if req.QtyXXL > 0 { jumlahSize++ }

	// 3. Kalkulasi Rasio (Cegah pembagian dengan nol)
	var rasioS, rasioM, rasioL, rasioXL, rasioXXL float64
	if totalQty > 0 {
		rasioS = req.QtyS / totalQty
		rasioM = req.QtyM / totalQty
		rasioL = req.QtyL / totalQty
		rasioXL = req.QtyXL / totalQty
		rasioXXL = req.QtyXXL / totalQty
	}

	// 4. Susun payload lengkap untuk dikirim ke Microservice Python
	aiReq := model.AIPredictionRequest{
		QtyS:               req.QtyS,
		QtyM:               req.QtyM,
		QtyL:               req.QtyL,
		QtyXL:              req.QtyXL,
		QtyXXL:             req.QtyXXL,
		QtyTotal:           totalQty,
		JumlahSize:         jumlahSize,
		RasioS:             rasioS,
		RasioM:             rasioM,
		RasioL:             rasioL,
		RasioXL:            rasioXL,
		RasioXXL:           rasioXXL,
		Jenis:              req.Jenis,
		MenWomen:           req.MenWomen,
		Panjang01:          req.Panjang01,
		Embro:              req.Embro,
		Furing:             req.Furing,
		CuttingInHouse:     req.CuttingInHouse,
		KonsumsiKainPerPcs: req.KonsumsiKainPerPcs,
		JenisKain:          req.JenisKain,
	}

	// 5. Eksekusi ke Python via Gateway
	return u.aiGateway.PredictSchedule(ctx, aiReq)
}