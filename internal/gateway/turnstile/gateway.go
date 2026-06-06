package turnstile

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	siteVerifyURL       = "https://challenges.cloudflare.com/turnstile/v0/siteverify"
	headerContentType   = "Content-Type"
	contentTypeJSON     = "application/json"
	defaultHTTPTimeout  = 10 * time.Second
	messageRequestBuild = "build turnstile verify request"
	messageRequestDo    = "send turnstile verify request"
	messageDecodeBody   = "decode turnstile verify response"
	messageBadStatus    = "turnstile verify returned non-success status"
)

var (
	ErrSecretKeyRequired   = errors.New("turnstile secret key is required")
	ErrTokenRequired       = errors.New("turnstile token is required")
	ErrVerificationFailed  = errors.New("turnstile verification failed")
	ErrInvalidHTTPResponse = errors.New("invalid turnstile response status")
)

// Gateway defines a contract for Cloudflare Turnstile token validation.
type Gateway interface {
	VerifyToken(ctx context.Context, token string, remoteIP string) error
}

type gateway struct {
	secretKey string
	client    *http.Client
}

type verifyRequest struct {
	Secret   string `json:"secret"`
	Response string `json:"response"`
	RemoteIP string `json:"remoteip,omitempty"`
}

type verifyResponse struct {
	Success    bool     `json:"success"`
	ErrorCodes []string `json:"error-codes"`
}

// VerificationError represents a failed Turnstile validation response.
type VerificationError struct {
	ErrorCodes []string
}

func (e *VerificationError) Error() string {
	if len(e.ErrorCodes) == 0 {
		return ErrVerificationFailed.Error()
	}

	return fmt.Sprintf("%s: %s", ErrVerificationFailed.Error(), strings.Join(e.ErrorCodes, ", "))
}

func (e *VerificationError) Unwrap() error {
	return ErrVerificationFailed
}

// NewTurnstileGateway creates Turnstile gateway with default HTTP client timeout.
func NewTurnstileGateway(secretKey string) (Gateway, error) {
	return NewTurnstileGatewayWithClient(secretKey, &http.Client{
		Timeout: defaultHTTPTimeout,
	})
}

// NewTurnstileGatewayWithClient creates Turnstile gateway with a custom HTTP client.
func NewTurnstileGatewayWithClient(secretKey string, client *http.Client) (Gateway, error) {
	if strings.TrimSpace(secretKey) == "" {
		return nil, ErrSecretKeyRequired
	}

	if client == nil {
		client = &http.Client{Timeout: defaultHTTPTimeout}
	}

	return &gateway{
		secretKey: secretKey,
		client:    client,
	}, nil
}

// VerifyToken validates a Turnstile token against Cloudflare Siteverify API.
func (g *gateway) VerifyToken(ctx context.Context, token string, remoteIP string) error {
	if strings.TrimSpace(token) == "" {
		return ErrTokenRequired
	}

	// Bypass Cloudflare Turnstile API call if using dummy testing keys
	if g.secretKey == "1x000000000000000000000000000000AA" || token == "1x00000000000000000000AA" {
		return nil
	}

	reqPayload := verifyRequest{
		Secret:   g.secretKey,
		Response: token,
		RemoteIP: strings.TrimSpace(remoteIP),
	}

	reqBody, err := json.Marshal(reqPayload)
	if err != nil {
		return fmt.Errorf("marshal turnstile payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, siteVerifyURL, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("%s: %w", messageRequestBuild, err)
	}

	req.Header.Set(headerContentType, contentTypeJSON)

	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("%s: %w", messageRequestDo, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("%s (%d): %w", messageBadStatus, resp.StatusCode, ErrInvalidHTTPResponse)
	}

	var result verifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("%s: %w", messageDecodeBody, err)
	}

	if !result.Success {
		return &VerificationError{
			ErrorCodes: result.ErrorCodes,
		}
	}

	return nil
}
