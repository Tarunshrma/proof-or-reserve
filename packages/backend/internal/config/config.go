package config

import (
	"fmt"
	"os"
)

type Config struct {
	ListenAddr         string
	StoragePath        string
	SignatureStorePath string
	EthereumRPC        string
	ContractAddress    string
	PrivateKey         string
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
