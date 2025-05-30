# Proof of Reserve Backend

This is the backend service for the Proof of Reserve (PoR) system. It handles signature generation, verification, and reserve management for the PoR smart contract.

## Prerequisites

- Go 1.21 or later
- Access to XDC Apothem testnet
- Private key with XDC test tokens

## Setup

1. **Environment Variables**

Create a `.env` file in the root directory:
```env
# Server Configuration
LISTEN_ADDR=:8080

# Storage Configuration
STORAGE_PATH=data/storage.json
SIGNATURE_STORE_PATH=data/signatures.json

# XDC Apothem Network Configuration
ETHEREUM_RPC=https://erpc.apothem.network
CONTRACT_ADDRESS=0x08277B05B850Ab59b44dA5E734aa8568277352D1

# Wallet Configuration
PRIVATE_KEY=your_private_key_here  # Without 0x prefix
```

2. **Reserve Configuration**

Create a `config/reserves.json` file:
```json
{
  "reserves": [
    {
      "token": "0xE99500AB4A413164DA49Af83B9824749059b46ce",
      "wallet": "0xaf28621e287e4EA0F14FA7e7ba365206FD6279DA",
      "balance": "10"
    }
  ]
}
```

3. **Install Dependencies**
```bash
go mod tidy
```

## Running the Service

1. **Start the Server**
```bash
go run cmd/server/main.go
```

2. **Configure Reserves**
```bash
go run cmd/setup/main.go --config config/reserves.json
```

## API Endpoints

### 1. Get Signature
```bash
curl -X GET "http://localhost:8080/signature/{token}/{wallet}"

# Example
curl -X GET "http://localhost:8080/signature/0xE99500AB4A413164DA49Af83B9824749059b46ce/0xaf28621e287e4EA0F14FA7e7ba365206FD6279DA"
```

### 2. Verify Signature
```bash
curl -X POST "http://localhost:8080/verifySignature" \
  -H "Content-Type: application/json" \
  -d '{
    "token": "0xE99500AB4A413164DA49Af83B9824749059b46ce",
    "wallet": "0xaf28621e287e4EA0F14FA7e7ba365206FD6279DA"
  }'
```

### 3. Get Last Verification
```bash
curl -X GET "http://localhost:8080/lastVerified/{token}/{wallet}"

# Example
curl -X GET "http://localhost:8080/lastVerified/0xE99500AB4A413164DA49Af83B9824749059b46ce/0xaf28621e287e4EA0F14FA7e7ba365206FD6279DA"
```

### 4. Get Reserve Balance
```bash
curl -X GET "http://localhost:8080/reserveBalance/{token}/{wallet}"

# Example
curl -X GET "http://localhost:8080/reserveBalance/0xE99500AB4A413164DA49Af83B9824749059b46ce/0xaf28621e287e4EA0F14FA7e7ba365206FD6279DA"
```

## Response Formats

### Signature Response
```json
{
  "token": "0xE99500AB4A413164DA49Af83B9824749059b46ce",
  "wallet": "0xaf28621e287e4EA0F14FA7e7ba365206FD6279DA",
  "signature": "0x...",
  "validUntil": 1234567890,
  "generatedAt": "2024-05-29T12:00:00Z"
}
```

### Verification Response
```json
{
  "token": "0xE99500AB4A413164DA49Af83B9824749059b46ce",
  "wallet": "0xaf28621e287e4EA0F14FA7e7ba365206FD6279DA",
  "balance": "10",
  "timestamp": "2024-05-29T12:00:00Z",
  "signature": "0x..."
}
```

### Balance Response
```json
{
  "token": "0xE99500AB4A413164DA49Af83B9824749059b46ce",
  "wallet": "0xaf28621e287e4EA0F14FA7e7ba365206FD6279DA",
  "balance": "10"
}
```

## Error Handling

The API returns appropriate HTTP status codes:
- 200: Success
- 400: Bad Request (invalid input)
- 404: Not Found (signature/verification not found)
- 500: Internal Server Error

Error responses include a message:
```json
{
  "error": "Error message here"
}
```

## Storage

The service uses two JSON files for persistence:
- `data/storage.json`: Stores verification records
- `data/signatures.json`: Stores generated signatures

These files are created automatically when needed.

## Development

### Project Structure
```
packages/backend/
├── cmd/
│   ├── server/     # Main server application
│   └── setup/      # Reserve configuration tool
├── internal/
│   ├── api/        # API handlers
│   ├── config/     # Configuration types
│   ├── service/    # Business logic
│   └── storage/    # Data persistence
├── config/         # Configuration files
├── data/          # Storage files
└── scripts/       # Utility scripts
```

### Adding New Features

1. Add new types in `internal/config/types.go`
2. Implement business logic in `internal/service/`
3. Add API handlers in `internal/api/handler.go`
4. Update storage if needed in `internal/storage/`

## Testing

Run all tests:
```bash
go test ./...
```

## Troubleshooting

1. **Server won't start**
   - Check if `.env` file exists and has correct values
   - Ensure required ports are available

2. **Signature verification fails**
   - Verify contract address is correct
   - Check if reserve is properly configured
   - Ensure private key has permission to sign

3. **Storage errors**
   - Check if `data/` directory exists and is writable
   - Verify JSON files are properly formatted 