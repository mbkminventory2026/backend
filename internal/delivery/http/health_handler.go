package httpdelivery

import (
	"github.com/gin-gonic/gin"

	"permatatex-inventory/pkg/response"
)

// HealthHandler handles health check transport concerns.
type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) RegisterRoutes(router gin.IRouter) {
	router.GET("/health", h.GetHealth)
}

// GetHealth godoc
// @Summary      Health Check
// @Description  Returns service health status
// @Tags         System
// @Produce      json
// @Success      200  {object}  response.HealthResponse
// @Router       /health [get]
func (h *HealthHandler) GetHealth(c *gin.Context) {
	response.Health(c)
}
