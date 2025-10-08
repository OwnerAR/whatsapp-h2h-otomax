package model

import "time"

// IncomingMessage represents message received from WhatsApp
type IncomingMessage struct {
	MessageID       string    `json:"message_id"`
	TrxID           string    `json:"trxid"`
	From            string    `json:"from"`
	FromName        string    `json:"from_name"`
	ChatJID         string    `json:"chat_jid"`
	ChatType        string    `json:"chat_type"`
	MessageType     string    `json:"message_type"`
	Message         string    `json:"message"`
	Timestamp       time.Time `json:"timestamp"`
	QuotedMessageID string    `json:"quoted_message_id,omitempty"`
}

// WebhookPayload represents payload sent to Otomax webhook
type WebhookPayload struct {
	Event   string         `json:"event"`
	Sender  Sender         `json:"sender"`
	Message MessageContent `json:"message"`
	Context MessageContext `json:"context"`
}

// Sender represents message sender information
type Sender struct {
	Phone string `json:"phone"`
	Name  string `json:"name"`
}

// MessageContent represents message content
type MessageContent struct {
	Type      string    `json:"type"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	MediaURL  string    `json:"media_url,omitempty"`
}

// MessageContext represents message context
type MessageContext struct {
	ChatType             string `json:"chat_type"`
	IsReply              bool   `json:"is_reply"`
	OriginalMessageID    string `json:"original_message_id,omitempty"`
	QuotedMessageContent string `json:"quoted_message_content,omitempty"` // Content dari message yang di-reply
}

// WebhookResponse represents response from Otomax webhook
type WebhookResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

