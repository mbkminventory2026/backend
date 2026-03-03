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

// JSONResponse is the standard response envelope for all HTTP endpoints.
type JSONResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
	Error   any    `json:"error,omitempty"`
}

// JSON writes a JSONResponse with a specific HTTP status code.
func JSON(c *gin.Context, httpCode int, payload JSONResponse) {
	c.JSON(httpCode, payload)
}

// Success writes a successful standardized response.
func Success(c *gin.Context, httpCode int, message string, data any) {
	JSON(c, httpCode, JSONResponse{
		Status:  StatusSuccess,
		Message: message,
		Data:    data,
		Error:   nil,
	})
}

// Fail writes a failed standardized response.
func Fail(c *gin.Context, httpCode int, message string, errDetail any) {
	JSON(c, httpCode, JSONResponse{
		Status:  StatusError,
		Message: message,
		Data:    nil,
		Error:   errDetail,
	})
}

// Health writes a minimal health check response.
func Health(c *gin.Context) {
	JSON(c, http.StatusOK, JSONResponse{Status: StatusOK})
}
