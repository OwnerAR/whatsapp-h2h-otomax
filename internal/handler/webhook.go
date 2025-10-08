package handler

import (
	"encoding/json"
	"net/http"

	"whatsapp-h2h-otomax/internal/config"
	"whatsapp-h2h-otomax/pkg/logger"
)

// WebhookHandler handles webhook-related requests
type WebhookHandler struct {
	config *config.Config
	logger *logger.Logger
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(cfg *config.Config, log *logger.Logger) *WebhookHandler {
	return &WebhookHandler{
		config: cfg,
		logger: log,
	}
}

// ReceiveMessage handles POST /api/v1/webhook/message
// Note: This endpoint is currently handled by WhatsApp event handler internally
// This can be used for testing or manual webhook triggers
func (h *WebhookHandler) ReceiveMessage(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Webhook message endpoint called",
		"method", r.Method,
		"remote_addr", r.RemoteAddr,
	)

	response := map[string]interface{}{
		"status":  "success",
		"message": "Webhook endpoint is available but messages are handled automatically by WhatsApp event listener",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

