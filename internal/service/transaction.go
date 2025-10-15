package service

import (
	"context"
	"fmt"
	"time"

	"whatsapp-h2h-otomax/internal/config"
	"whatsapp-h2h-otomax/internal/model"
	"whatsapp-h2h-otomax/internal/repository"
	"whatsapp-h2h-otomax/pkg/logger"
)

// TransactionService handles transaction processing
type TransactionService struct {
	whatsappService *WhatsAppService
	repo            *repository.TransactionRepository
	ttl             time.Duration
	logger          *logger.Logger
}

// NewTransactionService creates a new transaction service
func NewTransactionService(waService *WhatsAppService, cfg *config.MessageTrackingConfig, log *logger.Logger) (*TransactionService, error) {
	// Initialize repository
	repo, err := repository.NewTransactionRepository(cfg.TrackingDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize transaction repository: %w", err)
	}

	service := &TransactionService{
		whatsappService: waService,
		repo:            repo,
		ttl:             cfg.TTL,
		logger:          log,
	}

	// Start cleanup goroutine
	go service.cleanupExpiredPeriodically()

	return service, nil
}

// Close closes the transaction service and database connection
func (s *TransactionService) Close() error {
	return s.repo.Close()
}

// ProcessTransaction processes transaction and sends to WhatsApp
func (s *TransactionService) ProcessTransaction(ctx context.Context, req *model.TransactionRequest) (*model.TransactionData, error) {
	// Check if transaction already exists (duplicate prevention)
	existingTrx, err := s.repo.GetByTrxID(req.TrxID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing transaction: %w", err)
	}
	if existingTrx != nil {
		return nil, fmt.Errorf("duplicate transaction: TrxID '%s' already exists and is still being tracked (sent at %s)", req.TrxID, existingTrx.SentAt.Format(time.RFC3339))
	}

	// Validate destination
	jid, destType, err := s.whatsappService.ValidateDestination(req.Destination)
	if err != nil {
		return nil, fmt.Errorf("invalid destination: %w", err)
	}

	// Format message
	message := req.Instructions;

	// Send message to WhatsApp
	messageID, err := s.whatsappService.SendMessage(ctx, jid, message)
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	// Save transaction to database
	now := time.Now()
	record := &repository.TransactionRecord{
		TrxID:           req.TrxID,
		MessageID:       messageID,
		Destination:     jid.String(),
		DestinationType: destType,
		SentAt:          now,
		ExpiresAt:       now.Add(s.ttl),
	}
	if err := s.repo.Save(record); err != nil {
		// Log error but don't fail the request (message already sent)
		s.logger.WithTrxID(req.TrxID).Error("Failed to save transaction to database", "error", err)
	}

	// Get current count for logging
	count, _ := s.repo.Count()

	// Log successful transaction
	s.logger.WithTrxID(req.TrxID).Info("Transaction sent",
		"destination", jid.String(),
		"type", destType,
		"message_id", messageID,
		"tracker_count", count,
	)

	return &model.TransactionData{
		TrxID:           req.TrxID,
		Destination:     jid.String(),
		DestinationType: destType,
		MessageID:       messageID,
		Timestamp:       now,
	}, nil
}

// formatMessage formats the transaction message
// func (s *TransactionService) formatMessage(req *model.TransactionRequest) string {
// 	return fmt.Sprintf(
// 		"🔔 TRANSAKSI BARU\n"+
// 			"━━━━━━━━━━━━━━━━\n"+
// 			"TRX ID: %s\n"+
// 			"Tujuan:\n%s\n\n"+
// 			"Catatan:\n%s\n",
// 		req.TrxID,
// 		req.Instructions,
// 		req.Descriptions,
// 	)
// }

// GetTransactionByDestination retrieves transaction info by destination JID
func (s *TransactionService) GetTransactionByDestination(destination string) (*repository.TransactionRecord, error) {
	return s.repo.GetByDestination(destination)
}

// GetRepository returns the transaction repository
func (s *TransactionService) GetRepository() *repository.TransactionRepository {
	return s.repo
}

// cleanupExpiredPeriodically runs cleanup every hour
func (s *TransactionService) cleanupExpiredPeriodically() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		count, err := s.repo.CleanupExpired()
		if err != nil {
			s.logger.Error("Failed to cleanup expired transactions", "error", err)
		} else if count > 0 {
			s.logger.Info("Cleaned up expired transactions", "count", count)
		}
	}
}
