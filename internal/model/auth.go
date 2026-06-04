package model

import "permatatex-inventory/pkg/response"

type LoginRequest struct {
	Username       string `json:"username" binding:"required"`
	Password       string `json:"password" binding:"required"`
	TurnstileToken string `json:"turnstile_token" binding:"required"`
}

type LoginResponse struct {
	AccessToken        string `json:"access_token"`
	TokenType          string `json:"token_type"`
	ExpiresIn          int64  `json:"expires_in"`
	IDRole             int32  `json:"id_role"`
	RoleName           string `json:"role_name"`
	MustChangePassword bool   `json:"must_change_password"`
}

// LoginSuccessDoc is the Swagger schema for successful login.
type LoginSuccessDoc struct {
	Status  string        `json:"status" example:"success"`
	Message string        `json:"message" example:"login successful"`
	Data    LoginResponse `json:"data"`
}

// LoginBadRequestDoc is the Swagger schema for malformed payload or captcha validation failures.
type LoginBadRequestDoc struct {
	Status  string `json:"status" example:"error"`
	Message string `json:"message" example:"bad request"`
	Error   any    `json:"error"`
}

// LoginUnauthorizedDoc is the Swagger schema for invalid credentials.
type LoginUnauthorizedDoc struct {
	Status  string `json:"status" example:"error"`
	Message string `json:"message" example:"invalid username or password"`
}

// LoginRateLimitErrorDetail contains machine-readable metadata for throttled login attempts.
type LoginRateLimitErrorDetail struct {
	Code              string `json:"code" example:"too_many_login_attempts"`
	RetryAfterSeconds int    `json:"retry_after_seconds" example:"60"`
}

// LoginTooManyRequestsDoc is the Swagger schema for login rate limit responses.
type LoginTooManyRequestsDoc struct {
	Status  string                    `json:"status" example:"error"`
	Message string                    `json:"message" example:"too many login attempts, please try again later"`
	Error   LoginRateLimitErrorDetail `json:"error"`
}

// LoginBadGatewayDoc is the Swagger schema for captcha upstream failures.
type LoginBadGatewayDoc struct {
	Status  string               `json:"status" example:"error"`
	Message string               `json:"message" example:"captcha verification service unavailable"`
	Error   TurnstileErrorDetail `json:"error"`
}

// AuthServiceUnavailableErrorDetail contains machine-readable metadata when auth service is unavailable.
type AuthServiceUnavailableErrorDetail struct {
	Code string `json:"code" example:"auth_service_unavailable"`
}

// LoginServiceUnavailableDoc is the Swagger schema for internal auth service errors.
type LoginServiceUnavailableDoc struct {
	Status  string                            `json:"status" example:"error"`
	Message string                            `json:"message" example:"authentication service unavailable"`
	Error   AuthServiceUnavailableErrorDetail `json:"error"`
}

// GetMeResponse is the profile payload returned by /api/v1/auth/me.
type GetMeResponse struct {
	UserID             int32  `json:"user_id" example:"1"`
	IDRole             int32  `json:"id_role" example:"1"`
	RoleName           string `json:"role_name" example:"OPERATOR"`
	MustChangePassword bool   `json:"must_change_password" example:"false"`
}

// GetMeSuccessDoc is the Swagger schema for successful profile retrieval.
type GetMeSuccessDoc struct {
	Status  string        `json:"status" example:"success"`
	Message string        `json:"message" example:"profile retrieved"`
	Data    GetMeResponse `json:"data"`
}

// GetMeUnauthorizedDoc is the Swagger schema for unauthorized profile access.
type GetMeUnauthorizedDoc struct {
	Status  string `json:"status" example:"error"`
	Message string `json:"message" example:"unauthorized"`
	Error   any    `json:"error,omitempty"`
}

// LoginValidationErrorDoc shows the shape used when payload binding/validation fails.
type LoginValidationErrorDoc struct {
	Status  string                         `json:"status" example:"error"`
	Message string                         `json:"message" example:"bad request"`
	Error   []response.ValidationErrorItem `json:"error"`
}

type RegisterMitraRequest struct {
	NamaPerusahaan string  `json:"nama_perusahaan" binding:"required"`
	TipePerusahaan string  `json:"tipe_perusahaan" binding:"required,oneof=Client Supplier"`
	Email          *string `json:"email" binding:"omitempty,email"`
	NoTelp         *string `json:"no_telp"`
	Alamat         *string `json:"alamat"`
	Kota           *string `json:"kota"`
	KodePos        *string `json:"kode_pos"`

	TurnstileToken string `json:"turnstile_token" binding:"required"`
}

type ChangePasswordRequest struct {
	CurrentPassword    string `json:"current_password" binding:"required,min=6"`
	NewPassword        string `json:"new_password" binding:"required,min=6"`
	ConfirmNewPassword string `json:"confirm_new_password" binding:"required,min=6"`
}

type ChangePasswordSuccessDoc struct {
	Status  string        `json:"status" example:"success"`
	Message string        `json:"message" example:"password changed successfully"`
	Data    LoginResponse `json:"data"`
}

type ForgotPasswordRequestCreateRequest struct {
	Username string `json:"username" binding:"required"`
	Reason   string `json:"reason"`
}

type RejectPasswordResetRequestRequest struct {
	RejectedReason string `json:"rejected_reason"`
}

type PasswordResetRequestResponse struct {
	IDPasswordResetRequest int32  `json:"id_password_reset_request"`
	IDUser                 int32  `json:"id_user"`
	Username               string `json:"username"`
	IDRole                 int32  `json:"id_role"`
	NamaRole               string `json:"nama_role"`
	Reason                 string `json:"reason"`
	Status                 string `json:"status"`
	RequestedAt            string `json:"requested_at"`
	ApprovedAt             string `json:"approved_at,omitempty"`
	RejectedAt             string `json:"rejected_at,omitempty"`
	CompletedAt            string `json:"completed_at,omitempty"`
	RejectedReason         string `json:"rejected_reason,omitempty"`
	ApprovedBy             *int32 `json:"approved_by,omitempty"`
	ApprovedByUsername     string `json:"approved_by_username,omitempty"`
	RejectedBy             *int32 `json:"rejected_by,omitempty"`
	RejectedByUsername     string `json:"rejected_by_username,omitempty"`
}

type PasswordResetRequestListSuccessDoc struct {
	Status  string                         `json:"status" example:"success"`
	Message string                         `json:"message" example:"password reset requests retrieved"`
	Data    []PasswordResetRequestResponse `json:"data"`
}

type ApprovePasswordResetResponse struct {
	PasswordResetRequestResponse
	TemporaryPassword string `json:"temporary_password"`
}

type ApprovePasswordResetSuccessDoc struct {
	Status  string                       `json:"status" example:"success"`
	Message string                       `json:"message" example:"password reset request approved"`
	Data    ApprovePasswordResetResponse `json:"data"`
}
