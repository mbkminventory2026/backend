package httpdelivery

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	turnstilegateway "permatatex-inventory/internal/gateway/turnstile"
	"permatatex-inventory/internal/model"
	"permatatex-inventory/internal/usecase"
	"permatatex-inventory/pkg/response"
)

const (
	messageAuthServiceUnavailable   = "authentication service unavailable"
	errorCodeAuthServiceUnavailable = "auth_service_unavailable"

	messageCaptchaRequired             = "captcha token is required"
	errorCodeCaptchaMissing            = "captcha_token_missing"
	messageCaptchaVerificationFailed   = "captcha verification failed"
	errorCodeCaptchaInvalid            = "captcha_token_invalid"
	messageCaptchaServiceUnavailable   = "captcha verification service unavailable"
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

	auth.POST("/register-mitra", h.RegisterMitra)
	auth.POST("/forgot-password-requests", h.CreateForgotPasswordRequest)

	// Protected routes
	protected := auth.Group("").Use(authMiddleware)
	protected.GET("/me", h.GetMe)
	protected.POST("/change-password", h.ChangePassword)
	protected.GET("/forgot-password-requests", RequirePermission(PermissionPasswordResetRequestRead), h.ListForgotPasswordRequests)
	protected.PATCH("/forgot-password-requests/:id/approve", RequirePermission(PermissionPasswordResetRequestApprove), h.ApproveForgotPasswordRequest)
	protected.PATCH("/forgot-password-requests/:id/reject", RequirePermission(PermissionPasswordResetRequestReject), h.RejectForgotPasswordRequest)
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

	payload, exists := c.Get(authorizationPayloadKey)
	if !exists {
		AbortWithError(c, NewHTTPError(http.StatusUnauthorized, "unauthorized", nil))
		return
	}

	claims, ok := payload.(jwt.MapClaims)
	if !ok {
		AbortWithError(c, NewHTTPError(http.StatusUnauthorized, "invalid token payload", nil))
		return
	}

	roleIDFloat, roleIDOK := claims["id_role"].(float64)
	roleName, roleNameOK := claims["role_name"].(string)
	mustChangePassword, mustChangePasswordOK := claims["must_change_password"].(bool)
	if !roleIDOK || !roleNameOK || !mustChangePasswordOK {
		AbortWithError(c, NewHTTPError(http.StatusUnauthorized, "invalid token payload", nil))
		return
	}

	response.Success(c, http.StatusOK, "profile retrieved", gin.H{
		"user_id":              userID,
		"id_role":              int32(roleIDFloat),
		"role_name":            roleName,
		"must_change_password": mustChangePassword,
	})
}

// RegisterMitra godoc
// @Summary      Self Register Mitra
// @Description  Allows a new partner/client to self-register an account and company profile.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        payload  body      model.RegisterMitraRequest  true  "Register payload"
// @Success      201      {object}  response.BaseResponse
// @Failure      400      {object}  model.LoginBadRequestDoc
// @Failure      409      {object}  model.LoginBadRequestDoc
// @Router       /api/v1/auth/register-mitra [post]
func (h *AuthHandler) RegisterMitra(c *gin.Context) {
	var req model.RegisterMitraRequest
	if !BindJSON(c, &req) {
		return
	}

	err := h.authUseCase.RegisterMitra(c.Request.Context(), req, c.ClientIP())
	if err != nil {
		if errors.Is(err, usecase.ErrUsernameAlreadyExists) {
			AbortWithError(c, NewHTTPError(http.StatusConflict, "username sudah digunakan", nil))
			return
		}

		h.handleLoginError(c, err)
		return
	}

	response.Success(c, http.StatusCreated, "registrasi berhasil, menunggu persetujuan admin", nil)
}

// ChangePassword godoc
// @Summary      Change Password
// @Description  Allows an authenticated user to change their password and refresh their session token.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        payload  body      model.ChangePasswordRequest  true  "Change password payload"
// @Success      200      {object}  model.ChangePasswordSuccessDoc
// @Failure      400      {object}  model.LoginBadRequestDoc
// @Failure      401      {object}  model.GetMeUnauthorizedDoc
// @Failure      503      {object}  model.LoginServiceUnavailableDoc
// @Router       /api/v1/auth/change-password [post]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID, ok := GetUserIDFromContext(c)
	if !ok {
		AbortWithError(c, NewHTTPError(http.StatusUnauthorized, "unauthorized", nil))
		return
	}

	var req model.ChangePasswordRequest
	if !BindJSON(c, &req) {
		return
	}

	res, err := h.authUseCase.ChangePassword(c.Request.Context(), userID, req)
	if err != nil {
		h.handlePasswordFlowError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "password changed successfully", res)
}

// CreateForgotPasswordRequest godoc
// @Summary      Create Forgot Password Request
// @Description  Allows a user to request a manual password reset without email.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        payload  body      model.ForgotPasswordRequestCreateRequest  true  "Forgot password request payload"
// @Success      201      {object}  response.BaseResponse
// @Failure      400      {object}  model.LoginBadRequestDoc
// @Failure      404      {object}  model.LoginBadRequestDoc
// @Failure      409      {object}  model.LoginBadRequestDoc
// @Failure      503      {object}  model.LoginServiceUnavailableDoc
// @Router       /api/v1/auth/forgot-password-requests [post]
func (h *AuthHandler) CreateForgotPasswordRequest(c *gin.Context) {
	var req model.ForgotPasswordRequestCreateRequest
	if !BindJSON(c, &req) {
		return
	}

	res, err := h.authUseCase.CreateForgotPasswordRequest(c.Request.Context(), req)
	if err != nil {
		h.handlePasswordFlowError(c, err)
		return
	}

	response.Success(c, http.StatusCreated, "password reset request created", res)
}

// ListForgotPasswordRequests godoc
// @Summary      List Forgot Password Requests
// @Description  Returns all password reset requests for operator review.
// @Tags         Auth
// @Produce      json
// @Security     BearerAuth
// @Success      200      {object}  model.PasswordResetRequestListSuccessDoc
// @Failure      401      {object}  model.GetMeUnauthorizedDoc
// @Failure      403      {object}  model.LoginBadRequestDoc
// @Failure      503      {object}  model.LoginServiceUnavailableDoc
// @Router       /api/v1/auth/forgot-password-requests [get]
func (h *AuthHandler) ListForgotPasswordRequests(c *gin.Context) {
	res, err := h.authUseCase.ListForgotPasswordRequests(c.Request.Context())
	if err != nil {
		h.handlePasswordFlowError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "password reset requests retrieved", res)
}

// ApproveForgotPasswordRequest godoc
// @Summary      Approve Forgot Password Request
// @Description  Approves a pending password reset request and issues a temporary password.
// @Tags         Auth
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Password reset request ID"
// @Success      200  {object}  model.ApprovePasswordResetSuccessDoc
// @Failure      400  {object}  model.LoginBadRequestDoc
// @Failure      401  {object}  model.GetMeUnauthorizedDoc
// @Failure      404  {object}  model.LoginBadRequestDoc
// @Failure      503  {object}  model.LoginServiceUnavailableDoc
// @Router       /api/v1/auth/forgot-password-requests/{id}/approve [patch]
func (h *AuthHandler) ApproveForgotPasswordRequest(c *gin.Context) {
	userID, ok := GetUserIDFromContext(c)
	if !ok {
		AbortWithError(c, NewHTTPError(http.StatusUnauthorized, "unauthorized", nil))
		return
	}

	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid password reset request id", nil))
		return
	}

	res, err := h.authUseCase.ApproveForgotPasswordRequest(c.Request.Context(), id, userID)
	if err != nil {
		h.handlePasswordFlowError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "password reset request approved", res)
}

// RejectForgotPasswordRequest godoc
// @Summary      Reject Forgot Password Request
// @Description  Rejects a pending password reset request.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      int                                      true  "Password reset request ID"
// @Param        payload  body      model.RejectPasswordResetRequestRequest  false  "Reject payload"
// @Success      200      {object}  response.BaseResponse
// @Failure      400      {object}  model.LoginBadRequestDoc
// @Failure      401      {object}  model.GetMeUnauthorizedDoc
// @Failure      404      {object}  model.LoginBadRequestDoc
// @Failure      503      {object}  model.LoginServiceUnavailableDoc
// @Router       /api/v1/auth/forgot-password-requests/{id}/reject [patch]
func (h *AuthHandler) RejectForgotPasswordRequest(c *gin.Context) {
	userID, ok := GetUserIDFromContext(c)
	if !ok {
		AbortWithError(c, NewHTTPError(http.StatusUnauthorized, "unauthorized", nil))
		return
	}

	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid password reset request id", nil))
		return
	}

	var req model.RejectPasswordResetRequestRequest
	if c.Request.ContentLength > 0 {
		if !BindJSON(c, &req) {
			return
		}
	}

	_, err = h.authUseCase.RejectForgotPasswordRequest(c.Request.Context(), id, userID, req.RejectedReason)
	if err != nil {
		h.handlePasswordFlowError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "password reset request rejected", nil)
}

func (h *AuthHandler) handlePasswordFlowError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usecase.ErrCurrentPasswordInvalid):
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, err.Error(), nil))
		return
	case errors.Is(err, usecase.ErrPasswordConfirmationMismatch):
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, err.Error(), nil))
		return
	case errors.Is(err, usecase.ErrPasswordResetRequestAlreadyOpen):
		AbortWithError(c, NewHTTPError(http.StatusConflict, err.Error(), nil))
		return
	case errors.Is(err, usecase.ErrPasswordResetRequestNotFound):
		AbortWithError(c, NewHTTPError(http.StatusNotFound, err.Error(), nil))
		return
	case errors.Is(err, usecase.ErrUserNotFound):
		AbortWithError(c, NewHTTPError(http.StatusNotFound, err.Error(), nil))
		return
	case errors.Is(err, usecase.ErrUserValidation):
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, err.Error(), nil))
		return
	default:
		h.handleLoginError(c, err)
	}
}
