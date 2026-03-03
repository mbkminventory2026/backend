package httpdelivery

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"permatatex-inventory/pkg/response"
)

// HTTPError represents a handled application error with explicit HTTP status code.
type HTTPError struct {
	Code    int
	Message string
	Detail  any
}

func (e *HTTPError) Error() string {
	return e.Message
}

// NewHTTPError constructs a typed HTTPError for delivery/usecase layers.
func NewHTTPError(code int, message string, detail any) *HTTPError {
	return &HTTPError{
		Code:    code,
		Message: message,
		Detail:  detail,
	}
}

// ErrorHandlerMiddleware standardizes panic and handler errors into response envelope.
func ErrorHandlerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				slog.Error(
					"panic recovered in request",
					slog.Any("panic", recovered),
					slog.String("method", c.Request.Method),
					slog.String("path", c.Request.URL.Path),
				)

				response.Fail(c, http.StatusInternalServerError, response.MessageInternalServerError, nil)
				c.Abort()
			}
		}()

		c.Next()

		if c.Writer.Written() || len(c.Errors) == 0 {
			return
		}

		err := c.Errors.Last().Err
		if response.IsValidationError(err) {
			response.Fail(c, http.StatusBadRequest, response.MessageValidationFailed, response.FormatValidationError(err))
			return
		}

		var httpErr *HTTPError
		if errors.As(err, &httpErr) {
			response.Fail(c, httpErr.Code, httpErr.Message, httpErr.Detail)
			return
		}

		slog.Error(
			"unhandled request error",
			slog.String("error", err.Error()),
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
		)
		response.Fail(c, http.StatusInternalServerError, response.MessageInternalServerError, nil)
	}
}

// AbortWithError appends an error to gin context and stops request chain.
func AbortWithError(c *gin.Context, err error) {
	_ = c.Error(err)
	c.Abort()
}

// AbortWithBadRequest is a helper for malformed payload and validation failures.
func AbortWithBadRequest(c *gin.Context, err error) {
	if err == nil {
		err = fmt.Errorf(response.MessageBadRequest)
	}
	AbortWithError(c, NewHTTPError(http.StatusBadRequest, response.MessageBadRequest, response.FormatValidationError(err)))
}
