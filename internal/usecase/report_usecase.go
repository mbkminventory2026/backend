package usecase

import (
	"context"
	"errors"
	"time"

	"permatatex-inventory/internal/entity"
	"permatatex-inventory/internal/model"
)

type ReportUseCase struct {
	repo entity.Querier
}

func NewReportUseCase(repo entity.Querier) (*ReportUseCase, error) {
	if repo == nil {
		return nil, errors.New("report repository is required")
	}
	return &ReportUseCase{repo: repo}, nil
}

func (u *ReportUseCase) GetStockReportPerKategori(ctx context.Context) ([]model.StockReportPerKategoriResponse, error) {
	rows, err := u.repo.GetStockReportPerKategori(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]model.StockReportPerKategoriResponse, 0, len(rows))
	for _, row := range rows {
		result = append(result, model.StockReportPerKategoriResponse{
			Kategori:   row.Kategori,
			NamaBarang: row.NamaBarang,
			Size:       row.Size,
			TotalStok:  row.TotalStok,
			Satuan:     row.Satuan,
		})
	}

	return result, nil
}

func (u *ReportUseCase) GetStockReportPerLokasi(ctx context.Context) ([]model.StockReportPerLokasiResponse, error) {
	rows, err := u.repo.GetStockReportPerLokasi(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]model.StockReportPerLokasiResponse, 0, len(rows))
	for _, row := range rows {
		result = append(result, model.StockReportPerLokasiResponse{
			LokasiRak:  row.LokasiRak,
			NamaBarang: row.NamaBarang,
			Size:       row.Size,
			TotalStok:  row.TotalStok,
			Satuan:     row.Satuan,
		})
	}

	return result, nil
}

func (u *ReportUseCase) GetMovementReport(ctx context.Context) ([]model.MovementReportResponse, error) {
	rows, err := u.repo.GetMovementReport(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]model.MovementReportResponse, 0, len(rows))
	for _, row := range rows {
		var tanggalStr string
		if row.Tanggal.Valid {
			tanggalStr = row.Tanggal.Time.Format("2006-01-02")
		} else {
			tanggalStr = time.Now().Format("2006-01-02")
		}

		result = append(result, model.MovementReportResponse{
			Tipe:           row.Tipe,
			Tanggal:        tanggalStr,
			Qty:            row.Qty,
			Keterangan:     row.Keterangan,
			NamaMaterial:   row.NamaMaterial,
			Uom:            row.Uom,
			WorkOrderModel: row.WorkOrderModel,
		})
	}

	return result, nil
}
