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

// GetFinanceDashboardMetrics mengambil data untuk dashboard Admin Keuangan
func (u *DashboardUseCase) GetFinanceDashboardMetrics(ctx context.Context) (*model.FinanceDashboardMetrics, error) {
	poClientCount, err := u.queries.GetFinanceTotalPOClientThisMonth(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get po client count: %w", err)
	}

	poInternalCount, err := u.queries.GetFinanceTotalPOInternalThisMonth(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get po internal count: %w", err)
	}

	prInternalCount, err := u.queries.GetFinanceTotalPRInternalThisMonth(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get pr internal count: %w", err)
	}

	recentPOClients, err := u.queries.GetFinanceRecentPOClients(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent po clients: %w", err)
	}

	var parsedRecentPOClients []model.RecentPOClient
	for _, pc := range recentPOClients {
		parsedRecentPOClients = append(parsedRecentPOClients, model.RecentPOClient{
			IDPoClient: pc.IDPoClient,
			PoNumber:   pc.PoNumber,
			Tanggal:    pc.Tanggal.Time.Format("2006-01-02"),
			MitraName:  pc.MitraName,
		})
	}

	recentPOInternals, err := u.queries.GetFinanceRecentPOInternals(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent po internals: %w", err)
	}

	var parsedRecentPOInternals []model.RecentPOInternal
	for _, poi := range recentPOInternals {
		parsedRecentPOInternals = append(parsedRecentPOInternals, model.RecentPOInternal{
			IDPoInternal: poi.IDPoInternal,
			NamaPo:       poi.NamaPo,
			Tanggal:      poi.Tanggal.Time.Format("2006-01-02"),
			SupplierName: poi.SupplierName,
		})
	}

	return &model.FinanceDashboardMetrics{
		TotalPOClientThisMonth:   poClientCount,
		TotalPOInternalThisMonth: poInternalCount,
		TotalPRInternalThisMonth: prInternalCount,
		RecentPOClients:          parsedRecentPOClients,
		RecentPOInternals:        parsedRecentPOInternals,
	}, nil
}

// GetProductionDashboardMetrics mengambil data untuk dashboard Admin Produksi
func (u *DashboardUseCase) GetProductionDashboardMetrics(ctx context.Context) (*model.ProductionDashboardMetrics, error) {
	// Reusing GetOperatorTargetProduksiHariIni for target produksi
	targetPcs, err := u.queries.GetOperatorTargetProduksiHariIni(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get target produksi: %w", err)
	}

	tlCount, err := u.queries.GetProductionTotalTimelineThisMonth(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get timeline count: %w", err)
	}

	mpCount, err := u.queries.GetProductionTotalMarkerPlanThisMonth(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get marker plan count: %w", err)
	}

	scpCount, err := u.queries.GetProductionTotalSpreadingCuttingPlanThisMonth(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get scp count: %w", err)
	}

	recentTimelines, err := u.queries.GetProductionRecentTimelines(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent timelines: %w", err)
	}

	var parsedTimelines []model.RecentTimeline
	for _, tl := range recentTimelines {
		parsedTimelines = append(parsedTimelines, model.RecentTimeline{
			IDTimeline:     tl.IDTimeline,
			TanggalDisusun: tl.TanggalDisusun.Time.Format("2006-01-02"),
			Notes:          tl.Notes,
			PoNumber:       tl.PoNumber,
		})
	}

	recentMPs, err := u.queries.GetProductionRecentMarkerPlans(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent marker plans: %w", err)
	}

	var parsedMPs []model.RecentMarkerPlan
	for _, mp := range recentMPs {
		parsedMPs = append(parsedMPs, model.RecentMarkerPlan{
			IDMarkerPlan:   mp.IDMarkerPlan,
			NoDokumen:      mp.NoDokumen,
			TanggalEfektif: mp.TanggalEfektif.Time.Format("2006-01-02"),
			Color:          mp.Color,
			Model:          mp.Model,
		})
	}

	recentSCPs, err := u.queries.GetProductionRecentSpreadingCuttingPlans(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent scps: %w", err)
	}

	var parsedSCPs []model.RecentSpreadingCuttingPlan
	for _, scp := range recentSCPs {
		parsedSCPs = append(parsedSCPs, model.RecentSpreadingCuttingPlan{
			IDSpreadingCuttingPlan: scp.IDSpreadingCuttingPlan,
			NoDokumen:              scp.NoDokumen,
			TanggalEfektif:         scp.TanggalEfektif.Time.Format("2006-01-02"),
			Model:                  scp.Model,
		})
	}

	return &model.ProductionDashboardMetrics{
		TargetProduksiPcs:                  targetPcs,
		TotalTimelineThisMonth:             tlCount,
		TotalMarkerPlanThisMonth:           mpCount,
		TotalSpreadingCuttingPlanThisMonth: scpCount,
		RecentTimelines:                    parsedTimelines,
		RecentMarkerPlans:                  parsedMPs,
		RecentSpreadingCuttingPlans:        parsedSCPs,
	}, nil
}

// GetWarehouseDashboardMetrics mengambil data untuk dashboard Admin Gudang
func (u *DashboardUseCase) GetWarehouseDashboardMetrics(ctx context.Context) (*model.WarehouseDashboardMetrics, error) {
	totalItems, err := u.queries.GetWarehouseTotalItems(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get total items: %w", err)
	}

	sjClientCount, err := u.queries.GetWarehouseTotalSuratJalanClientThisMonth(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get surat jalan client count: %w", err)
	}

	sjInternalCount, err := u.queries.GetWarehouseTotalSuratJalanInternalThisMonth(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get surat jalan internal count: %w", err)
	}

	lowStocks, err := u.queries.GetLowStockAlerts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get low stock alerts: %w", err)
	}

	var parsedLowStocks []model.LowStockAlert
	for _, ls := range lowStocks {
		parsedLowStocks = append(parsedLowStocks, model.LowStockAlert{
			IDRekonsiliasiMaterial: ls.IDRekonsiliasiMaterial,
			Description:            ls.Description,
			Size:                   ls.Size,
			Balance:                ls.Balance,
			LastBalance:            ls.LastBalance,
			Satuan:                 ls.Satuan,
			MinStock:               ls.MinStock,
		})
	}

	recentSJC, err := u.queries.GetWarehouseRecentSuratJalanClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent surat jalan client: %w", err)
	}

	var parsedSJC []model.RecentWarehouseSuratJalanClient
	for _, sj := range recentSJC {
		parsedSJC = append(parsedSJC, model.RecentWarehouseSuratJalanClient{
			IDSuratJalanClient:  sj.IDSuratJalanClient,
			Tanggal:             sj.Tanggal.Time.Format("2006-01-02"),
			Keterangan:          sj.Keterangan,
			MaterialDescription: sj.MaterialDescription,
		})
	}

	recentSJI, err := u.queries.GetWarehouseRecentSuratJalanInternal(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent surat jalan internal: %w", err)
	}

	var parsedSJI []model.RecentWarehouseSuratJalanInternal
	for _, sj := range recentSJI {
		parsedSJI = append(parsedSJI, model.RecentWarehouseSuratJalanInternal{
			IDSuratJalanInternal: sj.IDSuratJalanInternal,
			CreatedAt:            sj.CreatedAt.Time.Format("2006-01-02"),
		})
	}

	recentBarang, err := u.queries.GetWarehouseRecentBarang(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent barang: %w", err)
	}

	var parsedBarang []model.RecentWarehouseBarang
	for _, b := range recentBarang {
		parsedBarang = append(parsedBarang, model.RecentWarehouseBarang{
			IDBarang:    b.IDBarang,
			NamaBarang:  b.NamaBarang,
			Kode:        b.Kode,
			StokMinimum: b.StokMinimum,
			CreatedAt:   b.CreatedAt.Time.Format("2006-01-02"),
		})
	}

	return &model.WarehouseDashboardMetrics{
		TotalItems:                       totalItems,
		TotalSuratJalanClientThisMonth:   sjClientCount,
		TotalSuratJalanInternalThisMonth: sjInternalCount,
		LowStockAlertsCount:              int64(len(parsedLowStocks)),
		RecentSuratJalanClients:          parsedSJC,
		RecentSuratJalanInternals:        parsedSJI,
		RecentBarangs:                    parsedBarang,
		LowStockAlerts:                   parsedLowStocks,
	}, nil
}
