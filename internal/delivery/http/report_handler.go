package httpdelivery

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"permatatex-inventory/internal/usecase"
	"permatatex-inventory/pkg/response"
)

type ReportHandler struct {
	useCase *usecase.ReportUseCase
}

func NewReportHandler(useCase *usecase.ReportUseCase) (*ReportHandler, error) {
	if useCase == nil {
		return nil, errors.New("report usecase is required")
	}
	return &ReportHandler{useCase: useCase}, nil
}

func (h *ReportHandler) RegisterRoutes(router gin.IRouter, authMiddleware gin.HandlerFunc) {
	v1 := router.Group("/api/v1").Use(authMiddleware, RequireInternalUser())
	{
		v1.GET("/reports/stock/category", RequirePermission(PermissionReportRead), h.GetStockReportPerKategori)
		v1.GET("/reports/stock/location", RequirePermission(PermissionReportRead), h.GetStockReportPerLokasi)
		v1.GET("/reports/movement", RequirePermission(PermissionReportRead), h.GetMovementReport)
	}
}

// GetStockReportPerKategori godoc
// @Summary      Laporan Stok per Kategori
// @Description  Mengambil laporan stok material yang teragregasi berdasarkan kategori barang garmen.
// @Tags         Reports
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  model.StockReportPerKategoriSuccessDoc
// @Failure      401  {object}  model.TransactionErrorDoc
// @Failure      403  {object}  model.TransactionErrorDoc
// @Failure      500  {object}  model.TransactionErrorDoc
// @Router       /api/v1/reports/stock/category [get]
func (h *ReportHandler) GetStockReportPerKategori(c *gin.Context) {
	result, err := h.useCase.GetStockReportPerKategori(c.Request.Context())
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "Gagal mengambil laporan stok per kategori", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "stock report per kategori retrieved", result)
}

// GetStockReportPerLokasi godoc
// @Summary      Laporan Stok per Lokasi
// @Description  Mengambil laporan stok material yang teragregasi berdasarkan lokasi penyimpanan rak.
// @Tags         Reports
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  model.StockReportPerLokasiSuccessDoc
// @Failure      401  {object}  model.TransactionErrorDoc
// @Failure      403  {object}  model.TransactionErrorDoc
// @Failure      500  {object}  model.TransactionErrorDoc
// @Router       /api/v1/reports/stock/location [get]
func (h *ReportHandler) GetStockReportPerLokasi(c *gin.Context) {
	result, err := h.useCase.GetStockReportPerLokasi(c.Request.Context())
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "Gagal mengambil laporan stok per lokasi", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "stock report per lokasi retrieved", result)
}

// GetMovementReport godoc
// @Summary      Laporan Riwayat Pergerakan Barang
// @Description  Mengambil laporan kronologis pergerakan stok masuk (received) dan keluar (surat jalan).
// @Tags         Reports
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  model.MovementReportSuccessDoc
// @Failure      401  {object}  model.TransactionErrorDoc
// @Failure      403  {object}  model.TransactionErrorDoc
// @Failure      500  {object}  model.TransactionErrorDoc
// @Router       /api/v1/reports/movement [get]
func (h *ReportHandler) GetMovementReport(c *gin.Context) {
	result, err := h.useCase.GetMovementReport(c.Request.Context())
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "Gagal mengambil laporan pergerakan barang", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "movement report retrieved", result)
}
