package httpdelivery

import (
	"errors"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"permatatex-inventory/internal/model"
	"permatatex-inventory/internal/usecase"
	"permatatex-inventory/pkg/response"
)

// Upgrader untuk mengubah HTTP menjadi WebSocket
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Di production, ganti dengan domain frontend kamu
	},
}

type DashboardHandler struct {
	useCase *usecase.DashboardUseCase
	clients map[*websocket.Conn]bool // Menyimpan daftar browser yang aktif membuka dashboard
	mu      sync.Mutex               // Menghindari race condition saat broadcast
}

func NewDashboardHandler(useCase *usecase.DashboardUseCase) (*DashboardHandler, error) {
	if useCase == nil {
		return nil, errors.New("dashboard usecase is required")
	}

	return &DashboardHandler{
		useCase: useCase,
		clients: make(map[*websocket.Conn]bool),
	}, nil
}

func (h *DashboardHandler) RegisterRoutes(router *gin.Engine, authMiddleware gin.HandlerFunc) {
	api := router.Group("/api/v1")
	api.Use(authMiddleware)
	{
		api.GET("/logs", h.GetLogs)
		api.POST("/dashboard/ai-estimation", h.PredictAIEstimation)
	}

	// Daftarkan rute WebSocket di sini (tanpa middleware HTTP biasa jika tidak diperlukan)
	router.GET("/ws/alerts", h.Alerts)
}

// GetLogs godoc
// @Summary      List Aktivitas Logs
// @Description  Mengambil log aktivitas sistem dengan paginasi.
// @Tags         Dashboard
// @Produce      json
// @Security     BearerAuth
// @Param        limit   query     int  false  "Limit (default 20)"
// @Param        offset  query     int  false  "Offset (default 0)"
// @Success      200     {object}  model.ListLogsSuccessDoc
// @Router       /api/v1/logs [get]
func (h *DashboardHandler) GetLogs(c *gin.Context) {
	var filter model.ListLogsFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		response.Fail(c, http.StatusBadRequest, "Parameter tidak valid", err.Error())
		return
	}

	result, err := h.useCase.GetLogs(c.Request.Context(), filter)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "Gagal mengambil log", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Logs berhasil diambil", result)
}

// PredictAIEstimation menangkap data pesanan baru dari Frontend
// @Summary AI Delivery Date Estimation untuk Order Baru
// @Description Memprediksi jadwal menggunakan model TabPFN via Python Service
// @Tags Dashboard
// @Accept json
// @Produce json
// @Param request body model.AIEstimationRequest true "Data Pesanan Mentah"
// @Success 200 {object} model.AIEstimationSuccessDoc
// @Security BearerAuth
// @Router /api/v1/dashboard/ai-estimation [post]
func (h *DashboardHandler) PredictAIEstimation(c *gin.Context) {
	var req model.AIEstimationRequest

	// 1. Parsing payload JSON dari Frontend
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Format request JSON tidak valid", err.Error())
		return
	}

	// 2. Panggil Use Case yang baru (yang akan menghitung rasio dan memanggil Python)
	result, err := h.useCase.PredictNewOrder(c.Request.Context(), req)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "Gagal memproses prediksi AI", err.Error())
		return
	}

	// 3. Kembalikan respons sukses ke Frontend
	response.Success(c, http.StatusOK, "Estimasi AI berhasil dihitung", result)
}

// Alerts menangani pendaftaran koneksi WebSocket
func (h *DashboardHandler) Alerts(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// Daftarkan client baru
	h.mu.Lock()
	h.clients[conn] = true
	h.mu.Unlock()

	// Pastikan dihapus jika browser ditutup
	defer func() {
		h.mu.Lock()
		delete(h.clients, conn)
		h.mu.Unlock()
	}()

	// Jaga koneksi tetap terbuka
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}

// BroadcastLowStockAlert Fungsi bantuan untuk dikirim dari mana saja saat stok tipis
func (h *DashboardHandler) BroadcastLowStockAlert(data interface{}) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for client := range h.clients {
		err := client.WriteJSON(data)
		if err != nil {
			client.Close()
			delete(h.clients, client)
		}
	}
}
