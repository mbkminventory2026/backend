package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	StatusSuccess = "success"
	StatusError   = "error"
	StatusOK      = "ok"
)

const (
	MessageSuccess             = "success"
	MessageBadRequest          = "bad request"
	MessageValidationFailed    = "validation failed"
	MessageInvalidRequestBody  = "invalid request body"
	MessageInternalServerError = "internal server error"
)

// BaseResponse is the standard response envelope for all HTTP endpoints.
type BaseResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
	Error   any    `json:"error,omitempty"`
}

// HealthResponse is the minimal payload for service health checks.
type HealthResponse struct {
	Status string `json:"status" example:"ok"`
}

// JSON writes a BaseResponse with a specific HTTP status code.
func JSON(c *gin.Context, httpCode int, payload BaseResponse) {
	c.JSON(httpCode, payload)
}

// Success writes a successful standardized response.
func Success(c *gin.Context, httpCode int, message string, data any) {
	JSON(c, httpCode, BaseResponse{
		Status:  StatusSuccess,
		Message: message,
		Data:    data,
		Error:   nil,
	})
}

// Fail writes a failed standardized response.
func Fail(c *gin.Context, httpCode int, message string, errDetail any) {
	JSON(c, httpCode, BaseResponse{
		Status:  StatusError,
		Message: message,
		Data:    nil,
		Error:   errDetail,
	})
}

// Health writes a minimal health check response.
func Health(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{Status: StatusOK})
}
