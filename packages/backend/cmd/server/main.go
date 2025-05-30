package main

import (
	"log"
	"os"
	"strings"

	"github.com/Tarunshrma/proof-or-reserve/internal/api"
	"github.com/Tarunshrma/proof-or-reserve/internal/config"
	"github.com/Tarunshrma/proof-or-reserve/internal/service"
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

	// Initialize storages
	store := storage.NewJSONStorage(cfg.StoragePath)
	if err := store.Load(); err != nil {
		log.Printf("Warning: Could not load existing storage: %v", err)
	}

	signatureStore := storage.NewSignatureStorage(cfg.SignatureStorePath)
	if err := signatureStore.Load(); err != nil {
		log.Printf("Warning: Could not load existing signatures: %v", err)
	}

	// Initialize signature service
	sigService, err := service.NewSignatureService(cfg.PrivateKey, signatureStore)
	if err != nil {
		log.Fatalf("Failed to initialize signature service: %v", err)
	}

	// Generate signatures for configured token-wallet pairs
	if tokenWalletPairs := os.Getenv("TOKEN_WALLET_PAIRS"); tokenWalletPairs != "" {
		pairs := strings.Split(tokenWalletPairs, ",")
		for _, pair := range pairs {
			parts := strings.Split(strings.TrimSpace(pair), ":")
			if len(parts) == 2 {
				token := strings.TrimSpace(parts[0])
				wallet := strings.TrimSpace(parts[1])
				if err := sigService.GenerateSignature(token, wallet); err != nil {
					log.Printf("Failed to generate signature for token %s and wallet %s: %v", token, wallet, err)
				} else {
					log.Printf("Generated signature for token %s and wallet %s", token, wallet)
				}
			}
		}
	}

	// Initialize API handlers with signature store
	handler := api.NewHandler(cfg, store, signatureStore)

	// Setup Gin router
	r := gin.Default()

	// Register routes
	r.GET("/reserveBalance/:token/:wallet", handler.GetReserveBalance)
	r.GET("/signature/:token/:wallet", handler.GetSignature)
	r.POST("/verifySignature", handler.VerifySignature)
	r.GET("/lastVerified/:token/:wallet", handler.GetLastVerified)

	// Start server
	log.Printf("Starting server on %s", cfg.ListenAddr)
	if err := r.Run(cfg.ListenAddr); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
