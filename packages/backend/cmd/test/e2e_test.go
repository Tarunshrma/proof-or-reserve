package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"
)

const (
	baseURL        = "http://localhost:8080"
	testTokenAddr  = "0xE99500AB4A413164DA49Af83B9824749059b46ce"
	testWalletAddr = "0xaf28621e287e4EA0F14FA7e7ba365206FD6279DA"
	validityPeriod = 24 * time.Hour
)

type SignatureResponse struct {
	Signature  string `json:"signature"`
	ValidUntil uint64 `json:"validUntil"`
}

type VerifyRequest struct {
	Token      string `json:"token"`
	Wallet     string `json:"wallet"`
	Signature  string `json:"signature"`
	ValidUntil uint64 `json:"validUntil"`
}

func TestEndToEndFlow(t *testing.T) {
	// Load environment variables
	err := godotenv.Load("../../.env")
	if err != nil {
		t.Logf("Warning: .env file not found: %v", err)
	}

	// 1. Test getting reserve balance
	t.Run("GetReserveBalance", func(t *testing.T) {
		url := fmt.Sprintf("%s/reserveBalance/%s/%s", baseURL, testTokenAddr, testWalletAddr)
		resp, err := http.Get(url)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var result struct {
			Balance string `json:"balance"`
		}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		t.Logf("Reserve balance: %s", result.Balance)
	})

	// 2. Test getting signature
	var signature SignatureResponse
	t.Run("GetSignature", func(t *testing.T) {
		url := fmt.Sprintf("%s/signature/%s/%s", baseURL, testTokenAddr, testWalletAddr)
		resp, err := http.Get(url)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		err = json.NewDecoder(resp.Body).Decode(&signature)
		require.NoError(t, err)
		require.NotEmpty(t, signature.Signature)
		require.Greater(t, signature.ValidUntil, uint64(time.Now().Unix()))
		t.Logf("Got signature: %s (valid until: %d)", signature.Signature, signature.ValidUntil)
	})

	// 3. Test verifying signature
	t.Run("VerifySignature", func(t *testing.T) {
		verifyReq := VerifyRequest{
			Token:      testTokenAddr,
			Wallet:     testWalletAddr,
			Signature:  signature.Signature,
			ValidUntil: signature.ValidUntil,
		}

		reqBody, err := json.Marshal(verifyReq)
		require.NoError(t, err)

		resp, err := http.Post(fmt.Sprintf("%s/verifySignature", baseURL), "application/json", bytes.NewBuffer(reqBody))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var result struct {
			Valid bool `json:"valid"`
		}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		require.True(t, result.Valid, "Signature verification failed")
	})

	// 4. Test getting last verified timestamp
	t.Run("GetLastVerified", func(t *testing.T) {
		url := fmt.Sprintf("%s/lastVerified/%s/%s", baseURL, testTokenAddr, testWalletAddr)
		resp, err := http.Get(url)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var result struct {
			LastVerified uint64 `json:"lastVerified"`
		}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		require.Greater(t, result.LastVerified, uint64(0))
		t.Logf("Last verified timestamp: %d", result.LastVerified)
	})
}
