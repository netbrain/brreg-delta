package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Storage struct {
	baseDir string
}

func NewStorage(baseDir string) *Storage {
	return &Storage{baseDir: baseDir}
}

// hashData returns SHA256 hash of data
func hashData(data json.RawMessage) string {
	h := sha256.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// SaveIfChanged saves data only if hash differs from existing file
// Returns true if file was updated
func (s *Storage) SaveIfChanged(orgnum string, filename string, data json.RawMessage) (bool, error) {
	dir := filepath.Join(s.baseDir, orgnum)
	path := filepath.Join(dir, filename)

	// Calculate new hash
	newHash := hashData(data)

	// Check existing file
	if existing, err := os.ReadFile(path); err == nil {
		existingHash := hashData(existing)
		if existingHash == newHash {
			// No change
			return false, nil
		}
	}

	// Create directory if needed
	if err := os.MkdirAll(dir, 0755); err != nil {
		return false, fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return false, fmt.Errorf("failed to write file: %w", err)
	}

	return true, nil
}

// SaveCompany saves company data if changed
func (s *Storage) SaveCompany(orgnum string, data json.RawMessage) (bool, error) {
	return s.SaveIfChanged(orgnum, "company.json", data)
}

// SaveRoles saves roles data if changed
func (s *Storage) SaveRoles(orgnum string, data json.RawMessage) (bool, error) {
	return s.SaveIfChanged(orgnum, "roles.json", data)
}
