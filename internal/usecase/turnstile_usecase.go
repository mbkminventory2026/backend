package usecase

import (
	"context"
	"errors"

	turnstilegateway "permatatex-inventory/internal/gateway/turnstile"
	"permatatex-inventory/internal/model"
)

// TurnstileVerifier abstracts token verification from external gateways.
type TurnstileVerifier interface {
	VerifyToken(ctx context.Context, token string, remoteIP string) error
}

// TurnstileUseCase orchestrates captcha verification business flow.
type TurnstileUseCase struct {
	verifier TurnstileVerifier
}

func NewTurnstileUseCase(verifier TurnstileVerifier) (*TurnstileUseCase, error) {
	if verifier == nil {
		return nil, errors.New("turnstile verifier is required")
	}

	return &TurnstileUseCase{
		verifier: verifier,
	}, nil
}

func (u *TurnstileUseCase) VerifyToken(
	ctx context.Context,
	req model.VerifyTurnstileRequest,
	remoteIP string,
) (*model.VerifyTurnstileResponse, error) {
	if err := u.verifier.VerifyToken(ctx, req.TurnstileToken, remoteIP); err != nil {
		return nil, err
	}

	return &model.VerifyTurnstileResponse{
		Verified: true,
	}, nil
}

func IsTurnstileVerificationError(err error) bool {
	return errors.Is(err, turnstilegateway.ErrVerificationFailed)
}

func IsTurnstileTokenRequiredError(err error) bool {
	return errors.Is(err, turnstilegateway.ErrTokenRequired)
}

func IsTurnstileTransportError(err error) bool {
	return errors.Is(err, turnstilegateway.ErrInvalidHTTPResponse)
}
