package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"time"

	"whatsapp-h2h-otomax/internal/config"
	"whatsapp-h2h-otomax/internal/model"
	"whatsapp-h2h-otomax/pkg/logger"
)

// OtomaxService handles webhook delivery to Otomax
type OtomaxService struct {
	httpClient *http.Client
	config     *config.OtomaxConfig
	logger     *logger.Logger
}

// NewOtomaxService creates a new Otomax service
func NewOtomaxService(cfg *config.OtomaxConfig, log *logger.Logger) *OtomaxService {
	return &OtomaxService{
		httpClient: &http.Client{
			Timeout: cfg.WebhookTimeout,
		},
		config: cfg,
		logger: log,
	}
}

// SendWebhook sends webhook payload to Otomax with retry mechanism
func (s *OtomaxService) SendWebhook(ctx context.Context, payload *model.WebhookPayload, trxID string) error {
	var lastErr error

	for attempt := 0; attempt <= s.config.RetryCount; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * time.Second
			s.logger.WithTrxID(trxID).Warn("Retrying webhook delivery",
				"attempt", attempt+1,
				"backoff_seconds", backoff.Seconds(),
			)
			time.Sleep(backoff)
		}

		err := s.send(ctx, payload)
		if err == nil {
			// Only log if retry attempt or first time success
			if attempt > 0 {
				s.logger.WithTrxID(trxID).Info("Webhook delivered",
					"attempt", attempt+1,
				)
			}
			return nil
		}

		lastErr = err
		s.logger.WithTrxID(trxID).Warn("Webhook delivery failed",
			"attempt", attempt+1,
			"error", err,
		)
	}

	return fmt.Errorf("webhook delivery failed after %d attempts: %w",
		s.config.RetryCount+1, lastErr)
}

// send performs the actual HTTP request to Otomax webhook
func (s *OtomaxService) send(ctx context.Context, payload *model.WebhookPayload) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.config.WebhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "whatsapp-h2h-otomax/1.0")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

