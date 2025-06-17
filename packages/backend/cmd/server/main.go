package main

import (
	"log"
	"net/http"
	"os"

	// "os" // No longer needed if using config.Load() fully

	"github.com/Tarunshrma/proof-or-reserve/internal/api"
	"github.com/Tarunshrma/proof-or-reserve/internal/config"  // Will use config.Load()
	"github.com/Tarunshrma/proof-or-reserve/internal/service" // Added for service creation
	"github.com/Tarunshrma/proof-or-reserve/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env in the current directory
	// When running `go run ./cmd/server/main.go` from `packages/backend/`,
	// the current directory for loading .env should be `packages/backend/`.
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found in the current directory. Defaults will be used if env vars are not set elsewhere.")
	}

	// Initialize config using the centralized Load function
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

	// Initialize services
	blockchainSvc, err := service.NewBlockchainService(cfg.EthereumRPC, cfg.ContractAddress, cfg.AbiFilePath)
	if err != nil {
		log.Fatalf("Failed to create blockchain service: %v", err)
	}

	// Set private key if available (for sending transactions)
	if privateKey := os.Getenv("PRIVATE_KEY"); privateKey != "" {
		if err := blockchainSvc.SetPrivateKey(privateKey); err != nil {
			log.Printf("Warning: Failed to set private key: %v", err)
		}
	} else {
		log.Printf("Warning: No private key set. On-chain verification will not be available.")
	}

	// Initialize signature service (no longer needs private key)
	sigSvc := service.NewSignatureService(signatureStore)

	// Initialize API handlers
	handler := api.NewHandler(cfg, store, signatureStore, blockchainSvc, sigSvc)

	// Setup Gin router
	r := gin.Default()

	// Add CORS middleware (important for frontend integration)
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*") // For demo, allow all. For prod, restrict to frontend URL.
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization") // Add Authorization if you plan to use it
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent) // Use 204 No Content for OPTIONS preflight
			return
		}
		c.Next()
	})

	// Register routes
	r.GET("/assets/configured", handler.GetConfiguredAssets) // New route for serving static asset configurations

	// Contract configuration
	r.GET("/contract/config", handler.GetContractConfig)

	// Signature management
	r.POST("/signature/submit", handler.SubmitSignature)
	r.POST("/signature/txhash", handler.UpdateLastVerifiedTxHash)

	// Reserve details and verification
	r.GET("/reserve-details/:token/:wallet", handler.GetReserveDetails)
	r.POST("/initiate-onchain-verification", handler.InitiateOnchainVerification)

	// Existing/Old routes (kept for reference or other uses if any)
	r.GET("/reserveBalance/:token/:wallet", handler.GetReserveBalance) // Covered by /reserve-details
	r.GET("/signature/:token/:wallet", handler.GetSignature)
	// r.POST("/verifySignature", handler.VerifySignature) // Superseded by /initiate-onchain-verification
	// r.GET("/lastVerified/:token/:wallet", handler.GetLastVerified) // Covered by /reserve-details

	r.GET("/healthz", handler.Healthz)
	r.GET("/livez", handler.Livez)

	// Start server
	log.Printf("Starting server on %s", cfg.ListenAddr)
	if err := r.Run(cfg.ListenAddr); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
