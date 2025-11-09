package main

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
)

type Processor struct {
	workers int
	storage *Storage
	client  *Client
}

func NewProcessor(workers int, dataDir string) *Processor {
	return &Processor{
		workers: workers,
		storage: NewStorage(dataDir),
		client:  NewClient(),
	}
}

type CompanyJob struct {
	Orgnum string
}

type Stats struct {
	processed       int64
	companyChanged  int64
	rolesChanged    int64
	errors          int64
}

func (s *Stats) String() string {
	return fmt.Sprintf("Processed: %d, Company changes: %d, Roles changes: %d, Errors: %d",
		s.processed, s.companyChanged, s.rolesChanged, s.errors)
}

// Run processes the batch
func (p *Processor) Run(batchNum, totalBatches int) error {
	log.Println("Loading organization numbers from data directory...")

	orgnums, err := GetCompanyOrgNums(p.storage.baseDir)
	if err != nil {
		return fmt.Errorf("failed to get organization numbers: %w", err)
	}

	log.Printf("Total companies: %d", len(orgnums))

	// Calculate batch slice
	batchSize := (len(orgnums) + totalBatches - 1) / totalBatches
	start := batchNum * batchSize
	end := start + batchSize
	if end > len(orgnums) {
		end = len(orgnums)
	}

	batchOrgnums := orgnums[start:end]
	log.Printf("Batch %d: processing %d companies (range %d-%d)",
		batchNum, len(batchOrgnums), start, end)

	// Create job queue
	jobs := make(chan CompanyJob, len(batchOrgnums))
	for _, orgnum := range batchOrgnums {
		jobs <- CompanyJob{Orgnum: orgnum}
	}
	close(jobs)

	// Worker pool
	var wg sync.WaitGroup
	stats := &Stats{}

	for i := 0; i < p.workers; i++ {
		wg.Add(1)
		go p.worker(&wg, jobs, stats)
	}

	wg.Wait()

	log.Printf("Batch complete: %s", stats.String())
	return nil
}

func (p *Processor) worker(wg *sync.WaitGroup, jobs <-chan CompanyJob, stats *Stats) {
	defer wg.Done()

	for job := range jobs {
		if err := p.processCompany(job.Orgnum, stats); err != nil {
			log.Printf("Error processing %s: %v", job.Orgnum, err)
			atomic.AddInt64(&stats.errors, 1)
		}
		atomic.AddInt64(&stats.processed, 1)

		// Log progress every 100 companies
		if atomic.LoadInt64(&stats.processed)%100 == 0 {
			log.Printf("Progress: %s", stats.String())
		}
	}
}

func (p *Processor) processCompany(orgnum string, stats *Stats) error {
	// Fetch roles (company data already exists from extraction)
	rolesData, err := p.client.FetchRoles(orgnum)
	if err != nil {
		return fmt.Errorf("fetch roles: %w", err)
	}

	// Save if changed
	changed, err := p.storage.SaveRoles(orgnum, rolesData)
	if err != nil {
		return fmt.Errorf("save roles: %w", err)
	}
	if changed {
		atomic.AddInt64(&stats.rolesChanged, 1)
	}

	return nil
}
