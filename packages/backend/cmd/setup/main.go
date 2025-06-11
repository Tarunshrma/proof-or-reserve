package main

import (
	"flag"
	"log"
	"os"

	"github.com/Tarunshrma/proof-or-reserve/internal/config"
	"github.com/ethereum/go-ethereum/common"
	"github.com/joho/godotenv"
)

func main() {
	configPath := flag.String("config", "config/reserves.json", "Path to reserve configuration file")
	flag.Parse()

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found")
	}

	// Contract address from environment
	contractAddr := os.Getenv("CONTRACT_ADDRESS")
	if contractAddr == "" {
		log.Fatal("CONTRACT_ADDRESS environment variable is required")
	}
	contractAddress := common.HexToAddress(contractAddr)

	// Load reserve configurations
	configs, err := config.LoadReserveConfigs(*configPath)
	if err != nil {
		log.Fatalf("Failed to load reserve configurations: %v", err)
	}

	log.Printf("Contract address: %s", contractAddress.Hex())
	log.Printf("Loaded %d reserve configurations", len(configs.Reserves))
	log.Printf("Please use the frontend to submit signatures for each reserve wallet")
}
