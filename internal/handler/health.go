package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"whatsapp-h2h-otomax/internal/config"
	"whatsapp-h2h-otomax/internal/service"
	"whatsapp-h2h-otomax/pkg/logger"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	whatsappService *service.WhatsAppService
	config          *config.Config
	logger          *logger.Logger
	startTime       time.Time
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(waService *service.WhatsAppService, cfg *config.Config, log *logger.Logger) *HealthHandler {
	return &HealthHandler{
		whatsappService: waService,
		config:          cfg,
		logger:          log,
		startTime:       time.Now(),
	}
}

// CheckHealth handles GET /health
func (h *HealthHandler) CheckHealth(w http.ResponseWriter, r *http.Request) {
	// Get WhatsApp connection status
	waStatus := h.whatsappService.GetConnectionStatus()

	// Calculate uptime
	uptime := time.Since(h.startTime)

	response := map[string]interface{}{
		"status": "healthy",
		"whatsapp": waStatus,
		"otomax_webhook": map[string]interface{}{
			"configured": h.config.Otomax.WebhookURL != "",
			"url":        h.config.Otomax.WebhookURL,
		},
		"uptime":    uptime.String(),
		"timestamp": time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

