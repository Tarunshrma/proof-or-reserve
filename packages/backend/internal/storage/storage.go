package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type VerificationRecord struct {
	Token     string    `json:"token"`
	Wallet    string    `json:"wallet"`
	Balance   string    `json:"balance"`
	Timestamp time.Time `json:"timestamp"`
	Signature string    `json:"signature"`
}

type JSONStorage struct {
	sync.RWMutex
	filePath  string
	records   map[string]map[string]*VerificationRecord // token -> wallet -> record
	lastSaved time.Time
}

func NewJSONStorage(filePath string) *JSONStorage {
	return &JSONStorage{
		filePath: filePath,
		records:  make(map[string]map[string]*VerificationRecord),
	}
}

func (s *JSONStorage) Load() error {
	s.Lock()
	defer s.Unlock()

	// Create directory if it doesn't exist
	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Try to read the file
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist yet, that's ok
		}
		return err
	}

	// Parse the JSON data
	var records []*VerificationRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return err
	}

	// Populate the in-memory map
	for _, record := range records {
		if s.records[record.Token] == nil {
			s.records[record.Token] = make(map[string]*VerificationRecord)
		}
		s.records[record.Token][record.Wallet] = record
	}

	s.lastSaved = time.Now()
	return nil
}

func (s *JSONStorage) Save() error {
	s.Lock()
	defer s.Unlock()

	// Convert map to slice for storage
	var records []*VerificationRecord
	for _, wallets := range s.records {
		for _, record := range wallets {
			records = append(records, record)
		}
	}

	// Create JSON data
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}

	// Write to file
	if err := os.WriteFile(s.filePath, data, 0644); err != nil {
		return err
	}

	s.lastSaved = time.Now()
	return nil
}

func (s *JSONStorage) StoreVerification(record *VerificationRecord) error {
	s.Lock()
	defer s.Unlock()

	if s.records[record.Token] == nil {
		s.records[record.Token] = make(map[string]*VerificationRecord)
	}
	s.records[record.Token][record.Wallet] = record

	return s.Save()
}

func (s *JSONStorage) GetLastVerification(token, wallet string) (*VerificationRecord, error) {
	s.RLock()
	defer s.RUnlock()

	if wallets, exists := s.records[token]; exists {
		if record, exists := wallets[wallet]; exists {
			return record, nil
		}
	}
	return nil, nil
}

func (s *JSONStorage) GetAllVerifications() []*VerificationRecord {
	s.RLock()
	defer s.RUnlock()

	var records []*VerificationRecord
	for _, wallets := range s.records {
		for _, record := range wallets {
			records = append(records, record)
		}
	}
	return records
}
