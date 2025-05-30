#!/bin/bash

# Configuration
BASE_URL="http://localhost:8080"
TOKEN="0xE99500AB4A413164DA49Af83B9824749059b46ce"
WALLET="0xaf28621e287e4EA0F14FA7e7ba365206FD6279DA"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

# Function to test endpoint
test_endpoint() {
    local name=$1
    local method=$2
    local url=$3
    local payload=$4

    echo -e "\nTesting ${name}..."
    
    if [ "$method" = "GET" ]; then
        response=$(curl -s -w "\n%{http_code}" "$url")
    else
        response=$(curl -s -w "\n%{http_code}" -X "$method" -H "Content-Type: application/json" -d "$payload" "$url")
    fi

    status_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed \$d)

    if [ "$status_code" = "200" ]; then
        echo -e "${GREEN}✓ Success${NC} (Status: $status_code)"
        echo "Response: $body"
    else
        echo -e "${RED}✗ Failed${NC} (Status: $status_code)"
        echo "Response: $body"
    fi
}

echo "Starting endpoint tests..."

# 1. Test Get Reserve Balance
test_endpoint "Get Reserve Balance" "GET" "${BASE_URL}/reserveBalance/${TOKEN}/${WALLET}"

# 2. Test Get Signature
signature_response=$(curl -s "${BASE_URL}/signature/${TOKEN}/${WALLET}")
signature=$(echo $signature_response | jq -r '.signature')
valid_until=$(echo $signature_response | jq -r '.validUntil')

test_endpoint "Get Signature" "GET" "${BASE_URL}/signature/${TOKEN}/${WALLET}"

# 3. Test Verify Signature
verify_payload="{\"token\":\"${TOKEN}\",\"wallet\":\"${WALLET}\",\"signature\":\"${signature}\",\"validUntil\":${valid_until}}"
test_endpoint "Verify Signature" "POST" "${BASE_URL}/verifySignature" "$verify_payload"

# 4. Test Get Last Verified
test_endpoint "Get Last Verified" "GET" "${BASE_URL}/lastVerified/${TOKEN}/${WALLET}"

echo -e "\nEndpoint tests completed!" 