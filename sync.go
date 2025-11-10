package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// updateJob represents a single entity update to process
type updateJob struct {
	orgnum       string
	isEnhet      bool
	oppdateringsid int64
}

// updateResult represents the result of processing an update
type updateResult struct {
	orgnum  string
	success bool
	err     error
}

// RunSync performs either initial sync or incremental sync based on state
func RunSync(config *SyncConfig) error {
	dataDir := config.DataDir
	if IsInitialSync(dataDir) {
		log.Println("No state file found - performing initial sync")
		return runInitialSync(config)
	}

	// Check if state has valid oppdateringsid (non-zero)
	state, err := LoadState(dataDir)
	if err == nil && (state.LastEnheterOppdateringsid == 0 || state.LastUnderenheterOppdateringsid == 0) {
		log.Println("State file exists but oppdateringsid is 0 - performing initial sync")
		return runInitialSync(config)
	}

	log.Println("State file exists - performing incremental sync")
	return runIncrementalSync(config)
}

// runInitialSync downloads complete datasets and extracts to files
func runInitialSync(config *SyncConfig) error {
	log.Println("Starting initial sync...")

	dataDir := config.DataDir
	client := NewClientWithOptions(config.RateLimit, config.RateBurst)
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

// processEnhetUpdate processes a single enhet update
func processEnhetUpdate(client *Client, storage *Storage, orgnum string) error {
	// Fetch latest enhet data
	enhetData, err := client.FetchEnhet(orgnum)
	if err != nil {
		// Check if entity was deleted (410 Gone)
		var deletedErr *EntityDeletedError
		if errors.As(err, &deletedErr) {
			log.Printf("Entity deleted: %s - recording for cleanup", orgnum)
			if err := storage.RecordDeletedEntity(orgnum); err != nil {
				return fmt.Errorf("failed to record deleted entity: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to fetch enhet: %w", err)
	}

	// Save enhet
	if err := storage.SaveEnhet(orgnum, enhetData); err != nil {
		return fmt.Errorf("failed to save enhet: %w", err)
	}

	// Fetch and save roles
	rollerData, err := client.FetchEnhetRoller(orgnum)
	if err != nil {
		return fmt.Errorf("failed to fetch roles: %w", err)
	}
	if err := storage.SaveEnhetRoller(orgnum, rollerData); err != nil {
		return fmt.Errorf("failed to save roles: %w", err)
	}

	return nil
}

// processUnderenhetUpdate processes a single underenhet update
func processUnderenhetUpdate(client *Client, storage *Storage, orgnum string) error {
	// Fetch latest underenhet data
	underenhetData, err := client.FetchUnderenhet(orgnum)
	if err != nil {
		// Check if entity was deleted (410 Gone)
		var deletedErr *EntityDeletedError
		if errors.As(err, &deletedErr) {
			log.Printf("Entity deleted: %s - recording for cleanup", orgnum)
			if err := storage.RecordDeletedEntity(orgnum); err != nil {
				return fmt.Errorf("failed to record deleted entity: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to fetch underenhet: %w", err)
	}

	// Parse to get parent orgnum
	var underenhet Underenhet
	if err := underenhet.UnmarshalJSON(underenhetData); err != nil {
		return fmt.Errorf("failed to parse underenhet: %w", err)
	}

	parentOrgnum := underenhet.OverordnetEnhet
	if parentOrgnum == "" {
		return fmt.Errorf("underenhet has no parent")
	}

	// Save underenhet
	if err := storage.SaveUnderenhet(parentOrgnum, orgnum, underenhetData); err != nil {
		return fmt.Errorf("failed to save underenhet: %w", err)
	}

	// Create/update symlink
	if err := storage.CreateSymlink(orgnum, parentOrgnum); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	// Fetch and save roles
	rollerData, err := client.FetchUnderenhetRoller(orgnum)
	if err != nil {
		return fmt.Errorf("failed to fetch roles: %w", err)
	}
	if err := storage.SaveUnderenhetRoller(parentOrgnum, orgnum, rollerData); err != nil {
		return fmt.Errorf("failed to save roles: %w", err)
	}

	return nil
}

// worker processes update jobs from the jobs channel
func worker(id int, jobs <-chan updateJob, results chan<- updateResult, client *Client, storage *Storage) {
	for job := range jobs {
		var err error
		if job.isEnhet {
			err = processEnhetUpdate(client, storage, job.orgnum)
		} else {
			err = processUnderenhetUpdate(client, storage, job.orgnum)
		}

		results <- updateResult{
			orgnum:  job.orgnum,
			success: err == nil,
			err:     err,
		}
	}
}

// runIncrementalSync fetches updates since last sync
func runIncrementalSync(config *SyncConfig) error {
	log.Println("Starting incremental sync...")

	dataDir := config.DataDir
	client := NewClientWithOptions(config.RateLimit, config.RateBurst)
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
	errorCount := 0
	consecutiveFailures := 0

	// Setup worker pool
	jobs := make(chan updateJob, 100)
	results := make(chan updateResult, 100)

	var wg sync.WaitGroup
	for w := 1; w <= config.NumWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			worker(workerID, jobs, results, client, storage)
		}(w)
	}

	// Results collector goroutine
	resultsDone := make(chan bool)
	go func() {
		for result := range results {
			if result.success {
				consecutiveFailures = 0
			} else {
				consecutiveFailures++
				errorCount++
				log.Printf("Warning: failed to process %s: %v", result.orgnum, result.err)

				if consecutiveFailures >= config.MaxFailures {
					log.Printf("FATAL: %d consecutive failures - exiting application", consecutiveFailures)
					os.Exit(1)
				}
			}
		}
		resultsDone <- true
	}()

	// Fetch enheter updates
	log.Println("Fetching enheter updates...")
	maxEnheterOppdateringsid := state.LastEnheterOppdateringsid

	for {
		updates, err := client.FetchEnheterUpdates(maxEnheterOppdateringsid, 1000)
		if err != nil {
			close(jobs)
			wg.Wait()
			close(results)
			<-resultsDone
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

			jobs <- updateJob{
				orgnum:       update.Organisasjonsnummer,
				isEnhet:      true,
				oppdateringsid: update.Oppdateringsid,
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
			close(jobs)
			wg.Wait()
			close(results)
			<-resultsDone
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

			jobs <- updateJob{
				orgnum:       update.Organisasjonsnummer,
				isEnhet:      false,
				oppdateringsid: update.Oppdateringsid,
			}
			underenheterUpdated++
		}

		// If we got less than requested size, we're done
		if len(updates.Embedded.OppdaterteUnderenheter) < 1000 {
			break
		}
	}

	// Close jobs channel and wait for all workers to finish
	close(jobs)
	wg.Wait()
	close(results)
	<-resultsDone

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
	log.Printf("  Errors: %d", errorCount)
	log.Printf("  New enheter oppdateringsid: %d", maxEnheterOppdateringsid)
	log.Printf("  New underenheter oppdateringsid: %d", maxUnderenheterOppdateringsid)

	return nil
}
