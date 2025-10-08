package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"whatsapp-h2h-otomax/internal/config"
	"whatsapp-h2h-otomax/internal/handler"
	"whatsapp-h2h-otomax/internal/middleware"
	"whatsapp-h2h-otomax/internal/service"
	"whatsapp-h2h-otomax/pkg/logger"
)

func main() {
	// Create .env from .env.example if not exists
	if err := ensureEnvFile(); err != nil {
		log.Printf("Warning: Failed to create .env file: %v", err)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	appLogger := logger.New(cfg.WhatsApp.LogLevel)
	appLogger.Info("Starting WhatsApp H2H Otomax service")

	// Initialize WhatsApp service
	whatsappService, err := service.NewWhatsAppService(&cfg.WhatsApp, appLogger)
	if err != nil {
		appLogger.Error("Failed to initialize WhatsApp service", "error", err)
		log.Fatalf("Failed to initialize WhatsApp service: %v", err)
	}

	// Initialize Otomax service
	otomaxService := service.NewOtomaxService(&cfg.Otomax, appLogger)

	// Initialize transaction service
	transactionService := service.NewTransactionService(whatsappService, &cfg.MessageTracking, appLogger)

	// Set dependencies
	whatsappService.SetOtomaxService(otomaxService)
	whatsappService.SetMessageTracker(transactionService.GetMessageTracker())

	// Connect to WhatsApp
	err = whatsappService.Connect()
	if err != nil {
		appLogger.Error("Failed to connect to WhatsApp", "error", err)
		log.Fatalf("Failed to connect to WhatsApp: %v\nPlease scan QR code first", err)
	}
	defer whatsappService.Disconnect()

	// Display joined groups
	ctx := context.Background()
	whatsappService.PrintJoinedGroups(ctx)

	// Initialize handlers
	transactionHandler := handler.NewTransactionHandler(transactionService, appLogger)
	webhookHandler := handler.NewWebhookHandler(cfg, appLogger)
	healthHandler := handler.NewHealthHandler(whatsappService, cfg, appLogger)
	groupsHandler := handler.NewGroupsHandler(whatsappService, appLogger)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(cfg.Security.APIKey, appLogger)

	// Setup HTTP routes
	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("/health", healthHandler.CheckHealth)

	// Protected routes
	mux.HandleFunc("/api/v1/forward", authMiddleware.Authenticate(transactionHandler.ForwardTransaction))
	mux.HandleFunc("/api/v1/webhook/message", authMiddleware.Authenticate(webhookHandler.ReceiveMessage))
	mux.HandleFunc("/api/v1/groups", authMiddleware.Authenticate(groupsHandler.ListGroups))

	// Create HTTP server
	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		appLogger.Info("HTTP server starting", "address", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.Error("HTTP server error", "error", err)
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	appLogger.Info("WhatsApp H2H Otomax service started successfully",
		"address", addr,
		"whatsapp_connected", whatsappService.IsConnected(),
	)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	appLogger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		appLogger.Error("Server forced to shutdown", "error", err)
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	appLogger.Info("Server stopped gracefully")
}

// ensureEnvFile creates .env from .env.example if .env doesn't exist
func ensureEnvFile() error {
	// Check if .env already exists
	if _, err := os.Stat(".env"); err == nil {
		return nil // .env already exists
	}

	// Check if .env.example exists
	if _, err := os.Stat(".env.example"); os.IsNotExist(err) {
		return fmt.Errorf(".env.example not found")
	}

	// Copy .env.example to .env
	source, err := os.Open(".env.example")
	if err != nil {
		return fmt.Errorf("failed to open .env.example: %w", err)
	}
	defer source.Close()

	destination, err := os.Create(".env")
	if err != nil {
		return fmt.Errorf("failed to create .env: %w", err)
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	if err != nil {
		return fmt.Errorf("failed to copy .env.example to .env: %w", err)
	}

	log.Println("✅ Created .env file from .env.example")
	return nil
}

