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
	messageAuthServiceUnavailable   = "authentication service unavailable"
	errorCodeAuthServiceUnavailable = "auth_service_unavailable"

	messageCaptchaRequired           = "captcha token is required"
	errorCodeCaptchaMissing          = "captcha_token_missing"
	messageCaptchaVerificationFailed = "captcha verification failed"
	errorCodeCaptchaInvalid          = "captcha_token_invalid"
	messageCaptchaServiceUnavailable = "captcha verification service unavailable"
	errorCodeCaptchaServiceUnavailable = "captcha_service_unavailable"
)

type AuthHandler struct {
	authUseCase *usecase.AuthUseCase
}

func NewAuthHandler(authUseCase *usecase.AuthUseCase) (*AuthHandler, error) {
	if authUseCase == nil {
		return nil, errors.New("auth usecase is required")
	}

	return &AuthHandler{
		authUseCase: authUseCase,
	}, nil
}

func (h *AuthHandler) RegisterRoutes(
	router gin.IRouter,
	authMiddleware gin.HandlerFunc,
	loginRateLimitMiddleware gin.HandlerFunc,
) {
	auth := router.Group("/api/v1/auth")

	if loginRateLimitMiddleware != nil {
		auth.POST("/login", loginRateLimitMiddleware, h.Login)
	} else {
		auth.POST("/login", h.Login)
	}

	// Protected routes
	protected := auth.Group("").Use(authMiddleware)
	protected.GET("/me", h.GetMe)
}

// Login godoc
// @Summary      Sign In
// @Description  Authenticates user using username, password, and Turnstile token, then returns JWT access token.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        payload  body      model.LoginRequest  true  "Login payload"
// @Success      200      {object}  model.LoginSuccessDoc
// @Failure      400      {object}  model.LoginBadRequestDoc
// @Failure      401      {object}  model.LoginUnauthorizedDoc
// @Failure      429      {object}  model.LoginTooManyRequestsDoc
// @Failure      502      {object}  model.LoginBadGatewayDoc
// @Failure      503      {object}  model.LoginServiceUnavailableDoc
// @Router       /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req model.LoginRequest
	if !BindJSON(c, &req) {
		return
	}

	res, err := h.authUseCase.Login(c.Request.Context(), req, c.ClientIP())
	if err != nil {
		h.handleLoginError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "login successful", res)
}

func (h *AuthHandler) handleLoginError(c *gin.Context, err error) {
	if errors.Is(err, usecase.ErrInvalidCredentials) {
		AbortWithError(c, NewHTTPError(http.StatusUnauthorized, usecase.ErrInvalidCredentials.Error(), nil))
		return
	}

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

	if errors.Is(err, usecase.ErrAuthServiceUnavailable) {
		AbortWithError(c, NewHTTPError(
			http.StatusServiceUnavailable,
			messageAuthServiceUnavailable,
			gin.H{"code": errorCodeAuthServiceUnavailable},
		))
		return
	}

	AbortWithError(c, err)
}

// GetMe godoc
// @Summary      Get Current User Profile
// @Description  Returns authenticated user ID extracted from JWT token.
// @Tags         Auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  model.GetMeSuccessDoc
// @Failure      401  {object}  model.GetMeUnauthorizedDoc
// @Router       /api/v1/auth/me [get]
func (h *AuthHandler) GetMe(c *gin.Context) {
	userID, ok := GetUserIDFromContext(c)
	if !ok {
		AbortWithError(c, NewHTTPError(http.StatusUnauthorized, "unauthorized", nil))
		return
	}

	response.Success(c, http.StatusOK, "profile retrieved", gin.H{
		"user_id": userID,
	})
}
