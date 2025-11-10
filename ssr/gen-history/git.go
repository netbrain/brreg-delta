package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// EntityChange represents a single change to an entity
type EntityChange struct {
	CommitHash string
	Date       time.Time
	Message    string
	FilePath   string
	Data       json.RawMessage
}

// GetEntityHistory retrieves all changes for a specific entity from git history
func GetEntityHistory(dataDir, orgnum string) ([]EntityChange, error) {
	// Convert orgnum to file paths
	// Example: 810034882 -> 810/034/882/enhet.json, 810/034/882/roller.json, etc.
	basePath := orgnumToPath(orgnum)

	var changes []EntityChange

	// Check for enhet.json
	enhetPath := filepath.Join("data", basePath, "enhet.json")
	enhetChanges, err := getFileHistory(dataDir, enhetPath)
	if err == nil {
		changes = append(changes, enhetChanges...)
	}

	// Check for underenhet.json
	underenhetPath := filepath.Join("data", basePath, "underenhet.json")
	underenhetChanges, err := getFileHistory(dataDir, underenhetPath)
	if err == nil {
		changes = append(changes, underenhetChanges...)
	}

	// Check for roller.json
	rollerPath := filepath.Join("data", basePath, "roller.json")
	rollerChanges, err := getFileHistory(dataDir, rollerPath)
	if err == nil {
		changes = append(changes, rollerChanges...)
	}

	if len(changes) == 0 {
		return nil, fmt.Errorf("no history found for entity %s", orgnum)
	}

	return changes, nil
}

// getFileHistory retrieves git history for a specific file
func getFileHistory(dataDir, filePath string) ([]EntityChange, error) {
	// Git log command to get all commits affecting this file
	// Format: commit_hash|unix_timestamp|commit_message
	cmd := exec.Command("git", "log",
		"--follow",
		"--pretty=format:%H|%at|%s",
		"--", filePath)
	cmd.Dir = dataDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git log failed for %s: %w", filePath, err)
	}

	if len(output) == 0 {
		return nil, fmt.Errorf("no commits found for %s", filePath)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	changes := make([]EntityChange, 0, len(lines))

	for _, line := range lines {
		parts := strings.SplitN(line, "|", 3)
		if len(parts) != 3 {
			continue
		}

		commitHash := parts[0]
		timestamp := parts[1]
		message := parts[2]

		// Parse timestamp
		var unixTime int64
		fmt.Sscanf(timestamp, "%d", &unixTime)
		date := time.Unix(unixTime, 0)

		// Get file content at this commit
		data, err := getFileAtCommit(dataDir, commitHash, filePath)
		if err != nil {
			// File might have been deleted
			continue
		}

		changes = append(changes, EntityChange{
			CommitHash: commitHash,
			Date:       date,
			Message:    message,
			FilePath:   filePath,
			Data:       data,
		})
	}

	return changes, nil
}

// getFileAtCommit retrieves file content at a specific commit
func getFileAtCommit(dataDir, commitHash, filePath string) (json.RawMessage, error) {
	cmd := exec.Command("git", "show", fmt.Sprintf("%s:%s", commitHash, filePath))
	cmd.Dir = dataDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git show failed: %w", err)
	}

	return json.RawMessage(output), nil
}

// GetCommitDiff retrieves the git diff for a specific commit and file
func GetCommitDiff(dataDir, commitHash, filePath string) (string, error) {
	// Get diff for this commit compared to its parent
	cmd := exec.Command("git", "show", "--pretty=format:", "--patch", commitHash, "--", filePath)
	cmd.Dir = dataDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git show diff failed: %w", err)
	}

	return string(output), nil
}

// GetChangedEntities returns orgnums that have changed since a specific commit
func GetChangedEntities(dataDir, sinceCommit string) ([]string, error) {
	// Get all files changed since the commit
	cmd := exec.Command("git", "diff", "--name-only", sinceCommit, "HEAD")
	cmd.Dir = dataDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git diff failed: %w", err)
	}

	if len(output) == 0 {
		return []string{}, nil
	}

	// Extract orgnums from file paths
	orgnumSet := make(map[string]bool)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		// Example: data/810/034/882/enhet.json -> 810034882
		if !strings.HasPrefix(line, "data/") {
			continue
		}

		// Remove data/ prefix
		path := strings.TrimPrefix(line, "data/")

		// Extract orgnum from path (810/034/882/enhet.json -> 810034882)
		parts := strings.Split(path, "/")
		if len(parts) < 3 {
			continue
		}

		orgnum := parts[0] + parts[1] + parts[2]

		// Validate orgnum (8 or 9 digits)
		if len(orgnum) == 8 || len(orgnum) == 9 {
			orgnumSet[orgnum] = true
		}
	}

	// Convert map to slice
	orgnums := make([]string, 0, len(orgnumSet))
	for orgnum := range orgnumSet {
		orgnums = append(orgnums, orgnum)
	}

	return orgnums, nil
}

// GetAllEntities returns all orgnums in the data directory
func GetAllEntities(dataDir string) ([]string, error) {
	// Use git ls-tree to list all files in data/
	cmd := exec.Command("git", "ls-tree", "-r", "--name-only", "HEAD", "data/")
	cmd.Dir = dataDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git ls-tree failed: %w", err)
	}

	if len(output) == 0 {
		return []string{}, nil
	}

	// Extract unique orgnums
	orgnumSet := make(map[string]bool)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		// Example: data/810/034/882/enhet.json -> 810034882
		if !strings.HasPrefix(line, "data/") {
			continue
		}

		path := strings.TrimPrefix(line, "data/")
		parts := strings.Split(path, "/")

		if len(parts) < 3 {
			continue
		}

		orgnum := parts[0] + parts[1] + parts[2]

		// Validate orgnum
		if len(orgnum) == 8 || len(orgnum) == 9 {
			orgnumSet[orgnum] = true
		}
	}

	orgnums := make([]string, 0, len(orgnumSet))
	for orgnum := range orgnumSet {
		orgnums = append(orgnums, orgnum)
	}

	return orgnums, nil
}

// orgnumToPath converts orgnum to sharded path
// Example: "810034882" -> "810/034/882"
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
