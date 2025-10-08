package middleware

import (
	"encoding/json"
	"net/http"

	"whatsapp-h2h-otomax/internal/model"
	"whatsapp-h2h-otomax/pkg/logger"
)

// AuthMiddleware provides API key authentication
type AuthMiddleware struct {
	apiKey string
	logger *logger.Logger
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(apiKey string, log *logger.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		apiKey: apiKey,
		logger: log,
	}
}

// Authenticate validates API key from request header
func (m *AuthMiddleware) Authenticate(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Skip authentication if API_KEY is not configured (local mode)
		if m.apiKey == "" {
			m.logger.Debug("API authentication disabled (running in local mode)",
				"path", r.URL.Path,
				"method", r.Method,
			)
			next(w, r)
			return
		}

		// API_KEY is configured, validate it
		apiKey := r.Header.Get("X-API-Key")

		if apiKey == "" {
			m.logger.Warn("Missing API key",
				"path", r.URL.Path,
				"method", r.Method,
				"remote_addr", r.RemoteAddr,
			)
			m.sendErrorResponse(w, "ERR_UNAUTHORIZED", "Missing API key", http.StatusUnauthorized)
			return
		}

		if apiKey != m.apiKey {
			m.logger.Warn("Invalid API key",
				"path", r.URL.Path,
				"method", r.Method,
				"remote_addr", r.RemoteAddr,
			)
			m.sendErrorResponse(w, "ERR_UNAUTHORIZED", "Invalid API key", http.StatusUnauthorized)
			return
		}

		// API key is valid, proceed to next handler
		next(w, r)
	}
}

// sendErrorResponse sends error response in JSON format
func (m *AuthMiddleware) sendErrorResponse(w http.ResponseWriter, code, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := model.TransactionResponse{
		Status:  "error",
		Message: message,
		Error: &model.TransactionError{
			Code:    code,
			Message: message,
		},
	}

	json.NewEncoder(w).Encode(response)
}

