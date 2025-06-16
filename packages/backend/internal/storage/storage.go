package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// JSONStorage provides a simple JSON-based storage implementation
type JSONStorage struct {
	path     string
	data     map[string]interface{}
	mu       sync.RWMutex
	modified bool
}

// NewJSONStorage creates a new JSONStorage instance
func NewJSONStorage(path string) *JSONStorage {
	return &JSONStorage{
		path: path,
		data: make(map[string]interface{}),
	}
}

// Load reads the storage file into memory
func (s *JSONStorage) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create directory if it doesn't exist
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	// Read file
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist yet, start with empty storage
		}
		return fmt.Errorf("failed to read storage file: %v", err)
	}

	// Parse JSON
	if err := json.Unmarshal(data, &s.data); err != nil {
		return fmt.Errorf("failed to parse storage file: %v", err)
	}

	return nil
}

// Save writes the current state to the storage file
func (s *JSONStorage) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.modified {
		return nil // No changes to save
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	// Marshal data
	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data: %v", err)
	}

	// Write file
	if err := os.WriteFile(s.path, data, 0644); err != nil {
		return fmt.Errorf("failed to write storage file: %v", err)
	}

	s.modified = false
	return nil
}

// Get retrieves a value from storage
func (s *JSONStorage) Get(key string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	value, exists := s.data[key]
	return value, exists
}

// Set stores a value in storage
func (s *JSONStorage) Set(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[key] = value
	s.modified = true
}

// Delete removes a value from storage
func (s *JSONStorage) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data, key)
	s.modified = true
}

// SignatureRecord represents a stored signature
type SignatureRecord struct {
	Token              string    `json:"token"`
	Wallet             string    `json:"wallet"`
	Signature          string    `json:"signature"`
	ValidUntil         uint64    `json:"validUntil"`
	GeneratedAt        time.Time `json:"generatedAt"`
	LastVerifiedTxHash string    `json:"lastVerifiedTxHash,omitempty"`
}

// SignatureStorage provides storage for signature records
type SignatureStorage struct {
	storage *JSONStorage
}

// NewSignatureStorage creates a new SignatureStorage instance
func NewSignatureStorage(path string) *SignatureStorage {
	return &SignatureStorage{
		storage: NewJSONStorage(path),
	}
}

// Load reads the signature storage file into memory
func (s *SignatureStorage) Load() error {
	return s.storage.Load()
}

// Save writes the current state to the signature storage file
func (s *SignatureStorage) Save() error {
	return s.storage.Save()
}

// GetSignature retrieves a signature record for a token-wallet pair
func (s *SignatureStorage) GetSignature(token, wallet string) (*SignatureRecord, error) {
	key := fmt.Sprintf("%s:%s", token, wallet)
	value, exists := s.storage.Get(key)
	if !exists {
		return nil, nil
	}

	// Convert map to SignatureRecord
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal stored data: %v", err)
	}

	var record SignatureRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("failed to unmarshal stored data: %v", err)
	}

	return &record, nil
}

// SaveSignature stores a signature record
func (s *SignatureStorage) SaveSignature(record *SignatureRecord) error {
	key := fmt.Sprintf("%s:%s", record.Token, record.Wallet)
	s.storage.Set(key, record)
	return s.storage.Save()
}

// DeleteSignature removes a signature record
func (s *SignatureStorage) DeleteSignature(token, wallet string) error {
	key := fmt.Sprintf("%s:%s", token, wallet)
	s.storage.Delete(key)
	return s.storage.Save()
}
