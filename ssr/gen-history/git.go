package main

import (
	"encoding/json"
	"fmt"
	"log"
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
	// Get diff for this commit compared to its parent (no context lines - only show changes)
	cmd := exec.Command("git", "show", "--pretty=format:", "--patch", "--unified=0", commitHash, "--", filePath)
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

// pathToOrgnum extracts orgnum from sharded path
// Example: "data/810/034/882/enhet.json" -> "810034882"
func pathToOrgnum(path string) string {
	// Remove "data/" prefix
	path = strings.TrimPrefix(path, "data/")

	// Split by / and take first 3 parts
	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		return ""
	}

	return parts[0] + parts[1] + parts[2]
}

// GitHistoryCache caches all git history for efficient lookup
type GitHistoryCache struct {
	// Map of orgnum -> list of changes
	entityChanges map[string][]EntityChange
}

// BuildGitHistoryCache builds a cache of git history for specific entities
func BuildGitHistoryCache(dataDir string, orgnums []string) (*GitHistoryCache, error) {
	log.Printf("Building git history cache for %d entities...", len(orgnums))

	// Build patterns for entities to limit git log output
	// Example: data/810/034/882/*.json
	patterns := make([]string, 0, len(orgnums))
	for _, orgnum := range orgnums {
		pattern := filepath.Join("data", orgnumToPath(orgnum), "*.json")
		patterns = append(patterns, pattern)
	}

	// If too many patterns, just use data/ and filter
	var cmd *exec.Cmd
	if len(patterns) > 1000 {
		log.Printf("Too many entities (%d), using full git log with filtering...", len(orgnums))
		cmd = exec.Command("git", "log",
			"--name-only",
			"--pretty=format:%H|%at|%s",
			"--", "data/")
	} else {
		// Use specific patterns for smaller batches
		args := []string{"log", "--name-only", "--pretty=format:%H|%at|%s", "--"}
		args = append(args, patterns...)
		cmd = exec.Command("git", args...)
	}
	cmd.Dir = dataDir

	log.Println("Running git log...")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git log failed: %w", err)
	}

	log.Println("Parsing git log output...")

	// Create orgnum set for fast filtering
	orgnumSet := make(map[string]bool)
	for _, orgnum := range orgnums {
		orgnumSet[orgnum] = true
	}

	// Parse output and group by entity
	cache := &GitHistoryCache{
		entityChanges: make(map[string][]EntityChange),
	}

	lines := strings.Split(string(output), "\n")
	var currentCommit, currentMessage string
	var currentTimestamp int64
	var filesInCommit []string

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		if line == "" {
			// Empty line marks end of file list for this commit
			if currentCommit != "" && len(filesInCommit) > 0 {
				// Process all files in this commit (only for our entities)
				cache.addCommitFilesFiltered(dataDir, currentCommit, currentTimestamp, currentMessage, filesInCommit, orgnumSet)
				filesInCommit = nil
			}
			continue
		}

		// Check if this is a commit line (contains |)
		if strings.Contains(line, "|") {
			parts := strings.SplitN(line, "|", 3)
			if len(parts) == 3 {
				currentCommit = parts[0]
				fmt.Sscanf(parts[1], "%d", &currentTimestamp)
				currentMessage = parts[2]
			}
			continue
		}

		// This is a file path
		if strings.HasPrefix(line, "data/") && strings.HasSuffix(line, ".json") {
			filesInCommit = append(filesInCommit, line)
		}
	}

	// Process last commit
	if currentCommit != "" && len(filesInCommit) > 0 {
		cache.addCommitFilesFiltered(dataDir, currentCommit, currentTimestamp, currentMessage, filesInCommit, orgnumSet)
	}

	log.Printf("Git history cache built: %d entities with history", len(cache.entityChanges))
	return cache, nil
}

// addCommitFilesFiltered adds files from a commit to the cache (only for specified entities)
func (c *GitHistoryCache) addCommitFilesFiltered(dataDir, commitHash string, timestamp int64, message string, files []string, orgnumSet map[string]bool) {
	date := time.Unix(timestamp, 0)

	for _, filePath := range files {
		orgnum := pathToOrgnum(filePath)
		if orgnum == "" {
			continue
		}

		// Skip if not in our entity set
		if !orgnumSet[orgnum] {
			continue
		}

		// Get file content at this commit
		data, err := getFileAtCommit(dataDir, commitHash, filePath)
		if err != nil {
			continue
		}

		change := EntityChange{
			CommitHash: commitHash,
			Date:       date,
			Message:    message,
			FilePath:   filePath,
			Data:       data,
		}

		c.entityChanges[orgnum] = append(c.entityChanges[orgnum], change)
	}
}

// GetEntityHistory retrieves changes for an entity from the cache
func (c *GitHistoryCache) GetEntityHistory(orgnum string) ([]EntityChange, error) {
	changes, ok := c.entityChanges[orgnum]
	if !ok || len(changes) == 0 {
		return nil, fmt.Errorf("no history found for entity %s", orgnum)
	}

	return changes, nil
}

// GetAllEntities returns all entity orgnums in the cache
func (c *GitHistoryCache) GetAllEntities() []string {
	orgnums := make([]string, 0, len(c.entityChanges))
	for orgnum := range c.entityChanges {
		orgnums = append(orgnums, orgnum)
	}
	return orgnums
}

// BuildGitHistoryCacheByShard builds cache for a single shard directory
func BuildGitHistoryCacheByShard(dataDir, shardPath string) (*GitHistoryCache, error) {
	log.Printf("Building git history cache for shard: %s", shardPath)

	// Run git log for this specific shard directory
	cmd := exec.Command("git", "log",
		"--name-only",
		"--pretty=format:%H|%at|%s",
		"--", shardPath+"/")
	cmd.Dir = dataDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git log failed for shard %s: %w", shardPath, err)
	}

	// Parse output
	cache := &GitHistoryCache{
		entityChanges: make(map[string][]EntityChange),
	}

	lines := strings.Split(string(output), "\n")
	var currentCommit, currentMessage string
	var currentTimestamp int64
	var filesInCommit []string

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		if line == "" {
			// Empty line marks end of file list for this commit
			if currentCommit != "" && len(filesInCommit) > 0 {
				// Process all files in this commit
				cache.addCommitFilesAll(dataDir, currentCommit, currentTimestamp, currentMessage, filesInCommit)
				filesInCommit = nil
			}
			continue
		}

		// Check if this is a commit line (contains |)
		if strings.Contains(line, "|") {
			parts := strings.SplitN(line, "|", 3)
			if len(parts) == 3 {
				currentCommit = parts[0]
				fmt.Sscanf(parts[1], "%d", &currentTimestamp)
				currentMessage = parts[2]
			}
			continue
		}

		// This is a file path
		if strings.HasPrefix(line, "data/") && strings.HasSuffix(line, ".json") {
			filesInCommit = append(filesInCommit, line)
		}
	}

	// Process last commit
	if currentCommit != "" && len(filesInCommit) > 0 {
		cache.addCommitFilesAll(dataDir, currentCommit, currentTimestamp, currentMessage, filesInCommit)
	}

	log.Printf("Git history cache built for shard %s: %d entities", shardPath, len(cache.entityChanges))
	return cache, nil
}

// addCommitFilesAll adds all files from a commit to the cache (no filtering)
func (c *GitHistoryCache) addCommitFilesAll(dataDir, commitHash string, timestamp int64, message string, files []string) {
	date := time.Unix(timestamp, 0)

	for _, filePath := range files {
		orgnum := pathToOrgnum(filePath)
		if orgnum == "" {
			continue
		}

		// Get file content at this commit
		data, err := getFileAtCommit(dataDir, commitHash, filePath)
		if err != nil {
			continue
		}

		change := EntityChange{
			CommitHash: commitHash,
			Date:       date,
			Message:    message,
			FilePath:   filePath,
			Data:       data,
		}

		c.entityChanges[orgnum] = append(c.entityChanges[orgnum], change)
	}
}
