package httpdelivery

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"permatatex-inventory/pkg/response"
)

// HealthHandler handles health check transport concerns.
type HealthHandler struct {
	db *pgxpool.Pool
}

func NewHealthHandler(db *pgxpool.Pool) *HealthHandler {
	return &HealthHandler{
		db: db,
	}
}

func (h *HealthHandler) RegisterRoutes(router gin.IRouter) {
	router.GET("/health", h.GetHealth)
}

// GetHealth godoc
// @Summary      Health Check
// @Description  Returns service health status, including database connectivity.
// @Tags         System
// @Produce      json
// @Success      200  {object}  response.HealthResponse
// @Failure      503  {object}  response.HealthResponse
// @Router       /health [get]
func (h *HealthHandler) GetHealth(c *gin.Context) {
	if h.db != nil {
		if err := h.db.Ping(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, response.HealthResponse{Status: "error"})
			return
		}
	}

	response.Health(c)
}
