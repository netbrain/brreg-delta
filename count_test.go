package main

import (
	"compress/gzip"
	"os"
	"testing"
)

func TestCountOrganizations(t *testing.T) {
	// Create temporary gzipped file with test data
	tmpFile, err := os.CreateTemp("", "count-test-*.json.gz")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	gzWriter := gzip.NewWriter(tmpFile)
	_, err = gzWriter.Write([]byte(sampleCompanyJSON))
	if err != nil {
		t.Fatalf("Failed to write gzip data: %v", err)
	}
	gzWriter.Close()
	tmpFile.Close()

	// Test counting
	count, err := CountOrganizations(tmpFile.Name())
	if err != nil {
		t.Fatalf("CountOrganizations failed: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected count 3, got %d", count)
	}
}

func TestCountOrganizationsEmptyFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "count-empty-*.json.gz")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	gzWriter := gzip.NewWriter(tmpFile)
	_, err = gzWriter.Write([]byte("[]"))
	if err != nil {
		t.Fatalf("Failed to write gzip data: %v", err)
	}
	gzWriter.Close()
	tmpFile.Close()

	count, err := CountOrganizations(tmpFile.Name())
	if err != nil {
		t.Fatalf("CountOrganizations failed: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected count 0 for empty array, got %d", count)
	}
}

func TestExtractOrganizationNumbers(t *testing.T) {
	// Create temporary input file
	tmpInput, err := os.CreateTemp("", "extract-input-*.json.gz")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpInput.Name())

	gzWriter := gzip.NewWriter(tmpInput)
	_, err = gzWriter.Write([]byte(sampleCompanyJSON))
	if err != nil {
		t.Fatalf("Failed to write gzip data: %v", err)
	}
	gzWriter.Close()
	tmpInput.Close()

	// Create temporary output file
	tmpOutput, err := os.CreateTemp("", "extract-output-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp output file: %v", err)
	}
	defer os.Remove(tmpOutput.Name())
	tmpOutput.Close()

	// Test extraction
	count, err := ExtractOrganizationNumbers(tmpInput.Name(), tmpOutput.Name())
	if err != nil {
		t.Fatalf("ExtractOrganizationNumbers failed: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected count 3, got %d", count)
	}

	// Verify output file exists and contains expected data
	if _, err := os.Stat(tmpOutput.Name()); os.IsNotExist(err) {
		t.Error("Output file should exist")
	}

	// Read and verify content
	data, err := os.ReadFile(tmpOutput.Name())
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	// Should contain the organization numbers
	content := string(data)
	if len(content) == 0 {
		t.Error("Output file should not be empty")
	}
}
