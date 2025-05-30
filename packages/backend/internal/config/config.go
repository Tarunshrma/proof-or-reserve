package config

import (
	"fmt"
	"os"
)

// Config holds all configuration for the application
type Config struct {
	// ListenAddr is the address and port the server will listen on
	ListenAddr string

	// StoragePath is the path to the JSON storage file
	StoragePath string

	// SignatureStorePath is the path to the signature storage file
	SignatureStorePath string

	// EthereumRPC is the URL of the Ethereum RPC endpoint
	EthereumRPC string

	// ContractAddress is the address of the deployed ProofOfReserve contract
	ContractAddress string

	// PrivateKey is the private key used for signing proofs
	PrivateKey string
}

func Load() (*Config, error) {
	cfg := &Config{
		ListenAddr:         getEnvOrDefault("LISTEN_ADDR", ":8080"),
		StoragePath:        getEnvOrDefault("STORAGE_PATH", "data/storage.json"),
		SignatureStorePath: getEnvOrDefault("SIGNATURE_STORE_PATH", "data/signatures.json"),
		EthereumRPC:        os.Getenv("ETHEREUM_RPC"),
	}

	// Required fields
	cfg.ContractAddress = os.Getenv("CONTRACT_ADDRESS")
	if cfg.ContractAddress == "" {
		return nil, fmt.Errorf("CONTRACT_ADDRESS environment variable is required")
	}

	cfg.PrivateKey = os.Getenv("PRIVATE_KEY")
	if cfg.PrivateKey == "" {
		return nil, fmt.Errorf("PRIVATE_KEY environment variable is required")
	}

	return cfg, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
