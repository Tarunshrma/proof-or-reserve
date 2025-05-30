package main

import (
	"context"
	"log"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	// Connect to Ethereum client
	client, err := ethclient.Dial("https://erpc.apothem.network")
	if err != nil {
		log.Fatalf("Failed to connect to the Ethereum client: %v", err)
	}

	// Get contract code
	contractAddr := common.HexToAddress("0x08277B05B850Ab59b44dA5E734aa8568277352D1")
	code, err := client.CodeAt(context.Background(), contractAddr, nil)
	if err != nil {
		log.Fatalf("Failed to get contract code: %v", err)
	}

	if len(code) == 0 {
		log.Fatalf("No contract code found at address: %s", contractAddr.Hex())
	}

	log.Printf("Contract exists at %s with code length: %d", contractAddr.Hex(), len(code))
}
