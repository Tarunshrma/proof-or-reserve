# Proof of Reserve (PoR) Implementation

This monorepo contains the implementation of a Proof of Reserve system with three main components:

1. Smart Contracts
2. Backend Service
3. Frontend Application

## Project Structure

```
.
├── packages/
│   ├── contracts/         # Smart contract implementation
│   │   ├── contracts/     # Solidity smart contracts
│   │   ├── scripts/       # Deployment and other scripts
│   │   ├── test/         # Contract test files
│   │   └── hardhat.config.ts
│   │
│   ├── backend/          # Backend service
│   │   ├── src/          # Source code
│   │   ├── tests/        # Backend tests
│   │   └── package.json
│   │
│   ├── frontend/         # Frontend application
│   │   ├── src/          # Source code
│   │   ├── public/       # Public assets
│   │   └── package.json
│   │
│   └── common/           # Shared types and utilities
│       ├── src/
│       └── package.json
│
├── package.json          # Root package.json for workspace management
└── README.md            # This file
```

## Getting Started

[Instructions for setup and development will be added]

## License

[License information will be added]

# Proof of Reserve Backend

This is the backend service for the Proof of Reserve (PoR) system. It provides APIs for managing and verifying reserve proofs for token-wallet pairs.

## Features

- Signature generation and storage for token-wallet pairs
- On-chain proof verification through smart contract
- Real-time balance checking
- Last verification timestamp tracking
- Persistent storage using JSON files
- Reserve configuration management

## Prerequisites

- Go 1.21 or later
- Access to an XDC Apothem testnet node
- Private key for signing proofs
- Contract owner's private key for reserve configuration

## Configuration

### Environment Setup

The service can be configured using environment variables or a `.env` file:

```env
# Server configuration
LISTEN_ADDR=:8080

# Storage paths
STORAGE_PATH=data/storage.json
SIGNATURE_STORE_PATH=data/signatures.json

# Blockchain configuration
ETHEREUM_RPC=https://erpc.apothem.network
CONTRACT_ADDRESS=your_contract_address_here
PRIVATE_KEY=your_private_key_here
```

### Reserve Configuration

1. Create or modify `config/reserves.json` with your reserve details:

```json
{
  "reserves": [
    {
      "token": "0xYourTokenAddress",
      "wallet": "0xYourWalletAddress",
      "balance": "1000000000000000000"
    }
  ]
}
```

2. Run the setup script to configure reserves:
```bash
go run cmd/setup/main.go -config config/reserves.json
```

Note: 
- The account specified in PRIVATE_KEY must be the contract owner
- Ensure the account has enough XDC for gas
- Each reserve configuration requires a separate transaction

## API Endpoints

### GET /signature/:token/:wallet
Get or generate a signature for a token-wallet pair.

**Parameters:**
- `token`: Token contract address
- `wallet`: Wallet address to verify

**Response:**
```json
{
  "token": "0x...",
  "wallet": "0x...",
  "signature": "0x...",
  "validUntil": 1234567890,
  "generatedAt": "2024-03-21T12:34:56Z"
}
```

### POST /verifySignature
Verify a stored signature against the smart contract.

**Request Body:**
```json
{
  "token": "0x...",
  "wallet": "0x..."
}
```

**Response:**
```json
{
  "token": "0x...",
  "wallet": "0x...",
  "isValid": true,
  "validUntil": 1234567890,
  "signature": "0x..."
}
```

### GET /reserveBalance/:token/:wallet
Get the current balance of a reserve wallet.

**Parameters:**
- `token`: Token contract address
- `wallet`: Wallet address

**Response:**
```json
{
  "token": "0x...",
  "wallet": "0x...",
  "balance": "1000000000000000000"
}
```

### GET /lastVerified/:token/:wallet
Get the last verification timestamp for a token-wallet pair.

**Parameters:**
- `token`: Token contract address
- `wallet`: Wallet address

**Response:**
```json
{
  "token": "0x...",
  "wallet": "0x...",
  "lastVerified": 1234567890,
  "generatedAt": "2024-03-21T12:34:56Z"
}
```

## Development

1. Clone the repository:
```bash
git clone https://github.com/Tarunshrma/proof-or-reserve.git
cd proof-or-reserve/packages/backend
```

2. Install dependencies:
```bash
go mod download
```

3. Create a `.env` file with your configuration.

4. Configure reserves:
```bash
# Create or modify config/reserves.json with your reserve details
go run cmd/setup/main.go -config config/reserves.json
```

5. Run the server:
```bash
go run cmd/server/main.go
```

## Testing

Run the test suite:
```bash
go test ./...
```

## Troubleshooting Reserve Configuration

If reserve configuration fails, check:
1. The account has sufficient XDC balance for gas
2. The PRIVATE_KEY in .env belongs to the contract owner
3. The contract address is correct
4. The token and wallet addresses are valid
5. The network (Apothem testnet) is responsive

## License

This project is licensed under the MIT License - see the LICENSE file for details. 