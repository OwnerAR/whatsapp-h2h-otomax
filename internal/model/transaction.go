package model

import "time"

// TransactionRequest represents incoming transaction request from Otomax
type TransactionRequest struct {
	Destination  string `json:"destination"`
	TrxID        string `json:"trxid"`
	Descriptions string `json:"descriptions"`
	Instructions string `json:"instructions"`
}

// TransactionResponse represents response for transaction forwarding
type TransactionResponse struct {
	Status  string              `json:"status"`
	Message string              `json:"message"`
	Data    *TransactionData    `json:"data,omitempty"`
	Error   *TransactionError   `json:"error,omitempty"`
}

// TransactionData represents successful transaction data
type TransactionData struct {
	TrxID           string    `json:"trxid"`
	Destination     string    `json:"destination"`
	DestinationType string    `json:"destination_type"`
	MessageID       string    `json:"message_id"`
	Timestamp       time.Time `json:"timestamp"`
}

// TransactionError represents error response
type TransactionError struct {
	Code    string `json:"error_code"`
	Message string `json:"message"`
}

// TrackingInfo holds message tracking information
type TrackingInfo struct {
	MessageID       string
	TrxID           string
	Destination     string
	DestinationType string
	SentAt          time.Time
	ExpiresAt       time.Time
}

