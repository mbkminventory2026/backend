package ai_test

import (
	"context"
	"testing"

	"permatatex-inventory/internal/gateway/ai"
	"permatatex-inventory/internal/model"
)

func TestPredictSchedule_Integration(t *testing.T) {
	// Arahkan ke server FastAPI lokal
	gateway := ai.NewGateway("http://127.0.0.1:8000")

	// Data tiruan yang sama persis dengan contoh Python
	reqData := model.AIPredictionRequest{
		QtyS:               180.0,
		QtyM:               540.0,
		QtyL:               540.0,
		QtyXL:              540.0,
		QtyXXL:             360.0,
		QtyTotal:           2160.0,
		JumlahSize:         5.0,
		RasioS:             0.08,
		RasioM:             0.25,
		RasioL:             0.25,
		RasioXL:            0.25,
		RasioXXL:           0.17,
		Jenis:              1.0,
		MenWomen:           1.0,
		Panjang01:          0.0,
		Embro:              1.0,
		Furing:             0.0,
		CuttingInHouse:     1.0,
		KonsumsiKainPerPcs: 1.28,
		JenisKain:          2.0,
	}

	// Eksekusi pemanggilan API ke Python
	resp, err := gateway.PredictSchedule(context.Background(), reqData)

	if err != nil {
		t.Fatalf("Gagal menghubungi AI Service: %v", err)
	}

	if resp == nil {
		t.Fatalf("Response kosong")
	}

	// Tampilkan hasil di terminal test
	t.Logf("ESTIMASI JADWAL PRODUKSI DARI PYTHON:")
	t.Logf("Waktu Total: %.1f Hari", resp.EstimasiWaktuTotalHari)
	t.Logf("Cutting: %.1f Hari", resp.EstimasiTahapCuttingHari)
	t.Logf("Sewing: %.1f Hari", resp.EstimasiTahapSewingHari)
	t.Logf("QC: %.1f Hari", resp.EstimasiTahapQCHari)
}
