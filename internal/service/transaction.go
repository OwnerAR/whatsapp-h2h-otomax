package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"whatsapp-h2h-otomax/internal/config"
	"whatsapp-h2h-otomax/internal/model"
	"whatsapp-h2h-otomax/pkg/logger"
)

// TransactionService handles transaction processing
type TransactionService struct {
	whatsappService *WhatsAppService
	messageTracker  *MessageTracker
	logger          *logger.Logger
}

// NewTransactionService creates a new transaction service
func NewTransactionService(waService *WhatsAppService, cfg *config.MessageTrackingConfig, log *logger.Logger) *TransactionService {
	return &TransactionService{
		whatsappService: waService,
		messageTracker:  NewMessageTracker(cfg.TTL),
		logger:          log,
	}
}

// ProcessTransaction processes transaction and sends to WhatsApp
func (s *TransactionService) ProcessTransaction(ctx context.Context, req *model.TransactionRequest) (*model.TransactionData, error) {
	// Validate destination
	jid, destType, err := s.whatsappService.ValidateDestination(req.Destination)
	if err != nil {
		return nil, fmt.Errorf("invalid destination: %w", err)
	}

	// Format message
	message := s.formatMessage(req)

	// Send message to WhatsApp
	messageID, err := s.whatsappService.SendMessage(ctx, jid, message)
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	// Track message for reply handling
	trackingInfo := &model.TrackingInfo{
		MessageID:       messageID,
		TrxID:           req.TrxID,
		Destination:     jid.String(),
		DestinationType: destType,
		SentAt:          time.Now(),
	}
	s.messageTracker.Track(trackingInfo)

	// Log successful transaction
	s.logger.WithTrxID(req.TrxID).Info("Transaction sent",
		"destination", jid.String(),
		"type", destType,
		"message_id", messageID,
		"tracker_count", s.messageTracker.Count(),
	)

	return &model.TransactionData{
		TrxID:           req.TrxID,
		Destination:     jid.String(),
		DestinationType: destType,
		MessageID:       messageID,
		Timestamp:       time.Now(),
	}, nil
}

// formatMessage formats the transaction message
func (s *TransactionService) formatMessage(req *model.TransactionRequest) string {
	return fmt.Sprintf(
		"🔔 TRANSAKSI BARU\n"+
			"━━━━━━━━━━━━━━━━\n"+
			"TRX ID: %s\n"+
			"Tujuan:\n%s\n\n"+
			"Catatan:\n%s\n\n"+
		req.TrxID,
		req.Instructions,
		req.Descriptions,
	)
}

// GetMessageTracker returns the message tracker
func (s *TransactionService) GetMessageTracker() *MessageTracker {
	return s.messageTracker
}

// MessageTracker handles message tracking for reply mapping
type MessageTracker struct {
	cache map[string]*model.TrackingInfo
	mu    sync.RWMutex
	ttl   time.Duration
}

// NewMessageTracker creates a new message tracker
func NewMessageTracker(ttl time.Duration) *MessageTracker {
	tracker := &MessageTracker{
		cache: make(map[string]*model.TrackingInfo),
		ttl:   ttl,
	}

	// Start cleanup goroutine
	go tracker.cleanupExpiredPeriodically()

	return tracker
}

// Track saves tracking information
func (mt *MessageTracker) Track(info *model.TrackingInfo) {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	info.ExpiresAt = time.Now().Add(mt.ttl)
	mt.cache[info.MessageID] = info
}

// GetByChat retrieves tracking info by chat JID
func (mt *MessageTracker) GetByChat(chatJID string) *model.TrackingInfo {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	now := time.Now()
	for _, info := range mt.cache {
		if info.Destination == chatJID && now.Before(info.ExpiresAt) {
			return info
		}
	}
	return nil
}

// GetByMessageID retrieves tracking info by message ID
func (mt *MessageTracker) GetByMessageID(messageID string) *model.TrackingInfo {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	if info, exists := mt.cache[messageID]; exists {
		if time.Now().Before(info.ExpiresAt) {
			return info
		}
	}
	return nil
}

// GetByTrxID retrieves tracking info by transaction ID
func (mt *MessageTracker) GetByTrxID(trxID string) *model.TrackingInfo {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	now := time.Now()
	for _, info := range mt.cache {
		if info.TrxID == trxID && now.Before(info.ExpiresAt) {
			return info
		}
	}
	return nil
}

// CleanupExpired removes expired tracking entries
func (mt *MessageTracker) CleanupExpired() int {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	now := time.Now()
	count := 0

	for msgID, info := range mt.cache {
		if now.After(info.ExpiresAt) {
			delete(mt.cache, msgID)
			count++
		}
	}

	return count
}

// cleanupExpiredPeriodically runs cleanup every hour
func (mt *MessageTracker) cleanupExpiredPeriodically() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		count := mt.CleanupExpired()
		if count > 0 {
			// Could log this if needed
		}
	}
}

// Count returns the number of tracked messages
func (mt *MessageTracker) Count() int {
	mt.mu.RLock()
	defer mt.mu.RUnlock()
	return len(mt.cache)
}

