package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type SignatureRecord struct {
	Token       string    `json:"token"`
	Wallet      string    `json:"wallet"`
	Signature   string    `json:"signature"`
	ValidUntil  uint64    `json:"validUntil"`
	GeneratedAt time.Time `json:"generatedAt"`
}

type SignatureStorage struct {
	sync.RWMutex
	filePath   string
	signatures map[string]map[string]*SignatureRecord // token -> wallet -> record
}

func NewSignatureStorage(filePath string) *SignatureStorage {
	return &SignatureStorage{
		filePath:   filePath,
		signatures: make(map[string]map[string]*SignatureRecord),
	}
}

func (s *SignatureStorage) Store(record *SignatureRecord) error {
	s.Lock()
	defer s.Unlock()

	if s.signatures[record.Token] == nil {
		s.signatures[record.Token] = make(map[string]*SignatureRecord)
	}
	s.signatures[record.Token][record.Wallet] = record

	return s.save()
}

func (s *SignatureStorage) Get(token, wallet string) (*SignatureRecord, error) {
	s.RLock()
	defer s.RUnlock()

	if wallets, exists := s.signatures[token]; exists {
		if record, exists := wallets[wallet]; exists {
			return record, nil
		}
	}
	return nil, fmt.Errorf("signature not found")
}

func (s *SignatureStorage) save() error {
	// Convert map to slice for storage
	var records []*SignatureRecord
	for _, wallets := range s.signatures {
		for _, record := range wallets {
			records = append(records, record)
		}
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Create JSON data
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(s.filePath, data, 0644)
}

func (s *SignatureStorage) Load() error {
	s.Lock()
	defer s.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist yet, that's ok
		}
		return err
	}

	var records []*SignatureRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return err
	}

	// Populate the in-memory map
	for _, record := range records {
		if s.signatures[record.Token] == nil {
			s.signatures[record.Token] = make(map[string]*SignatureRecord)
		}
		s.signatures[record.Token][record.Wallet] = record
	}

	return nil
}
