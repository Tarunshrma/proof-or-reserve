package main

import (
	"log"
	"os"

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
	cfg := &config.Config{
		ListenAddr:         os.Getenv("LISTEN_ADDR"),
		StoragePath:        os.Getenv("STORAGE_PATH"),
		SignatureStorePath: os.Getenv("SIGNATURE_STORE_PATH"),
		EthereumRPC:        os.Getenv("ETHEREUM_RPC"),
		ContractAddress:    os.Getenv("CONTRACT_ADDRESS"),
		PrivateKey:         os.Getenv("PRIVATE_KEY"),
	}

	// Set defaults if not provided
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = ":8080"
	}
	if cfg.StoragePath == "" {
		cfg.StoragePath = "data/storage.json"
	}
	if cfg.SignatureStorePath == "" {
		cfg.SignatureStorePath = "data/signatures.json"
	}
	if cfg.EthereumRPC == "" {
		cfg.EthereumRPC = "https://erpc.apothem.network"
	}

	// Validate required config
	if cfg.ContractAddress == "" {
		log.Fatal("CONTRACT_ADDRESS is required")
	}
	if cfg.PrivateKey == "" {
		log.Fatal("PRIVATE_KEY is required")
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

	// Initialize API handlers
	handler := api.NewHandler(cfg, store, signatureStore)

	// Setup Gin router
	r := gin.Default()

	// Register routes
	r.GET("/reserveBalance/:token/:wallet", handler.GetReserveBalance)
	r.GET("/signature/:token/:wallet", handler.GetSignature)
	r.POST("/verifySignature", handler.VerifySignature)
	r.GET("/lastVerified/:token/:wallet", handler.GetLastVerified)

	// Add CORS middleware if needed
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Start server
	log.Printf("Starting server on %s", cfg.ListenAddr)
	if err := r.Run(cfg.ListenAddr); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
