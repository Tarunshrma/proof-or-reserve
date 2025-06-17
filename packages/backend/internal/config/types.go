package config

import (
	"encoding/json"
	"os"

	"github.com/ethereum/go-ethereum/common"
)

// ReserveConfig represents a single reserve configuration
type ReserveConfig struct {
	Token            string `json:"token"`
	Wallet           string `json:"wallet"`
	Target           string `json:"target"`
	ThresholdPercent int    `json:"thresholdPercent"`
}

// ReserveConfigurations represents multiple reserve configurations
type ReserveConfigurations struct {
	Reserves []ReserveConfig `json:"reserves"`
}

// LoadReserveConfigs loads reserve configurations from a JSON file
func LoadReserveConfigs(filepath string) (*ReserveConfigurations, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	var configs ReserveConfigurations
	if err := json.Unmarshal(data, &configs); err != nil {
		return nil, err
	}

	return &configs, nil
}

// ToOnChainValues converts string values to their on-chain representation
func (rc *ReserveConfig) ToOnChainValues() (token, wallet common.Address, err error) {
	token = common.HexToAddress(rc.Token)
	wallet = common.HexToAddress(rc.Wallet)
	return token, wallet, nil
}

// ServerConfig holds settings for the server
type ServerConfig struct {
	ListenAddr string `env:"LISTEN_ADDR,default=:8082"`
}

// StorageConfig holds paths for data storage
type StorageConfig struct {
	StoragePath        string `env:"STORAGE_PATH,default=./data/reserves.json"`
	SignatureStorePath string `env:"SIGNATURE_STORE_PATH,default=./data/signatures.json"`
	AssetsConfigPath   string `env:"ASSETS_CONFIG_PATH,default=./data/reserves_config.json"`
}

// EthereumConfig holds settings for connecting to an Ethereum node
// ... rest of the file
