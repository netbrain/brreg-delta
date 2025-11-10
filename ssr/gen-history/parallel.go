package main

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// ProcessInParallel processes items in parallel with a worker pool
func ProcessInParallel(items []string, workers int, fn func(string) error) error {
	if workers <= 0 {
		workers = 1
	}

	jobs := make(chan string, len(items))
	results := make(chan error, len(items))
	var wg sync.WaitGroup

	// Track progress
	var processed atomic.Int64
	var failed atomic.Int64
	total := int64(len(items))

	// Start workers
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for item := range jobs {
				err := fn(item)

				if err != nil {
					failed.Add(1)
					results <- fmt.Errorf("worker %d: %w", workerID, err)
				} else {
					results <- nil
				}

				count := processed.Add(1)

				// Log progress every 100 items
				if count%100 == 0 {
					failCount := failed.Load()
					fmt.Printf("Progress: %d/%d (%.1f%%) - %d failed\n",
						count, total,
						float64(count)/float64(total)*100,
						failCount)
				}
			}
		}(w)
	}

	// Send jobs
	for _, item := range items {
		jobs <- item
	}
	close(jobs)

	// Wait for all workers to finish
	wg.Wait()
	close(results)

	// Collect errors
	var errors []error
	for err := range results {
		if err != nil {
			errors = append(errors, err)
		}
	}

	finalProcessed := processed.Load()
	finalFailed := failed.Load()

	fmt.Printf("\nProcessing complete:\n")
	fmt.Printf("  Total: %d\n", total)
	fmt.Printf("  Successful: %d\n", finalProcessed-finalFailed)
	fmt.Printf("  Failed: %d\n", finalFailed)

	if len(errors) > 0 {
		// Show first 10 errors
		fmt.Printf("\nFirst errors:\n")
		for i, err := range errors {
			if i >= 10 {
				fmt.Printf("  ... and %d more errors\n", len(errors)-10)
				break
			}
			fmt.Printf("  - %v\n", err)
		}
	}

	return nil
}

// ProcessBatch processes items in batches sequentially
func ProcessBatch(items []string, batchSize int, fn func([]string) error) error {
	total := len(items)
	batches := (total + batchSize - 1) / batchSize

	fmt.Printf("Processing %d items in %d batches of ~%d\n", total, batches, batchSize)

	for i := 0; i < total; i += batchSize {
		end := i + batchSize
		if end > total {
			end = total
		}

		batch := items[i:end]
		batchNum := i/batchSize + 1

		fmt.Printf("Processing batch %d/%d (%d items)\n", batchNum, batches, len(batch))

		if err := fn(batch); err != nil {
			return fmt.Errorf("batch %d failed: %w", batchNum, err)
		}
	}

	fmt.Printf("All %d batches processed successfully\n", batches)
	return nil
}
