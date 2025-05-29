package main

import (
	"log"

	"github.com/Tarunshrma/proof-or-reserve/internal/api"
	"github.com/Tarunshrma/proof-or-reserve/internal/config"
	"github.com/Tarunshrma/proof-or-reserve/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found")
	}

	// Initialize config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize storage
	store := storage.NewJSONStorage(cfg.StoragePath)
	if err := store.Load(); err != nil {
		log.Printf("Warning: Could not load existing storage: %v", err)
	}

	// Initialize API handlers
	handler := api.NewHandler(cfg, store)

	// Setup Gin router
	r := gin.Default()

	// Register routes
	r.GET("/reserveBalance/:token/:wallet", handler.GetReserveBalance)
	r.POST("/verifySignature", handler.VerifySignature)
	r.GET("/lastVerified/:token/:wallet", handler.GetLastVerified)

	// Start server
	log.Printf("Starting server on %s", cfg.ListenAddr)
	if err := r.Run(cfg.ListenAddr); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
