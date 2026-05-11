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

func (h *DashboardHandler) RegisterRoutes(router gin.IRouter, authMiddleware gin.HandlerFunc) {
	// Endpoint API standard dengan proteksi Auth
	api := router.Group("/api/v1").Use(authMiddleware)
	{
		api.GET("/logs", h.GetLogs)
		api.GET("/dashboard/ai-estimation", h.GetAIEstimation)
	}

	// Endpoint WebSocket (URL: ws://localhost:8080/ws/alerts)
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

// GetAIEstimation godoc
// @Summary      AI Delivery Date Estimation
// @Description  Mendapatkan rumus regresi linier berdasarkan data historis produksi.
// @Tags         Dashboard
// @Produce      json
// @Security     BearerAuth
// @Success      200     {object}  model.AIEstimationSuccessDoc
// @Router       /api/v1/dashboard/ai-estimation [get]
func (h *DashboardHandler) GetAIEstimation(c *gin.Context) {
	result, err := h.useCase.GetAIEstimation(c.Request.Context())
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "Gagal menghitung estimasi", err.Error())
		return
	}

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
