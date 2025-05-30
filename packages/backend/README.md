# Proof of Reserve Backend

This is the backend service for the Proof of Reserve (PoR) system. It provides APIs for managing and verifying reserve proofs for token-wallet pairs.

## Features

- Signature generation and storage for token-wallet pairs
- On-chain proof verification through smart contract
- Real-time balance checking
- Last verification timestamp tracking
- Persistent storage using JSON files

## Prerequisites

- Go 1.21 or later
- Access to an XDC Apothem testnet node
- Private key for signing proofs

## Configuration

The service can be configured using environment variables or a `.env` file:

```env
# Server configuration
LISTEN_ADDR=:8080

# Storage paths
STORAGE_PATH=data/storage.json
SIGNATURE_STORE_PATH=data/signatures.json

# Blockchain configuration
ETHEREUM_RPC=https://erpc.apothem.network
CONTRACT_ADDRESS=0x08277B05B850Ab59b44dA5E734aa8568277352D1
PRIVATE_KEY=your_private_key_here
```

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

4. Run the server:
```bash
go run cmd/server/main.go
```

## Testing

Run the test suite:
```bash
go test ./...
```

## License

This project is licensed under the MIT License - see the LICENSE file for details. 