package config

import (
	"encoding/json"
	"os"

	"github.com/ethereum/go-ethereum/common"
)

// ReserveConfig represents a single reserve configuration
type ReserveConfig struct {
	Token  string `json:"token"`
	Wallet string `json:"wallet"`
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
