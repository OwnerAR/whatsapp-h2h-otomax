package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	waLog "go.mau.fi/whatsmeow/util/log"
	qrcode "github.com/skip2/go-qrcode"

	"whatsapp-h2h-otomax/internal/config"
	"whatsapp-h2h-otomax/internal/model"
	"whatsapp-h2h-otomax/internal/repository"
	"whatsapp-h2h-otomax/pkg/logger"
)

// WhatsAppService handles WhatsApp operations
type WhatsAppService struct {
	client            *whatsmeow.Client
	container         *sqlstore.Container
	logger            *logger.Logger
	otomaxService     *OtomaxService
	repo              *repository.TransactionRepository
	webhookWhitelist  []string
}

// NewWhatsAppService creates a new WhatsApp service
func NewWhatsAppService(cfg *config.WhatsAppConfig, log *logger.Logger) (*WhatsAppService, error) {
	ctx := context.Background()
	
	// Ensure database directory exists
	dbDir := filepath.Dir(cfg.DBPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}
	
	log.Info("Database directory ready", "path", dbDir)
	
	// Setup database for session storage
	container, err := sqlstore.New(ctx, "sqlite3", fmt.Sprintf("file:%s?_foreign_keys=on", cfg.DBPath), waLog.Noop)
	if err != nil {
		return nil, fmt.Errorf("failed to create store: %w", err)
	}

	// Get first device or create new one
	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	// Create WhatsApp client
	client := whatsmeow.NewClient(deviceStore, waLog.Noop)

	service := &WhatsAppService{
		client:    client,
		container: container,
		logger:    log,
	}

	return service, nil
}

// SetOtomaxService sets the Otomax service for webhook delivery
func (s *WhatsAppService) SetOtomaxService(otomaxService *OtomaxService) {
	s.otomaxService = otomaxService
}

// SetTransactionRepository sets the transaction repository for tracking
func (s *WhatsAppService) SetTransactionRepository(repo *repository.TransactionRepository) {
	s.repo = repo
}

// SetWebhookWhitelist sets the whitelist of JIDs/Groups allowed for webhook
func (s *WhatsAppService) SetWebhookWhitelist(whitelist []string) {
	s.webhookWhitelist = whitelist
}

// Connect connects to WhatsApp
func (s *WhatsAppService) Connect() error {
	// Check if we have a logged in session
	if s.client.Store.ID == nil {
		// No logged in session, need to pair with QR code
		s.logger.Info("No logged in session found, starting QR code pairing...")
		return s.connectWithEventLogin()
	}
	
	// We have a session, just connect
	s.logger.Info("Existing session found, connecting...")
	
	// Register event handler
	s.client.AddEventHandler(s.handleEvent)
	
	err := s.client.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	s.logger.Info("WhatsApp client connected successfully")
	return nil
}

// connectWithEventLogin is event-based login using QR code with GetQRChannel
func (s *WhatsAppService) connectWithEventLogin() error {
	s.logger.Info("Starting QR code login flow...")
	
	qrCount := 0
	
	// Get QR channel
	qrChan, err := s.client.GetQRChannel(context.Background())
	if err != nil {
		// If GetQRChannel not supported, use event-based fallback
		s.logger.Warn("QR channel not supported, using event-based login")
		return s.connectWithEventBasedLogin()
	}
	
	// Add event handler for connection events
	eventHandler := func(evt interface{}) {
		switch v := evt.(type) {
		case *events.PairSuccess:
			s.logger.Info("Pairing successful!", "jid", v.ID.String())
			fmt.Println("\n✅ Pairing successful! Finalizing connection...\n")
			
		case *events.Connected:
			s.logger.Info("Connection established successfully")
			fmt.Println("✅ Connected to WhatsApp successfully!\n")
			
		case *events.LoggedOut:
			s.logger.Error("Device logged out", "reason", v.Reason)
			fmt.Printf("\n❌ Device logged out: %s\n\n", v.Reason)
		}
	}
	
	s.client.AddEventHandler(eventHandler)
	
	// Connect to WhatsApp
	err = s.client.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to WhatsApp: %w", err)
	}
	
	s.logger.Info("Connected to WhatsApp, waiting for QR code...")
	
	// Process QR codes from channel
	for evt := range qrChan {
		if evt.Event == "code" {
			qrCount++
			
			// Generate QR code as PNG file
			qrFilename := "whatsapp-qrcode.png"
			err := qrcode.WriteFile(evt.Code, qrcode.Medium, 512, qrFilename)
			if err != nil {
				s.logger.Error("Failed to generate QR code PNG", "error", err)
				continue
			}
			
			// Get absolute path
			absPath, _ := os.Getwd()
			fullPath := fmt.Sprintf("%s/%s", absPath, qrFilename)
			
			if qrCount == 1 {
				s.logger.Info("QR Code generated successfully")
				fmt.Println("\n╔══════════════════════════════════════════════════════════════════╗")
				fmt.Println("║                   QR CODE GENERATED!                            ║")
				fmt.Println("╚══════════════════════════════════════════════════════════════════╝")
				fmt.Printf("\n📱 QR Code saved to: %s\n\n", fullPath)
				fmt.Println("🔹 Steps to link:")
				fmt.Println("   1. Open the QR code image file")
				fmt.Println("   2. Open WhatsApp on your phone")
				fmt.Println("   3. Go to: Settings > Linked Devices")
				fmt.Println("   4. Tap 'Link a Device'")
				fmt.Println("   5. Scan the QR code from the image file")
				fmt.Println("\n⏳ Waiting for scan... (Press Ctrl+C to cancel)")
				fmt.Println("   QR code will auto-refresh every ~20 seconds\n")
			} else {
				s.logger.Info("QR Code refreshed", "count", qrCount)
				fmt.Printf("🔄 QR Code refreshed (#%d)\n", qrCount)
			}
			
			s.logger.Info("QR code saved", "file", fullPath, "refresh_count", qrCount)
		} else {
			s.logger.Info("QR channel event", "event", evt.Event)
			// Channel closed, check if we're connected
			if s.client.IsLoggedIn() {
				s.logger.Info("Successfully logged in!")
				// Register main event handler
				s.client.AddEventHandler(s.handleEvent)
				return nil
			}
			// If not logged in and channel closed, return error
			if evt.Event == "timeout" {
				return fmt.Errorf("QR code scan timeout")
			}
			if evt.Event == "error" {
				return fmt.Errorf("QR code error: %v", evt.Error)
			}
		}
	}
	
	s.logger.Info("Login completed successfully")
	return nil
}

// connectWithEventBasedLogin is fallback for older whatsmeow versions
func (s *WhatsAppService) connectWithEventBasedLogin() error {
	s.logger.Info("Using event-based QR code login (legacy mode)...")
	
	loginChan := make(chan error, 1)
	qrCount := 0
	
	eventHandler := func(evt interface{}) {
		switch v := evt.(type) {
		case *events.QR:
			qrCount++
			qrCodeString := v.Codes[len(v.Codes)-1]
			
			qrFilename := "whatsapp-qrcode.png"
			err := qrcode.WriteFile(qrCodeString, qrcode.Medium, 512, qrFilename)
			if err != nil {
				s.logger.Error("Failed to generate QR code PNG", "error", err)
				return
			}
			
			absPath, _ := os.Getwd()
			fullPath := fmt.Sprintf("%s/%s", absPath, qrFilename)
			
			if qrCount == 1 {
				s.logger.Info("QR Code generated successfully")
				fmt.Println("\n╔══════════════════════════════════════════════════════════════════╗")
				fmt.Println("║                   QR CODE GENERATED!                            ║")
				fmt.Println("╚══════════════════════════════════════════════════════════════════╝")
				fmt.Printf("\n📱 QR Code saved to: %s\n\n", fullPath)
				fmt.Println("⏳ Waiting for scan...\n")
			} else {
				s.logger.Info("QR Code refreshed", "count", qrCount)
				fmt.Printf("🔄 QR Code refreshed (#%d)\n", qrCount)
			}
			
		case *events.PairSuccess:
			s.logger.Info("Pairing successful!", "jid", v.ID.String())
			fmt.Println("\n✅ Pairing successful!\n")
			
		case *events.Connected:
			s.logger.Info("Connection established successfully")
			fmt.Println("✅ Connected to WhatsApp successfully!\n")
			s.client.AddEventHandler(s.handleEvent)
			loginChan <- nil
			
		case *events.LoggedOut:
			s.logger.Error("Device logged out", "reason", v.Reason)
			loginChan <- fmt.Errorf("device logged out: %s", v.Reason)
		}
	}
	
	s.client.AddEventHandler(eventHandler)
	
	err := s.client.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	
	return <-loginChan
}

// Disconnect disconnects from WhatsApp
func (s *WhatsAppService) Disconnect() {
	s.client.Disconnect()
	s.logger.Info("WhatsApp client disconnected")
}

// IsConnected checks if client is connected
func (s *WhatsAppService) IsConnected() bool {
	return s.client.IsConnected()
}

// GetClient returns the WhatsApp client
func (s *WhatsAppService) GetClient() *whatsmeow.Client {
	return s.client
}

// ValidateDestination validates and parses destination (personal or group)
func (s *WhatsAppService) ValidateDestination(destination string) (types.JID, string, error) {
	// Check if it's a group JID
	if strings.Contains(destination, "@g.us") {
		jid, err := types.ParseJID(destination)
		if err != nil {
			return types.JID{}, "", fmt.Errorf("invalid group JID: %w", err)
		}

		// Verify group exists and bot is member
		_, err = s.client.GetGroupInfo(jid)
		if err != nil {
			return types.JID{}, "", fmt.Errorf("group not found or bot not a member: %w", err)
		}

		return jid, "group", nil
	}

	// Handle personal chat
	phone := s.normalizePhoneNumber(destination)
	if phone == "" {
		return types.JID{}, "", fmt.Errorf("invalid phone number format")
	}

	// Check if number is on WhatsApp
	resp, err := s.client.IsOnWhatsApp([]string{phone})
	if err != nil {
		return types.JID{}, "", fmt.Errorf("failed to check WhatsApp status: %w", err)
	}

	if len(resp) == 0 || !resp[0].IsIn {
		return types.JID{}, "", fmt.Errorf("phone number not registered on WhatsApp")
	}

	jid := types.NewJID(phone, types.DefaultUserServer)
	return jid, "personal", nil
}

// normalizePhoneNumber normalizes phone number to format 628xxx
func (s *WhatsAppService) normalizePhoneNumber(phone string) string {
	// Remove all non-digit characters
	re := regexp.MustCompile(`[^\d]`)
	phone = re.ReplaceAllString(phone, "")

	// Remove leading zeros
	phone = strings.TrimLeft(phone, "0")

	// Add 62 prefix if not present
	if !strings.HasPrefix(phone, "62") {
		phone = "62" + phone
	}

	// Validate Indonesian phone number (basic validation)
	if len(phone) < 11 || len(phone) > 15 {
		return ""
	}

	return phone
}

// SendMessage sends a text message to WhatsApp
func (s *WhatsAppService) SendMessage(ctx context.Context, to types.JID, text string) (string, error) {
	if !s.IsConnected() {
		return "", fmt.Errorf("WhatsApp client not connected")
	}

	message := &waProto.Message{
		Conversation: &text,
	}

	resp, err := s.client.SendMessage(ctx, to, message)
	if err != nil {
		return "", fmt.Errorf("failed to send message: %w", err)
	}

	return resp.ID, nil
}

// handleEvent handles WhatsApp events
func (s *WhatsAppService) handleEvent(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		s.handleIncomingMessage(v)
	case *events.Connected:
		s.logger.Info("WhatsApp client connected")
	case *events.Disconnected:
		s.logger.Warn("WhatsApp client disconnected")
	}
}

// handleIncomingMessage handles incoming WhatsApp messages
func (s *WhatsAppService) handleIncomingMessage(evt *events.Message) {
	// Ignore our own messages
	if evt.Info.IsFromMe {
		return
	}

	// Check if repository is set
	if s.repo == nil {
		return
	}

	// Check whitelist if configured
	chatJID := evt.Info.Chat.String()
	if len(s.webhookWhitelist) > 0 {
		if !s.isWhitelisted(chatJID) {
			s.logger.Info("Message from non-whitelisted JID ignored",
				"jid", chatJID,
			)
			return
		}
	}

	// Get tracking info for this chat from database
	trackingRecord, err := s.repo.GetByDestination(chatJID)
	if err != nil {
		s.logger.Error("Failed to get tracking info", "error", err, "jid", chatJID)
		return
	}
	if trackingRecord == nil {
		// Not related to any tracked transaction
		return
	}

	// Extract message content
	messageContent := ""
	messageType := "text"

	if evt.Message.Conversation != nil {
		messageContent = *evt.Message.Conversation
	} else if evt.Message.ExtendedTextMessage != nil {
		messageContent = *evt.Message.ExtendedTextMessage.Text
	}

	// Build webhook payload
	payload := &model.WebhookPayload{
		Event: "message_received",
		Sender: model.Sender{
			Phone: evt.Info.Sender.User,
			Name:  evt.Info.PushName,
		},
		Message: model.MessageContent{
			Type:      messageType,
			Content:   messageContent,
			Timestamp: evt.Info.Timestamp,
		},
		Context: model.MessageContext{
			ChatType:          trackingRecord.DestinationType,
			IsReply:           false,
			OriginalMessageID: trackingRecord.MessageID,
		},
	}

	// Extract quoted message content if this is a reply
	if evt.Message.ExtendedTextMessage != nil && 
	   evt.Message.ExtendedTextMessage.ContextInfo != nil {
		
		payload.Context.IsReply = true
		
		// Get quoted message ID (use StanzaID field)
		if evt.Message.ExtendedTextMessage.ContextInfo.StanzaID != nil {
			payload.Context.OriginalMessageID = *evt.Message.ExtendedTextMessage.ContextInfo.StanzaID
		}
		
		// Get quoted message content
		if evt.Message.ExtendedTextMessage.ContextInfo.QuotedMessage != nil {
			quotedMsg := evt.Message.ExtendedTextMessage.ContextInfo.QuotedMessage
			
			if quotedMsg.Conversation != nil {
				payload.Context.QuotedMessageContent = *quotedMsg.Conversation
			} else if quotedMsg.ExtendedTextMessage != nil && quotedMsg.ExtendedTextMessage.Text != nil {
				payload.Context.QuotedMessageContent = *quotedMsg.ExtendedTextMessage.Text
			}
		}
	}

	// Send to Otomax webhook
	if s.otomaxService != nil {
		ctx := context.Background()
		err := s.otomaxService.SendWebhook(ctx, payload, trackingRecord.TrxID)
		if err != nil {
			s.logger.WithTrxID(trackingRecord.TrxID).Error("Failed to send webhook",
				"error", err,
				"from", evt.Info.Sender.User,
			)
			return
		}
		
		// Log successful webhook delivery
		s.logger.WithTrxID(trackingRecord.TrxID).Info("Message received and forwarded to webhook",
			"from", evt.Info.Sender.User,
			"message", messageContent,
		)
	}
}

// isWhitelisted checks if JID is in whitelist
func (s *WhatsAppService) isWhitelisted(jid string) bool {
	for _, whitelisted := range s.webhookWhitelist {
		if whitelisted == jid {
			return true
		}
	}
	return false
}

// GetConnectionStatus returns connection status information
func (s *WhatsAppService) GetConnectionStatus() map[string]interface{} {
	status := map[string]interface{}{
		"connected": s.IsConnected(),
	}

	if s.client.Store.ID != nil {
		status["phone"] = s.client.Store.ID.User
		status["device"] = "whatsapp-h2h-otomax"
	}

	return status
}

// GetJoinedGroups retrieves all groups that the bot is a member of
func (s *WhatsAppService) GetJoinedGroups(ctx context.Context) ([]*types.GroupInfo, error) {
	if !s.IsConnected() {
		return nil, fmt.Errorf("WhatsApp client not connected")
	}

	groups, err := s.client.GetJoinedGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get joined groups: %w", err)
	}

	return groups, nil
}

// PrintJoinedGroups displays all joined groups to console
func (s *WhatsAppService) PrintJoinedGroups(ctx context.Context) {
	groups, err := s.GetJoinedGroups(ctx)
	if err != nil {
		s.logger.Error("Failed to get joined groups", "error", err)
		return
	}

	if len(groups) == 0 {
		fmt.Println("\n📋 No groups found. Bot is not a member of any group.")
		return
	}

	fmt.Println("\n╔══════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                        JOINED WHATSAPP GROUPS                            ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════╝")
	fmt.Printf("\nTotal: %d group(s)\n\n", len(groups))

	for i, group := range groups {
		fmt.Printf("%d. %s\n", i+1, group.Name)
		fmt.Printf("   JID: %s\n", group.JID.String())
		fmt.Printf("   Participants: %d\n", len(group.Participants))
		
		// Get group info for more details
		groupInfo, err := s.client.GetGroupInfo(group.JID)
		if err == nil {
			if groupInfo.Topic != "" {
				fmt.Printf("   Topic: %s\n", groupInfo.Topic)
			}
			if groupInfo.IsAnnounce {
				fmt.Printf("   Type: Announcement Only\n")
			}
		}
		fmt.Println()
	}

	fmt.Println("╔══════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║  Copy the JID above to use as 'destination' parameter in API requests   ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════╝\n")

	s.logger.Info("Group list displayed", "total_groups", len(groups))
}

