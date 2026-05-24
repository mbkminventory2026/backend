package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"permatatex-inventory/internal/model" // Sesuaikan nama module dengan go.mod kamu
)

type Gateway struct {
	baseURL    string
	httpClient *http.Client
}

// NewGateway inisialisasi AI client
func NewGateway(baseURL string) *Gateway {
	return &Gateway{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 60 * time.Second, // Timeout aman untuk proses AI
		},
	}
}

// PredictSchedule menembak API FastAPI Python
func (g *Gateway) PredictSchedule(ctx context.Context, reqData model.AIPredictionRequest) (*model.AIPredictionResponseData, error) {
	reqBody, err := json.Marshal(reqData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/predict", g.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request to AI service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("AI service returned status: %d", resp.StatusCode)
	}

	var aiResp model.AIPredictionResponse
	if err := json.NewDecoder(resp.Body).Decode(&aiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if aiResp.Status != "success" {
		return nil, fmt.Errorf("AI service error: %s", aiResp.Message)
	}

	return &aiResp.Data, nil
}