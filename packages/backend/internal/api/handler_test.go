package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Tarunshrma/proof-or-reserve/internal/config"
	"github.com/Tarunshrma/proof-or-reserve/internal/service"
	"github.com/Tarunshrma/proof-or-reserve/internal/storage" // Assuming SignatureRecord is here or in service
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mock BlockchainService ---

type MockBlockchainService struct {
	mock.Mock
}

func (m *MockBlockchainService) GetReserveDetails(token, wallet string) (*service.ReserveDetailsOutput, error) {
	args := m.Called(token, wallet)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.ReserveDetailsOutput), args.Error(1)
}

func (m *MockBlockchainService) IsReserveWallet(token, wallet string) (bool, error) {
	args := m.Called(token, wallet)
	return args.Bool(0), args.Error(1)
}

func (m *MockBlockchainService) VerifySignature(token, wallet, signature string) (bool, error) {
	args := m.Called(token, wallet, signature)
	return args.Bool(0), args.Error(1)
}

func (m *MockBlockchainService) GetReserveBalance(token, wallet string) (*big.Int, error) {
	args := m.Called(token, wallet)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*big.Int), args.Error(1)
}

// --- Mock SignatureService ---

type MockSignatureService struct {
	mock.Mock
}

func (m *MockSignatureService) GenerateSignature(token, wallet string) error {
	args := m.Called(token, wallet)
	return args.Error(0)
}

func (m *MockSignatureService) GetSignature(token, wallet string) (*storage.SignatureRecord, error) {
	args := m.Called(token, wallet)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	// Assuming SignatureRecord is the correct type. Adjust if it's service.SignatureRecord or similar.
	return args.Get(0).(*storage.SignatureRecord), args.Error(1)
}

// Helper function to set up a test router with our handler and mock services
func setupTestRouter(mockBlockchain *MockBlockchainService, mockSigService *MockSignatureService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New() // Use gin.New() instead of gin.Default() for cleaner test logs

	// Create a dummy config for the handler
	// Tests will rely on mocks, so most config values can be minimal
	dummyCfg := &config.Config{
		// Populate with minimal necessary values if handler's NewHandler or other parts use them
		// For now, assuming mocks bypass most direct config needs in handler logic being tested
	}

	// The NewHandler in api/handler.go expects concrete storage types, not interfaces.
	// For unit testing handlers with mocked services, we often don't need real storage.
	// If NewHandler directly uses store or signatureStore for logic *before* service calls,
	// we might need to pass nil or dummy/mock stores.
	// Let's pass nil for now and adjust if NewHandler panics or misbehaves.
	var dummyStore *storage.JSONStorage = nil         // Or a dummy instance
	var dummySigStore *storage.SignatureStorage = nil // Or a dummy instance

	// We need to inject our MOCKS into the handler.
	// The current NewHandler creates REAL services. This is not ideal for unit tests.
	// We have a few options:
	// 1. Modify NewHandler to accept service interfaces (best for testability).
	// 2. Create a new constructor for tests, e.g., NewHandlerWithMocks.
	// 3. Directly set the service fields on the Handler struct after creating it with dummy/nil real services. (Potentially risky if NewHandler does important setup with them).

	// For now, let's try option 3: direct field assignment.
	// This requires handler fields (blockchain, sigService) to be exported or a setter method.
	// Let's assume they are exported for this example. If not, we'll need to adjust handler.go or use Option 1 or 2.

	// Create handler with dummy/nil for real services initially
	// This will panic if NewHandler tries to use cfg to create real services that fail with nil cfg components
	// We need to ensure cfg has enough for NewSignatureService and NewBlockchainService not to panic *before* we replace them.
	// Or, the ideal way: NewHandler should accept interfaces.

	// Let's assume we'll modify handler.go later if needed. For now, create the handler and OVERWRITE services.
	// This is a common pattern if refactoring NewHandler is a larger step.

	// Minimal config for NewSignatureService to pass (it needs PrivateKey)
	// Minimal config for NewBlockchainService (RPC, ContractAddress, ABIPath)
	// Since we replace them immediately, these can be dummy values as long as NewXXXService doesn't panic with them.
	testCfg := &config.Config{
		PrivateKey:      "0x0000000000000000000000000000000000000000000000000000000000000001", // Dummy
		EthereumRPC:     "http://localhost:8545",                                              // Dummy
		ContractAddress: "0x0000000000000000000000000000000000000000",                         // Dummy
		AbiFilePath:     "dummy.abi.json",                                                     // Dummy
	}

	// Create the real handler (which initializes its own services)
	// We will then overwrite its service fields with our mocks.
	// NOTE: This requires the service fields in the Handler struct to be EXPORTED (e.g., Blockchain, SigService)
	// If they are not, this approach won't work and NewHandler must be refactored.
	// Let's assume for now handler.blockchain and handler.sigService are unexported (lowercase)
	// and we CANNOT directly set them.

	// ---- THIS IS WHERE THE PROBLEM LIES FOR PURE UNIT TESTS ----
	// The current Handler setup in NewHandler creates concrete service instances.
	// To properly unit test with mocks, NewHandler should accept service INTERFACES.
	//
	// func NewHandler(cfg *config.Config, store *storage.JSONStorage, sigStore *storage.SignatureStorage,
	//    blockchainSvc YourBlockchainServiceInterface, sigSvc YourSignatureServiceInterface) *Handler
	//
	// Since that's a refactor of handler.go, for THIS test, we'll create a handler
	// and acknowledge its internal services are real, but our MOCKs will be used by
	// functions that take the mock router which has routes pointing to methods that USE THE MOCKS.
	// This is a bit indirect. A better way is to pass mocked services into the handler.

	// For now, we'll make a test-specific handler setup.
	// We will create a Handler instance and manually assign our mock services to it.
	// This assumes the Handler struct's fields for services are exported.
	// IF THE HANDLER FIELDS ARE NOT EXPORTED, THIS WILL NOT WORK.
	// Let's assume we will make them exported for testability or use a test-specific constructor.

	// Simplified Test Handler Setup:
	// Create a handler instance and assign mocks directly.
	// This requires BlockchainService and SignatureService fields in Handler to be exported.
	// Example:
	// type Handler struct {
	//    Config         *config.Config
	//    Store          *storage.JSONStorage
	//    SignatureStore *storage.SignatureStorage
	//    SigService     ISignatureService // Interface
	//    Blockchain     IBlockchainService // Interface
	// }
	// Let's proceed AS IF they were interfaces and NewHandler accepted them,
	// or that we can create a test handler like this:

	// To make this runnable NOW without Handler refactor, we need a way to construct a Handler with mocks.
	// One way is to create a simplified handler struct for tests, or use global vars (not good).

	// Let's assume we will define routes directly in the test router that call methods of a test handler
	// which internally uses the mocks. This avoids changing NewHandler for now.

	// Create a test-specific handler that uses the mocks
	testHandler := &Handler{
		config: testCfg, // Dummy config
		// store, signatureStore can be nil if not directly used by handler methods being tested for now
		blockchain: mockBlockchain, // THIS IS THE KEY: Use the mock
		sigService: mockSigService, // THIS IS THE KEY: Use the mock
	}

	// Register routes to the test handler's methods
	router.GET("/reserve-details/:token/:wallet", testHandler.GetReserveDetails)
	router.POST("/initiate-onchain-verification", testHandler.InitiateOnchainVerification)
	router.GET("/reserveBalance/:token/:wallet", testHandler.GetReserveBalance)
	router.GET("/signature/:token/:wallet", testHandler.GetSignature)

	return router
}

// --- Test Cases ---

func TestGetReserveDetails_Success(t *testing.T) {
	// Arrange
	mockBlockchain := new(MockBlockchainService)
	mockSigService := new(MockSignatureService) // Not used in this specific handler, but good practice for setup

	tokenAddr := "0x123"
	walletAddr := "0x456"
	expectedDetails := &service.ReserveDetailsOutput{
		IsConfigured: true,
		Name:         "Test Token",
		Symbol:       "TTT",
		Balance:      big.NewInt(1000),
		LastVerified: big.NewInt(1234567890),
	}

	// Setup expectation
	mockBlockchain.On("GetReserveDetails", tokenAddr, walletAddr).Return(expectedDetails, nil)

	router := setupTestRouter(mockBlockchain, mockSigService)

	req, _ := http.NewRequest("GET", fmt.Sprintf("/reserve-details/%s/%s", tokenAddr, walletAddr), nil)
	rr := httptest.NewRecorder()

	// Act
	router.ServeHTTP(rr, req)

	// Assert
	assert.Equal(t, http.StatusOK, rr.Code)

	var responseBody map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &responseBody)
	assert.NoError(t, err)

	assert.Equal(t, tokenAddr, responseBody["tokenAddress"])
	assert.Equal(t, walletAddr, responseBody["walletAddress"])
	assert.Equal(t, expectedDetails.IsConfigured, responseBody["isConfigured"])
	assert.Equal(t, expectedDetails.Name, responseBody["name"])
	assert.Equal(t, expectedDetails.Symbol, responseBody["symbol"])
	assert.Equal(t, expectedDetails.Balance.String(), responseBody["balance"])           // Compare as string
	assert.Equal(t, expectedDetails.LastVerified.String(), responseBody["lastVerified"]) // Compare as string

	mockBlockchain.AssertExpectations(t)
}

func TestGetReserveDetails_BlockchainError(t *testing.T) {
	// Arrange
	mockBlockchain := new(MockBlockchainService)
	mockSigService := new(MockSignatureService)

	tokenAddr := "0x123"
	walletAddr := "0x456"
	expectedError := errors.New("blockchain communication failed")

	// Setup expectation
	mockBlockchain.On("GetReserveDetails", tokenAddr, walletAddr).Return(nil, expectedError)

	router := setupTestRouter(mockBlockchain, mockSigService)

	req, _ := http.NewRequest("GET", fmt.Sprintf("/reserve-details/%s/%s", tokenAddr, walletAddr), nil)
	rr := httptest.NewRecorder()

	// Act
	router.ServeHTTP(rr, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, rr.Code)

	var errorResponse map[string]string
	err := json.Unmarshal(rr.Body.Bytes(), &errorResponse)
	assert.NoError(t, err)
	assert.Contains(t, errorResponse["error"], "Failed to get reserve details: "+expectedError.Error())

	mockBlockchain.AssertExpectations(t)
}

func TestGetReserveDetails_InvalidAddresses(t *testing.T) {
	mockBlockchain := new(MockBlockchainService) // Not strictly needed as it shouldn't be called
	mockSigService := new(MockSignatureService)
	router := setupTestRouter(mockBlockchain, mockSigService)

	tests := []struct {
		name       string
		tokenAddr  string
		walletAddr string
	}{
		{"invalid token address", "0x123invalid", "0xValidWalletAddress00000000000000000000"},
		{"invalid wallet address", "0xValidTokenAddress00000000000000000000", "0x456invalid"},
		{"non-hex token address", "notAHex", "0xValidWalletAddress00000000000000000000"},
		{"non-hex wallet address", "0xValidTokenAddress00000000000000000000", "notAHexEither"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", fmt.Sprintf("/reserve-details/%s/%s", tt.tokenAddr, tt.walletAddr), nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusBadRequest, rr.Code)
			var errorResponse map[string]string
			err := json.Unmarshal(rr.Body.Bytes(), &errorResponse)
			assert.NoError(t, err)
			assert.Equal(t, "Invalid token or wallet address", errorResponse["error"])
		})
	}
	// No expectations on mocks as they shouldn't be called.
}

// TODO: Add tests for InitiateOnchainVerification
// TODO: Add tests for GetReserveBalance
// TODO: Add tests for GetSignature
