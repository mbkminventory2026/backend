package model

// --- REQUESTS ---

// --- RESPONSES ---

type StockReportPerKategoriResponse struct {
	Kategori   string `json:"kategori"`
	NamaBarang string `json:"nama_barang"`
	Size       string `json:"size"`
	TotalStok  int64  `json:"total_stok"`
	Satuan     string `json:"satuan"`
}

type StockReportPerLokasiResponse struct {
	LokasiRak  string `json:"lokasi_rak"`
	NamaBarang string `json:"nama_barang"`
	Size       string `json:"size"`
	TotalStok  int64  `json:"total_stok"`
	Satuan     string `json:"satuan"`
}

type MovementReportResponse struct {
	Tipe           string `json:"tipe"` // "IN" atau "OUT"
	Tanggal        string `json:"tanggal"`
	Qty            int32  `json:"qty"`
	Keterangan     string `json:"keterangan"`
	NamaMaterial   string `json:"nama_material"`
	Size           string `json:"size"`
	Uom            string `json:"uom"`
	WorkOrderModel string `json:"work_order_model"`
}

// --- SWAGGER SUCCESS DOCS ---

type StockReportPerKategoriSuccessDoc struct {
	Status  string                           `json:"status" example:"success"`
	Message string                           `json:"message" example:"stock report per kategori retrieved"`
	Data    []StockReportPerKategoriResponse `json:"data"`
}

type StockReportPerLokasiSuccessDoc struct {
	Status  string                         `json:"status" example:"success"`
	Message string                         `json:"message" example:"stock report per lokasi retrieved"`
	Data    []StockReportPerLokasiResponse `json:"data"`
}

type MovementReportSuccessDoc struct {
	Status  string                   `json:"status" example:"success"`
	Message string                   `json:"message" example:"movement report retrieved"`
	Data    []MovementReportResponse `json:"data"`
}
