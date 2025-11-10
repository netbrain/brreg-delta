package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Storage struct {
	baseDir string
}

func NewStorage(baseDir string) *Storage {
	return &Storage{baseDir: baseDir}
}

// orgnumToPath converts organization number to sharded path using triple-digit sharding
// Example: "810034882" -> "810/034/882"
// Example: "12345678" -> "123/456/78" (8-digit orgnum)
func orgnumToPath(orgnum string) string {
	var parts []string
	for i := 0; i < len(orgnum); i += 3 {
		end := i + 3
		if end > len(orgnum) {
			end = len(orgnum)
		}
		parts = append(parts, orgnum[i:end])
	}
	return strings.Join(parts, "/")
}

// save prettifies and saves JSON data to file (internal helper)
func (s *Storage) save(path string, data json.RawMessage) error {
	// Prettify JSON for human readability and git diffs
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	prettyData, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Create directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write prettified file
	if err := os.WriteFile(path, prettyData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// SaveEnhet saves enhet data
func (s *Storage) SaveEnhet(orgnum string, data json.RawMessage) error {
	path := filepath.Join(s.baseDir, orgnumToPath(orgnum), "enhet.json")
	return s.save(path, data)
}

// SaveEnhetRoller saves roles data for an enhet
func (s *Storage) SaveEnhetRoller(orgnum string, data json.RawMessage) error {
	path := filepath.Join(s.baseDir, orgnumToPath(orgnum), "roller.json")
	return s.save(path, data)
}

// SaveUnderenhet saves underenhet data at its own canonical path
func (s *Storage) SaveUnderenhet(parentOrgnum, underenhetOrgnum string, data json.RawMessage) error {
	path := filepath.Join(s.baseDir, orgnumToPath(underenhetOrgnum), "underenhet.json")
	return s.save(path, data)
}

// SaveUnderenhetRoller saves roles data for an underenhet at its own canonical path
func (s *Storage) SaveUnderenhetRoller(parentOrgnum, underenhetOrgnum string, data json.RawMessage) error {
	path := filepath.Join(s.baseDir, orgnumToPath(underenhetOrgnum), "roller.json")
	return s.save(path, data)
}

// RecordDeletedEntity appends deleted orgnum to .deleted-entities file (stored in data dir root)
func (s *Storage) RecordDeletedEntity(orgnum string) error {
	deletedFile := filepath.Join(s.baseDir, ".deleted-entities")

	// Open file for appending (create if doesn't exist)
	f, err := os.OpenFile(deletedFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open deleted-entities file: %w", err)
	}
	defer f.Close()

	// Write orgnum with newline
	if _, err := f.WriteString(orgnum + "\n"); err != nil {
		return fmt.Errorf("failed to write to deleted-entities file: %w", err)
	}

	return nil
}

// CreateSymlink creates a symlink from parent's underenhet/ directory to underenhet's canonical location
// Example: data/810/034/882/underenhet/914930553 -> ../../../../914/930/553
func (s *Storage) CreateSymlink(underenhetOrgnum, parentOrgnum string) error {
	// Link location: baseDir/parent_path/underenhet/underenhetOrgnum
	linkPath := filepath.Join(s.baseDir, orgnumToPath(parentOrgnum), "underenhet", underenhetOrgnum)

	// Target: relative path from link to baseDir/underenhet_path
	// Count levels in parent path (3-digit sharding = 3 levels for 9-digit, 3 levels for 8-digit)
	parentPathParts := strings.Split(orgnumToPath(parentOrgnum), "/")
	upLevels := strings.Repeat("../", len(parentPathParts)+1) // +1 for underenhet dir
	targetPath := filepath.Join(upLevels, orgnumToPath(underenhetOrgnum))

	// Create underenhet directory if needed
	if err := os.MkdirAll(filepath.Dir(linkPath), 0755); err != nil {
		return fmt.Errorf("failed to create underenhet directory: %w", err)
	}

	// Remove existing symlink if it exists
	os.Remove(linkPath)

	// Create symlink
	if err := os.Symlink(targetPath, linkPath); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	return nil
}

// Backwards compatibility
func (s *Storage) SaveCompany(orgnum string, data json.RawMessage) error {
	return s.SaveEnhet(orgnum, data)
}

func (s *Storage) SaveRoles(orgnum string, data json.RawMessage) error {
	return s.SaveEnhetRoller(orgnum, data)
}
