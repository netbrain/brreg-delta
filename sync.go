package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// RunSync performs either initial sync or incremental sync based on state
func RunSync(dataDir string) error {
	if IsInitialSync(dataDir) {
		log.Println("No state file found - performing initial sync")
		return runInitialSync(dataDir)
	}

	// Check if state has valid oppdateringsid (non-zero)
	state, err := LoadState(dataDir)
	if err == nil && (state.LastEnheterOppdateringsid == 0 || state.LastUnderenheterOppdateringsid == 0) {
		log.Println("State file exists but oppdateringsid is 0 - performing initial sync")
		return runInitialSync(dataDir)
	}

	log.Println("State file exists - performing incremental sync")
	return runIncrementalSync(dataDir)
}

// runInitialSync downloads complete datasets and extracts to files
func runInitialSync(dataDir string) error {
	log.Println("Starting initial sync...")

	client := NewClient()
	storage := NewStorage(dataDir)

	// Initialize state with current timestamp
	state := &SyncState{
		LastEnheterOppdateringsid:      0,
		LastUnderenheterOppdateringsid: 0,
		SyncStartedAt:                  time.Now(),
	}

	// Save state before fetching to mark sync start
	if err := SaveState(dataDir, state); err != nil {
		return fmt.Errorf("failed to save initial state: %w", err)
	}

	// Download and extract enheter
	log.Println("Downloading enheter bulk file...")
	// Note: In real implementation, we'd download from /api/enheter/lastned
	// For now, assume we have enheter.json.gz file

	// Load enheter from file (if it exists)
	enheterFile := "enheter.json.gz"
	log.Printf("Loading enheter from %s...", enheterFile)
	enheter, err := client.LoadEnheterFromFile(enheterFile)
	if err != nil {
		return fmt.Errorf("failed to load enheter: %w", err)
	}

	// Load roller from bulk file
	rollerFile := "roller.json.gz"
	log.Printf("Loading roller from %s...", rollerFile)
	roller, err := client.LoadRollerFromFile(rollerFile)
	if err != nil {
		return fmt.Errorf("failed to load roller: %w", err)
	}

	// Create a map for quick lookup of roles by organisasjonsnummer
	log.Printf("Indexing %d roller entries...", len(roller))
	rollerMap := make(map[string]json.RawMessage, len(roller))
	for _, r := range roller {
		rollerMap[r.Organisasjonsnummer] = r.RawData
	}

	log.Printf("Extracting %d enheter...", len(enheter))
	for i, enhet := range enheter {
		if err := storage.SaveEnhet(enhet.Organisasjonsnummer, enhet.RawData); err != nil {
			log.Printf("Warning: failed to save enhet %s: %v", enhet.Organisasjonsnummer, err)
			continue
		}

		// Save roles from bulk data if available
		if rollerData, ok := rollerMap[enhet.Organisasjonsnummer]; ok {
			if err := storage.SaveEnhetRoller(enhet.Organisasjonsnummer, rollerData); err != nil {
				log.Printf("Warning: failed to save roles for enhet %s: %v", enhet.Organisasjonsnummer, err)
			}
		}

		if (i+1)%10000 == 0 {
			log.Printf("Extracted %d/%d enheter", i+1, len(enheter))
		}
	}

	// Download and extract underenheter
	underenheterFile := "underenheter.json.gz"
	log.Printf("Loading underenheter from %s...", underenheterFile)
	underenheter, err := client.LoadUnderenheterFromFile(underenheterFile)
	if err != nil {
		return fmt.Errorf("failed to load underenheter: %w", err)
	}

	log.Printf("Extracting %d underenheter...", len(underenheter))
	for i, underenhet := range underenheter {
		parentOrgnum := underenhet.OverordnetEnhet
		if parentOrgnum == "" {
			log.Printf("Warning: underenhet %s has no parent", underenhet.Organisasjonsnummer)
			continue
		}

		if err := storage.SaveUnderenhet(parentOrgnum, underenhet.Organisasjonsnummer, underenhet.RawData); err != nil {
			log.Printf("Warning: failed to save underenhet %s: %v", underenhet.Organisasjonsnummer, err)
			continue
		}

		// Create symlink
		if err := storage.CreateSymlink(underenhet.Organisasjonsnummer, parentOrgnum); err != nil {
			log.Printf("Warning: failed to create symlink for %s: %v", underenhet.Organisasjonsnummer, err)
		}

		// Save roles from bulk data if available
		if rollerData, ok := rollerMap[underenhet.Organisasjonsnummer]; ok {
			if err := storage.SaveUnderenhetRoller(parentOrgnum, underenhet.Organisasjonsnummer, rollerData); err != nil {
				log.Printf("Warning: failed to save roles for underenhet %s: %v", underenhet.Organisasjonsnummer, err)
			}
		}

		if (i+1)%10000 == 0 {
			log.Printf("Extracted %d/%d underenheter", i+1, len(underenheter))
		}
	}

	// Fetch the current latest oppdateringsid to set as starting point for incremental sync
	log.Println("Fetching latest oppdateringsid values...")
	latestEnheterOppdateringsid, err := client.GetLatestEnheterOppdateringsid()
	if err != nil {
		log.Printf("Warning: failed to get latest enheter oppdateringsid: %v", err)
		log.Println("Incremental sync will start from 0 (may include some duplicates)")
	} else {
		state.LastEnheterOppdateringsid = latestEnheterOppdateringsid
		log.Printf("Latest enheter oppdateringsid: %d", latestEnheterOppdateringsid)
	}

	latestUnderenheterOppdateringsid, err := client.GetLatestUnderenheterOppdateringsid()
	if err != nil {
		log.Printf("Warning: failed to get latest underenheter oppdateringsid: %v", err)
		log.Println("Incremental sync will start from 0 (may include some duplicates)")
	} else {
		state.LastUnderenheterOppdateringsid = latestUnderenheterOppdateringsid
		log.Printf("Latest underenheter oppdateringsid: %d", latestUnderenheterOppdateringsid)
	}

	// Update state with successful completion
	state.LastSuccessfulSync = time.Now()
	if err := SaveState(dataDir, state); err != nil {
		return fmt.Errorf("failed to update state: %w", err)
	}

	log.Printf("Initial sync complete: %d enheter, %d underenheter", len(enheter), len(underenheter))
	return nil
}

// runIncrementalSync fetches updates since last sync
func runIncrementalSync(dataDir string) error {
	log.Println("Starting incremental sync...")

	client := NewClient()
	storage := NewStorage(dataDir)

	// Load current state
	state, err := LoadState(dataDir)
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	log.Printf("Last sync: %s", state.LastSuccessfulSync.Format(time.RFC3339))
	log.Printf("Last enheter oppdateringsid: %d", state.LastEnheterOppdateringsid)
	log.Printf("Last underenheter oppdateringsid: %d", state.LastUnderenheterOppdateringsid)

	// Update sync started timestamp BEFORE fetching
	state.SyncStartedAt = time.Now()
	if err := SaveState(dataDir, state); err != nil {
		return fmt.Errorf("failed to update sync start time: %w", err)
	}

	// Track progress
	enheterUpdated := 0
	underenheterUpdated := 0
	errors := 0

	// Fetch enheter updates
	log.Println("Fetching enheter updates...")
	maxEnheterOppdateringsid := state.LastEnheterOppdateringsid

	for {
		updates, err := client.FetchEnheterUpdates(maxEnheterOppdateringsid, 1000)
		if err != nil {
			return fmt.Errorf("failed to fetch enheter updates: %w", err)
		}

		if len(updates.Embedded.OppdaterteEnheter) == 0 {
			log.Println("No more enheter updates")
			break
		}

		log.Printf("Processing %d enheter updates...", len(updates.Embedded.OppdaterteEnheter))

		for _, update := range updates.Embedded.OppdaterteEnheter {
			// Update max oppdateringsid
			if update.Oppdateringsid > maxEnheterOppdateringsid {
				maxEnheterOppdateringsid = update.Oppdateringsid
			}

			// Fetch latest enhet data
			enhetData, err := client.FetchEnhet(update.Organisasjonsnummer)
			if err != nil {
				log.Printf("Warning: failed to fetch enhet %s: %v", update.Organisasjonsnummer, err)
				errors++
				continue
			}

			// Save enhet
			if err := storage.SaveEnhet(update.Organisasjonsnummer, enhetData); err != nil {
				log.Printf("Warning: failed to save enhet %s: %v", update.Organisasjonsnummer, err)
				errors++
				continue
			}

			// Fetch and save roles
			rollerData, err := client.FetchEnhetRoller(update.Organisasjonsnummer)
			if err != nil {
				log.Printf("Warning: failed to fetch roles for enhet %s: %v", update.Organisasjonsnummer, err)
				errors++
			} else if err := storage.SaveEnhetRoller(update.Organisasjonsnummer, rollerData); err != nil {
				log.Printf("Warning: failed to save roles for enhet %s: %v", update.Organisasjonsnummer, err)
				errors++
			}

			enheterUpdated++
		}

		// If we got less than requested size, we're done
		if len(updates.Embedded.OppdaterteEnheter) < 1000 {
			break
		}
	}

	// Fetch underenheter updates
	log.Println("Fetching underenheter updates...")
	maxUnderenheterOppdateringsid := state.LastUnderenheterOppdateringsid

	for {
		updates, err := client.FetchUnderenheterUpdates(maxUnderenheterOppdateringsid, 1000)
		if err != nil {
			return fmt.Errorf("failed to fetch underenheter updates: %w", err)
		}

		if len(updates.Embedded.OppdaterteUnderenheter) == 0 {
			log.Println("No more underenheter updates")
			break
		}

		log.Printf("Processing %d underenheter updates...", len(updates.Embedded.OppdaterteUnderenheter))

		for _, update := range updates.Embedded.OppdaterteUnderenheter {
			// Update max oppdateringsid
			if update.Oppdateringsid > maxUnderenheterOppdateringsid {
				maxUnderenheterOppdateringsid = update.Oppdateringsid
			}

			// Fetch latest underenhet data
			underenhetData, err := client.FetchUnderenhet(update.Organisasjonsnummer)
			if err != nil {
				log.Printf("Warning: failed to fetch underenhet %s: %v", update.Organisasjonsnummer, err)
				errors++
				continue
			}

			// Parse to get parent orgnum
			var underenhet Underenhet
			if err := underenhet.UnmarshalJSON(underenhetData); err != nil {
				log.Printf("Warning: failed to parse underenhet %s: %v", update.Organisasjonsnummer, err)
				errors++
				continue
			}

			parentOrgnum := underenhet.OverordnetEnhet
			if parentOrgnum == "" {
				log.Printf("Warning: underenhet %s has no parent", update.Organisasjonsnummer)
				errors++
				continue
			}

			// Save underenhet
			if err := storage.SaveUnderenhet(parentOrgnum, update.Organisasjonsnummer, underenhetData); err != nil {
				log.Printf("Warning: failed to save underenhet %s: %v", update.Organisasjonsnummer, err)
				errors++
				continue
			}

			// Create/update symlink
			if err := storage.CreateSymlink(update.Organisasjonsnummer, parentOrgnum); err != nil {
				log.Printf("Warning: failed to create symlink for %s: %v", update.Organisasjonsnummer, err)
				errors++
			}

			// Fetch and save roles
			rollerData, err := client.FetchUnderenhetRoller(update.Organisasjonsnummer)
			if err != nil {
				log.Printf("Warning: failed to fetch roles for underenhet %s: %v", update.Organisasjonsnummer, err)
				errors++
			} else if err := storage.SaveUnderenhetRoller(parentOrgnum, update.Organisasjonsnummer, rollerData); err != nil {
				log.Printf("Warning: failed to save roles for underenhet %s: %v", update.Organisasjonsnummer, err)
				errors++
			}

			underenheterUpdated++
		}

		// If we got less than requested size, we're done
		if len(updates.Embedded.OppdaterteUnderenheter) < 1000 {
			break
		}
	}

	// Update state with new oppdateringsid and successful completion
	state.LastEnheterOppdateringsid = maxEnheterOppdateringsid
	state.LastUnderenheterOppdateringsid = maxUnderenheterOppdateringsid
	state.LastSuccessfulSync = time.Now()

	if err := SaveState(dataDir, state); err != nil {
		return fmt.Errorf("failed to update state: %w", err)
	}

	log.Printf("Incremental sync complete:")
	log.Printf("  Enheter updated: %d", enheterUpdated)
	log.Printf("  Underenheter updated: %d", underenheterUpdated)
	log.Printf("  Errors: %d", errors)
	log.Printf("  New enheter oppdateringsid: %d", maxEnheterOppdateringsid)
	log.Printf("  New underenheter oppdateringsid: %d", maxUnderenheterOppdateringsid)

	return nil
}
