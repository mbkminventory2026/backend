package model

// VerifyTurnstileRequest is the payload sent by frontend for server-side captcha verification.
type VerifyTurnstileRequest struct {
	TurnstileToken string `json:"turnstile_token" binding:"required,max=2048"`
}

// VerifyTurnstileResponse contains successful verification state.
type VerifyTurnstileResponse struct {
	Verified bool `json:"verified" example:"true"`
}

// TurnstileErrorDetail provides machine-readable captcha failure details.
type TurnstileErrorDetail struct {
	Code       string   `json:"code" example:"captcha_invalid"`
	ErrorCodes []string `json:"error_codes,omitempty"`
}

// VerifyTurnstileSuccessDoc is the Swagger schema for successful captcha verification.
type VerifyTurnstileSuccessDoc struct {
	Status  string                  `json:"status" example:"success"`
	Message string                  `json:"message" example:"captcha verification succeeded"`
	Data    VerifyTurnstileResponse `json:"data"`
}

// VerifyTurnstileBadRequestDoc is the Swagger schema for invalid captcha request/token.
type VerifyTurnstileBadRequestDoc struct {
	Status  string               `json:"status" example:"error"`
	Message string               `json:"message" example:"captcha verification failed"`
	Error   TurnstileErrorDetail `json:"error"`
}

// VerifyTurnstileBadGatewayDoc is the Swagger schema for Turnstile upstream failures.
type VerifyTurnstileBadGatewayDoc struct {
	Status  string               `json:"status" example:"error"`
	Message string               `json:"message" example:"captcha verification service unavailable"`
	Error   TurnstileErrorDetail `json:"error"`
}
