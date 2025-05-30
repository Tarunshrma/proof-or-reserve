#!/bin/bash

# Create .env.example file
cat > .env.example << EOL
# Server Configuration
LISTEN_ADDR=:8080

# Storage Configuration
STORAGE_PATH=data/storage.json
SIGNATURE_STORE_PATH=data/signatures.json

# Ethereum Configuration
ETHEREUM_RPC=http://localhost:8545
CONTRACT_ADDRESS=0x0000000000000000000000000000000000000000  # Replace with your deployed contract address

# Wallet Configuration
PRIVATE_KEY=0000000000000000000000000000000000000000000000000000000000000000  # Replace with your private key (without 0x prefix)

# Token-Wallet Pairs Configuration
# Format: token1:wallet1,token2:wallet2
# Example: 0x1234...5678:0xabcd...ef01,0x9876...5432:0xfedc...ba98
TOKEN_WALLET_PAIRS=

# Optional Configuration
# LOG_LEVEL=debug      # debug, info, warn, error
# ENABLE_METRICS=true  # Enable Prometheus metrics
# ENABLE_CORS=true     # Enable CORS for all origins
EOL

echo "Created .env.example file"
echo "To create your .env file, copy .env.example to .env and update the values:"
echo "cp .env.example .env" 