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
	if req.QtyS > 0 {
		jumlahSize++
	}
	if req.QtyM > 0 {
		jumlahSize++
	}
	if req.QtyL > 0 {
		jumlahSize++
	}
	if req.QtyXL > 0 {
		jumlahSize++
	}
	if req.QtyXXL > 0 {
		jumlahSize++
	}

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

// GetOperatorDashboardMetrics mengambil data real-time untuk dashboard operator
func (u *DashboardUseCase) GetOperatorDashboardMetrics(ctx context.Context) (*model.OperatorDashboardMetrics, error) {
	activeWO, err := u.queries.GetOperatorActiveWorkOrdersCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get active WO: %w", err)
	}

	targetPcs, err := u.queries.GetOperatorTargetProduksiHariIni(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get target produksi: %w", err)
	}

	outputToday, err := u.queries.GetOperatorOutputHariIni(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get output hari ini: %w", err)
	}

	ongoingWOs, err := u.queries.GetOperatorOngoingWorkOrders(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get ongoing WOs: %w", err)
	}

	var ongoingData []model.OngoingWorkOrder
	for _, wo := range ongoingWOs {
		ongoingData = append(ongoingData, model.OngoingWorkOrder{
			IDWO:        wo.IDWo,
			Buyer:       wo.Buyer,
			Model:       wo.Model,
			Qty:         wo.Qty,
			TotalOutput: wo.TotalOutput,
		})
	}

	// Hardcode rasio reject untuk saat ini karena tabel QC Finish belum melacak cacat
	rasioReject := 0.0

	return &model.OperatorDashboardMetrics{
		ActiveWorkOrders:  activeWO,
		TargetProduksiPcs: targetPcs,
		OutputHariIni:     outputToday,
		RasioReject:       rasioReject,
		OngoingWorkOrders: ongoingData,
	}, nil
}
