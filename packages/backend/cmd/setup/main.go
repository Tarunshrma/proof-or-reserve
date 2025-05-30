package main

import (
	"context"
	"crypto/ecdsa"
	"flag"
	"log"
	"math/big"
	"os"
	"strings"

	"github.com/Tarunshrma/proof-or-reserve/internal/config"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
)

const contractABI = `[
    {"inputs":[],"stateMutability":"nonpayable","type":"constructor"},
    {"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"previousOwner","type":"address"},{"indexed":true,"internalType":"address","name":"newOwner","type":"address"}],"name":"OwnershipTransferred","type":"event"},
    {"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"token","type":"address"},{"indexed":true,"internalType":"address","name":"wallet","type":"address"},{"indexed":false,"internalType":"uint256","name":"validUntil","type":"uint256"},{"indexed":false,"internalType":"bool","name":"success","type":"bool"}],"name":"ProofVerified","type":"event"},
    {"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"token","type":"address"},{"indexed":true,"internalType":"address","name":"wallet","type":"address"}],"name":"ReserveConfigured","type":"event"},
    {"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"token","type":"address"},{"indexed":true,"internalType":"address","name":"wallet","type":"address"}],"name":"ReserveDeactivated","type":"event"},
    {"inputs":[{"internalType":"address","name":"token","type":"address"},{"internalType":"address","name":"wallet","type":"address"}],"name":"configureReserve","outputs":[],"stateMutability":"nonpayable","type":"function"},
    {"inputs":[{"internalType":"address","name":"token","type":"address"},{"internalType":"address","name":"wallet","type":"address"}],"name":"deactivateReserve","outputs":[],"stateMutability":"nonpayable","type":"function"},
    {"inputs":[{"internalType":"bytes32","name":"messageHash","type":"bytes32"}],"name":"getEthSignedMessageHash","outputs":[{"internalType":"bytes32","name":"","type":"bytes32"}],"stateMutability":"pure","type":"function"},
    {"inputs":[{"internalType":"address","name":"token","type":"address"},{"internalType":"address","name":"wallet","type":"address"}],"name":"getReserveBalance","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
    {"inputs":[{"internalType":"address","name":"token","type":"address"},{"internalType":"address","name":"wallet","type":"address"}],"name":"isProofValid","outputs":[{"internalType":"bool","name":"isValid","type":"bool"},{"internalType":"uint256","name":"validUntil","type":"uint256"}],"stateMutability":"view","type":"function"},
    {"inputs":[{"internalType":"address","name":"","type":"address"},{"internalType":"address","name":"","type":"address"}],"name":"isReserveWallet","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},
    {"inputs":[{"internalType":"address","name":"token","type":"address"},{"internalType":"address","name":"wallet","type":"address"}],"name":"lastVerifiedTimestamp","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
    {"inputs":[],"name":"owner","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
    {"inputs":[{"internalType":"bytes32","name":"ethSignedMessageHash","type":"bytes32"},{"internalType":"bytes","name":"signature","type":"bytes"}],"name":"recoverSigner","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"pure","type":"function"},
    {"inputs":[{"internalType":"address","name":"newOwner","type":"address"}],"name":"transferOwnership","outputs":[],"stateMutability":"nonpayable","type":"function"},
    {"inputs":[{"internalType":"address","name":"token","type":"address"},{"internalType":"address","name":"wallet","type":"address"}],"name":"validUntilTimestamp","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
    {"inputs":[{"internalType":"address","name":"token","type":"address"},{"internalType":"address","name":"wallet","type":"address"},{"internalType":"bytes","name":"signature","type":"bytes"},{"internalType":"uint256","name":"validUntil","type":"uint256"}],"name":"verifyProof","outputs":[{"internalType":"bool","name":"success","type":"bool"}],"stateMutability":"nonpayable","type":"function"}
]`

func main() {
	configPath := flag.String("config", "config/reserves.json", "Path to reserve configuration file")
	flag.Parse()

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found")
	}

	// Load private key from environment
	privateKeyHex := os.Getenv("PRIVATE_KEY")
	if privateKeyHex == "" {
		log.Fatal("PRIVATE_KEY environment variable is required")
	}

	// Connect to Ethereum client
	client, err := ethclient.Dial(os.Getenv("ETHEREUM_RPC"))
	if err != nil {
		log.Fatalf("Failed to connect to the Ethereum client: %v", err)
	}

	// Load private key
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		log.Fatalf("Failed to load private key: %v", err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("Failed to get public key")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	log.Printf("Using address: %s", fromAddress.Hex())

	// Get chain ID
	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		log.Fatalf("Failed to get chain ID: %v", err)
	}

	// Create auth
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		log.Fatalf("Failed to create authorized transactor: %v", err)
	}

	// Contract address from environment
	contractAddr := os.Getenv("CONTRACT_ADDRESS")
	if contractAddr == "" {
		log.Fatal("CONTRACT_ADDRESS environment variable is required")
	}
	contractAddress := common.HexToAddress(contractAddr)

	// Parse contract ABI
	parsedABI, err := abi.JSON(strings.NewReader(contractABI))
	if err != nil {
		log.Fatalf("Failed to parse contract ABI: %v", err)
	}

	// Create contract instance
	contract := bind.NewBoundContract(contractAddress, parsedABI, client, client, client)

	// Load reserve configurations
	configs, err := config.LoadReserveConfigs(*configPath)
	if err != nil {
		log.Fatalf("Failed to load reserve configurations: %v", err)
	}

	// Configure each reserve
	for _, reserve := range configs.Reserves {
		token, wallet, err := reserve.ToOnChainValues()
		if err != nil {
			log.Printf("Failed to parse reserve config: %v", err)
			continue
		}

		// Get nonce for this transaction
		nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
		if err != nil {
			log.Printf("Failed to get nonce: %v", err)
			continue
		}

		// Get gas price
		gasPrice, err := client.SuggestGasPrice(context.Background())
		if err != nil {
			log.Printf("Failed to get gas price: %v", err)
			continue
		}

		// Update auth for this transaction
		auth.Nonce = big.NewInt(int64(nonce))
		auth.Value = big.NewInt(0)      // in wei
		auth.GasLimit = uint64(3000000) // in units
		auth.GasPrice = gasPrice

		// Configure reserve
		tx, err := contract.Transact(auth, "configureReserve", token, wallet)
		if err != nil {
			log.Printf("Failed to configure reserve: %v", err)
			continue
		}

		log.Printf("Reserve configuration transaction sent: %s", tx.Hash().Hex())
		log.Printf("Waiting for transaction to be mined...")

		receipt, err := bind.WaitMined(context.Background(), client, tx)
		if err != nil {
			log.Printf("Failed to wait for transaction to be mined: %v", err)
			continue
		}

		if receipt.Status == 1 {
			log.Printf("Reserve configured successfully!")
			log.Printf("Token: %s", token.Hex())
			log.Printf("Wallet: %s", wallet.Hex())
		} else {
			log.Printf("Transaction failed!")
		}
	}
}
