package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// SyncState tracks the last synchronization state
type SyncState struct {
	LastEnheterOppdateringsid      int64     `json:"lastEnheterOppdateringsid"`
	LastUnderenheterOppdateringsid int64     `json:"lastUnderenheterOppdateringsid"`
	SyncStartedAt                  time.Time `json:"syncStartedAt"`
	LastSuccessfulSync             time.Time `json:"lastSuccessfulSync"`
}

const stateFileName = ".sync-state.json"

// LoadState loads the sync state from data directory
func LoadState(dataDir string) (*SyncState, error) {
	path := filepath.Join(dataDir, stateFileName)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return initial state if file doesn't exist
			return &SyncState{}, nil
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state SyncState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return &state, nil
}

// SaveState saves the sync state to data directory
func SaveState(dataDir string, state *SyncState) error {
	path := filepath.Join(dataDir, stateFileName)

	// Prettify for human readability
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// IsInitialSync checks if this is the first sync (no state file exists)
func IsInitialSync(dataDir string) bool {
	path := filepath.Join(dataDir, stateFileName)
	_, err := os.Stat(path)
	return os.IsNotExist(err)
}
