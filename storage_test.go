package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveEnhet(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "storage-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storage := NewStorage(tmpDir)
	orgnum := "123456789"

	enhetData := json.RawMessage(`{"organisasjonsnummer": "123456789", "navn": "Test AS"}`)
	rollerData := json.RawMessage(`{"roller": [{"type": "DAGL", "navn": "John Doe"}]}`)

	// Save enhet
	if err := storage.SaveEnhet(orgnum, enhetData); err != nil {
		t.Fatalf("SaveEnhet failed: %v", err)
	}

	// Save roller
	if err := storage.SaveEnhetRoller(orgnum, rollerData); err != nil {
		t.Fatalf("SaveEnhetRoller failed: %v", err)
	}

	// Verify directory structure with triple-digit sharding (123/456/789)
	enhetPath := filepath.Join(tmpDir, "123", "456", "789", "enhet.json")
	rollerPath := filepath.Join(tmpDir, "123", "456", "789", "roller.json")

	if _, err := os.Stat(enhetPath); os.IsNotExist(err) {
		t.Error("enhet.json should exist")
	}

	if _, err := os.Stat(rollerPath); os.IsNotExist(err) {
		t.Error("roller.json should exist")
	}

	// Verify prettification
	savedData, _ := os.ReadFile(enhetPath)
	if !bytes.Contains(savedData, []byte("\n")) {
		t.Error("Saved data should be prettified with newlines")
	}
}

func TestSaveUnderenhet(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "storage-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storage := NewStorage(tmpDir)
	parentOrgnum := "123456789"
	underenhetOrgnum := "987654321"

	underenhetData := json.RawMessage(`{"organisasjonsnummer": "987654321", "overordnetEnhet": "123456789", "navn": "Test Underenhet"}`)
	rollerData := json.RawMessage(`{"roller": [{"type": "LEDE", "navn": "Jane Doe"}]}`)

	// Save underenhet
	if err := storage.SaveUnderenhet(parentOrgnum, underenhetOrgnum, underenhetData); err != nil {
		t.Fatalf("SaveUnderenhet failed: %v", err)
	}

	// Save roller for underenhet
	if err := storage.SaveUnderenhetRoller(parentOrgnum, underenhetOrgnum, rollerData); err != nil {
		t.Fatalf("SaveUnderenhetRoller failed: %v", err)
	}

	// Verify underenhet is stored at its own canonical path with triple-digit sharding (987/654/321)
	underenhetPath := filepath.Join(tmpDir, "987", "654", "321", "underenhet.json")
	rollerPath := filepath.Join(tmpDir, "987", "654", "321", "roller.json")

	if _, err := os.Stat(underenhetPath); os.IsNotExist(err) {
		t.Error("underenhet.json should exist in nested path")
	}

	if _, err := os.Stat(rollerPath); os.IsNotExist(err) {
		t.Error("roller.json should exist for underenhet")
	}
}

func TestCreateSymlink(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "storage-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storage := NewStorage(tmpDir)
	parentOrgnum := "123456789"
	underenhetOrgnum := "987654321"

	// First create the underenhet directory
	underenhetData := json.RawMessage(`{"organisasjonsnummer": "987654321"}`)
	if err := storage.SaveUnderenhet(parentOrgnum, underenhetOrgnum, underenhetData); err != nil {
		t.Fatalf("SaveUnderenhet failed: %v", err)
	}

	// Create symlink
	if err := storage.CreateSymlink(underenhetOrgnum, parentOrgnum); err != nil {
		t.Fatalf("CreateSymlink failed: %v", err)
	}

	// Verify symlink exists at parent/underenhet/underenhetOrgnum
	symlinkPath := filepath.Join(tmpDir, "123", "456", "789", "underenhet", underenhetOrgnum)
	info, err := os.Lstat(symlinkPath)
	if err != nil {
		t.Fatalf("Symlink should exist: %v", err)
	}

	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("Should be a symlink")
	}

	// Verify symlink points to correct target (relative path)
	target, err := os.Readlink(symlinkPath)
	if err != nil {
		t.Fatalf("Failed to read symlink: %v", err)
	}

	// Expected: ../../../../987/654/321 (4 levels up from 123/456/789/underenhet/)
	expectedTarget := filepath.Join("../../../../", "987", "654", "321")
	if target != expectedTarget {
		t.Errorf("Symlink target mismatch: got %s, want %s", target, expectedTarget)
	}
}

func TestBackwardsCompatibility(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "storage-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storage := NewStorage(tmpDir)
	orgnum := "123456789"

	companyData := json.RawMessage(`{"organisasjonsnummer": "123456789"}`)
	rolesData := json.RawMessage(`{"roller": []}`)

	// Old method names should still work
	if err := storage.SaveCompany(orgnum, companyData); err != nil {
		t.Fatalf("SaveCompany (backwards compat) failed: %v", err)
	}

	if err := storage.SaveRoles(orgnum, rolesData); err != nil {
		t.Fatalf("SaveRoles (backwards compat) failed: %v", err)
	}

	// But should create files with new naming and triple-digit sharding
	enhetPath := filepath.Join(tmpDir, "123", "456", "789", "enhet.json")
	if _, err := os.Stat(enhetPath); os.IsNotExist(err) {
		t.Error("Backwards compat should create enhet.json with triple-digit sharding")
	}
}
