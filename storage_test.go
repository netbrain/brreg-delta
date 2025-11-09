package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveIfChanged(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "storage-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storage := NewStorage(tmpDir)

	data1 := json.RawMessage(`{"name": "Test Company", "value": 123}`)
	data2 := json.RawMessage(`{"name": "Test Company", "value": 456}`)
	data3 := json.RawMessage(`{"name": "Test Company", "value": 123}`) // Same as data1

	// First save - should return true (new file)
	changed, err := storage.SaveIfChanged("123456789", "test.json", data1)
	if err != nil {
		t.Fatalf("First save failed: %v", err)
	}
	if !changed {
		t.Error("First save should indicate change")
	}

	// Save same data - should return false (no change)
	changed, err = storage.SaveIfChanged("123456789", "test.json", data3)
	if err != nil {
		t.Fatalf("Second save failed: %v", err)
	}
	if changed {
		t.Error("Saving identical data should not indicate change")
	}

	// Save different data - should return true (changed)
	changed, err = storage.SaveIfChanged("123456789", "test.json", data2)
	if err != nil {
		t.Fatalf("Third save failed: %v", err)
	}
	if !changed {
		t.Error("Saving different data should indicate change")
	}

	// Verify final content
	savedData, err := os.ReadFile(filepath.Join(tmpDir, "123456789", "test.json"))
	if err != nil {
		t.Fatalf("Failed to read saved file: %v", err)
	}

	if string(savedData) != string(data2) {
		t.Error("Saved data does not match expected data")
	}
}

func TestSaveCompanyAndRoles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "storage-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storage := NewStorage(tmpDir)
	orgnum := "987654321"

	companyData := json.RawMessage(`{"organisasjonsnummer": "987654321", "navn": "Test AS"}`)
	rolesData := json.RawMessage(`{"roller": [{"type": "DAGL", "navn": "John Doe"}]}`)

	// Save company
	changed, err := storage.SaveCompany(orgnum, companyData)
	if err != nil {
		t.Fatalf("SaveCompany failed: %v", err)
	}
	if !changed {
		t.Error("SaveCompany should indicate change for new file")
	}

	// Save roles
	changed, err = storage.SaveRoles(orgnum, rolesData)
	if err != nil {
		t.Fatalf("SaveRoles failed: %v", err)
	}
	if !changed {
		t.Error("SaveRoles should indicate change for new file")
	}

	// Verify directory structure
	companyPath := filepath.Join(tmpDir, orgnum, "company.json")
	rolesPath := filepath.Join(tmpDir, orgnum, "roles.json")

	if _, err := os.Stat(companyPath); os.IsNotExist(err) {
		t.Error("company.json should exist")
	}

	if _, err := os.Stat(rolesPath); os.IsNotExist(err) {
		t.Error("roles.json should exist")
	}
}

func TestHashConsistency(t *testing.T) {
	// Test that the same data produces the same hash
	data := json.RawMessage(`{"test": "data"}`)

	hash1 := hashData(data)
	hash2 := hashData(data)

	if hash1 != hash2 {
		t.Error("Same data should produce same hash")
	}

	// Different data should produce different hash
	differentData := json.RawMessage(`{"test": "different"}`)
	hash3 := hashData(differentData)

	if hash1 == hash3 {
		t.Error("Different data should produce different hash")
	}
}

func TestWhitespaceChanges(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "storage-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storage := NewStorage(tmpDir)

	// Same data with different whitespace
	data1 := json.RawMessage(`{"name":"Test","value":123}`)
	data2 := json.RawMessage(`{
		"name": "Test",
		"value": 123
	}`)

	// First save
	storage.SaveIfChanged("123", "test.json", data1)

	// Second save with different whitespace - should detect as change
	// because we compare raw bytes, not semantic JSON
	changed, _ := storage.SaveIfChanged("123", "test.json", data2)

	if !changed {
		t.Error("Different whitespace should be detected as change (byte-level comparison)")
	}
}
