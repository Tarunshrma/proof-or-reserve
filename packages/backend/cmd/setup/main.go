package main

import (
	"flag"
	"log"
	"math/big"
	"os"

	"github.com/Tarunshrma/proof-or-reserve/internal/config"
	"github.com/Tarunshrma/proof-or-reserve/internal/service"
	"github.com/joho/godotenv"
)

func main() {
	configPath := flag.String("config", "config/reserves.json", "Path to reserve configuration file")
	flag.Parse()

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found")
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize blockchain service
	blockchainSvc, err := service.NewBlockchainService(cfg.EthereumRPC, cfg.ContractAddress, cfg.AbiFilePath)
	if err != nil {
		log.Fatalf("Failed to initialize blockchain service: %v", err)
	}

	// Set private key for transactions
	privateKey := os.Getenv("PRIVATE_KEY")
	if privateKey == "" {
		log.Fatal("PRIVATE_KEY environment variable is required")
	}
	if err := blockchainSvc.SetPrivateKey(privateKey); err != nil {
		log.Fatalf("Failed to set private key: %v", err)
	}

	// Load reserve configurations
	configs, err := config.LoadReserveConfigs(*configPath)
	if err != nil {
		log.Fatalf("Failed to load reserve configurations: %v", err)
	}

	log.Printf("Contract address: %s", cfg.ContractAddress)
	log.Printf("Loaded %d reserve configurations", len(configs.Reserves))

	// Configure each reserve
	for _, reserve := range configs.Reserves {
		log.Printf("Configuring reserve: token=%s, wallet=%s", reserve.Token, reserve.Wallet)

		// Check if already configured
		isConfigured, err := blockchainSvc.IsReserveWallet(reserve.Token, reserve.Wallet)
		if err != nil {
			log.Printf("Failed to check if reserve is configured: %v", err)
			continue
		}

		if isConfigured {
			log.Printf("Reserve already configured: token=%s, wallet=%s", reserve.Token, reserve.Wallet)
			continue
		}

		// Configure the reserve
		target := new(big.Int)
		target.SetString(reserve.Target, 10)
		if err := blockchainSvc.ConfigureReserve(reserve.Token, reserve.Wallet, target, reserve.ThresholdPercent); err != nil {
			log.Printf("Failed to configure reserve: %v", err)
			continue
		}

		log.Printf("Successfully configured reserve: token=%s, wallet=%s", reserve.Token, reserve.Wallet)
	}

	log.Printf("Setup complete")
}
