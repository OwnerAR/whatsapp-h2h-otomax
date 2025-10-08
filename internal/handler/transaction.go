package handler

import (
	"encoding/json"
	"net/http"

	"whatsapp-h2h-otomax/internal/model"
	"whatsapp-h2h-otomax/internal/service"
	"whatsapp-h2h-otomax/pkg/logger"
)

// TransactionHandler handles transaction forwarding requests
type TransactionHandler struct {
	transactionService *service.TransactionService
	logger             *logger.Logger
}

// NewTransactionHandler creates a new transaction handler
func NewTransactionHandler(txService *service.TransactionService, log *logger.Logger) *TransactionHandler {
	return &TransactionHandler{
		transactionService: txService,
		logger:             log,
	}
}

// ForwardTransaction handles GET /api/v1/forward
func (h *TransactionHandler) ForwardTransaction(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	destination := r.URL.Query().Get("destination")
	trxID := r.URL.Query().Get("trxid")
	descriptions := r.URL.Query().Get("descriptions")
	instructions := r.URL.Query().Get("instructions")

	// Validate required parameters
	if destination == "" || trxID == "" || descriptions == "" || instructions == "" {
		h.sendErrorResponse(w, "ERR_MISSING_PARAMETER", "Missing required parameters", http.StatusBadRequest)
		return
	}

	// Validate length limits
	if len(descriptions) > 4096 || len(instructions) > 4096 {
		h.sendErrorResponse(w, "ERR_INVALID_PARAMETER", "Description or instruction too long (max 4096 chars)", http.StatusBadRequest)
		return
	}

	// Check if transaction with same TrxID already exists
	existingTrx := h.transactionService.GetMessageTracker().GetByTrxID(trxID)
	if existingTrx != nil {
		h.logger.WithTrxID(trxID).Warn("Duplicate transaction detected",
			"existing_message_id", existingTrx.MessageID,
			"existing_destination", existingTrx.Destination,
			"tracker_count", h.transactionService.GetMessageTracker().Count(),
		)
		h.sendErrorResponse(w, "ERR_DUPLICATE_TRANSACTION", "Transaction with this TrxID already exists and is still being tracked", http.StatusConflict)
		return
	}
	
	h.logger.WithTrxID(trxID).Info("New transaction request", 
		"destination", destination,
		"tracker_count", h.transactionService.GetMessageTracker().Count(),
	)

	// Create request model
	req := &model.TransactionRequest{
		Destination:  destination,
		TrxID:        trxID,
		Descriptions: descriptions,
		Instructions: instructions,
	}

	// Process transaction
	data, err := h.transactionService.ProcessTransaction(r.Context(), req)
	if err != nil {
		h.logger.WithTrxID(trxID).Error("Failed to process transaction",
			"error", err,
			"destination", destination,
		)
		h.sendErrorResponse(w, h.mapErrorCode(err), err.Error(), http.StatusInternalServerError)
		return
	}

	// Send success response
	h.sendSuccessResponse(w, data)
}

// sendSuccessResponse sends success response
func (h *TransactionHandler) sendSuccessResponse(w http.ResponseWriter, data *model.TransactionData) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := model.TransactionResponse{
		Status:  "success",
		Message: "Transaction forwarded successfully",
		Data:    data,
	}

	json.NewEncoder(w).Encode(response)
}

// sendErrorResponse sends error response
func (h *TransactionHandler) sendErrorResponse(w http.ResponseWriter, code, message string, statusCode int) {
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

// mapErrorCode maps error to error code
func (h *TransactionHandler) mapErrorCode(err error) string {
	errMsg := err.Error()

	switch {
	case contains(errMsg, "invalid destination"):
		return "ERR_INVALID_DESTINATION"
	case contains(errMsg, "not connected"):
		return "ERR_WHATSAPP_NOT_CONNECTED"
	case contains(errMsg, "group not found"):
		return "ERR_GROUP_NOT_FOUND"
	case contains(errMsg, "not registered on WhatsApp"):
		return "ERR_DESTINATION_NOT_ON_WHATSAPP"
	case contains(errMsg, "failed to send"):
		return "ERR_MESSAGE_SEND_FAILED"
	default:
		return "ERR_INTERNAL_SERVER"
	}
}

// contains checks if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > 0 && len(substr) > 0 && s[:len(s)] != s[:0] && s[len(s)-len(s):] != s[:0] && 
		findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

