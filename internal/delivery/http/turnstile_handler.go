package httpdelivery

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	turnstilegateway "permatatex-inventory/internal/gateway/turnstile"
	"permatatex-inventory/internal/model"
	"permatatex-inventory/internal/usecase"
	"permatatex-inventory/pkg/response"
)

const (
	messageCaptchaVerificationSucceeded = "captcha verification succeeded"
	messageCaptchaVerificationFailed    = "captcha verification failed"
	messageCaptchaRequired              = "captcha is required"
	messageCaptchaServiceUnavailable    = "captcha verification service unavailable"
	errorCodeCaptchaInvalid             = "captcha_invalid"
	errorCodeCaptchaMissing             = "captcha_missing"
	errorCodeCaptchaServiceUnavailable  = "captcha_service_unavailable"
)

// TurnstileHandler handles transport logic for captcha verification endpoints.
type TurnstileHandler struct {
	useCase *usecase.TurnstileUseCase
}

func NewTurnstileHandler(useCase *usecase.TurnstileUseCase) (*TurnstileHandler, error) {
	if useCase == nil {
		return nil, errors.New("turnstile usecase is required")
	}

	return &TurnstileHandler{
		useCase: useCase,
	}, nil
}

func (h *TurnstileHandler) RegisterRoutes(router gin.IRouter) {
	auth := router.Group("/api/v1/auth")
	auth.POST("/turnstile/verify", h.VerifyTurnstile)
}

// VerifyTurnstile godoc
// @Summary      Verify Turnstile Token
// @Description  Validates frontend Turnstile token via Cloudflare siteverify API.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        payload  body      model.VerifyTurnstileRequest  true  "Turnstile payload"
// @Success      200      {object}  model.VerifyTurnstileSuccessDoc
// @Failure      400      {object}  model.VerifyTurnstileBadRequestDoc
// @Failure      502      {object}  model.VerifyTurnstileBadGatewayDoc
// @Router       /api/v1/auth/turnstile/verify [post]
func (h *TurnstileHandler) VerifyTurnstile(c *gin.Context) {
	var req model.VerifyTurnstileRequest
	if !BindJSON(c, &req) {
		return
	}

	result, err := h.useCase.VerifyToken(c.Request.Context(), req, c.ClientIP())
	if err != nil {
		h.handleVerificationError(c, err)
		return
	}

	response.Success(c, http.StatusOK, messageCaptchaVerificationSucceeded, result)
}

func (h *TurnstileHandler) handleVerificationError(c *gin.Context, err error) {
	if usecase.IsTurnstileTokenRequiredError(err) {
		AbortWithError(c, NewHTTPError(
			http.StatusBadRequest,
			messageCaptchaRequired,
			model.TurnstileErrorDetail{
				Code: errorCodeCaptchaMissing,
			},
		))
		return
	}

	if usecase.IsTurnstileVerificationError(err) {
		errorDetail := model.TurnstileErrorDetail{
			Code: errorCodeCaptchaInvalid,
		}

		var verificationErr *turnstilegateway.VerificationError
		if errors.As(err, &verificationErr) {
			errorDetail.ErrorCodes = verificationErr.ErrorCodes
		}

		AbortWithError(c, NewHTTPError(
			http.StatusBadRequest,
			messageCaptchaVerificationFailed,
			errorDetail,
		))
		return
	}

	if usecase.IsTurnstileTransportError(err) {
		AbortWithError(c, NewHTTPError(
			http.StatusBadGateway,
			messageCaptchaServiceUnavailable,
			model.TurnstileErrorDetail{
				Code: errorCodeCaptchaServiceUnavailable,
			},
		))
		return
	}

	AbortWithError(c, err)
}
