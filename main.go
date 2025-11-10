package main

import (
	"flag"
	"log"
)

// SyncConfig holds configuration for sync operations
type SyncConfig struct {
	DataDir            string
	RateLimit          float64
	RateBurst          int
	NumWorkers         int
	MaxFailures        int
}

func main() {
	// Configuration flags
	dataDir := flag.String("data", "data", "Data directory")
	rateLimit := flag.Float64("rate-limit", 10.0, "API requests per second (default: 10)")
	rateBurst := flag.Int("rate-burst", 20, "Maximum burst size for rate limiter (default: 20)")
	numWorkers := flag.Int("workers", 10, "Number of concurrent workers (default: 10)")
	maxFailures := flag.Int("max-failures", 50, "Maximum consecutive failures before exit (default: 50)")

	flag.Parse()

	config := &SyncConfig{
		DataDir:     *dataDir,
		RateLimit:   *rateLimit,
		RateBurst:   *rateBurst,
		NumWorkers:  *numWorkers,
		MaxFailures: *maxFailures,
	}

	log.Printf("Brreg Delta Sync - Configuration:")
	log.Printf("  Data directory: %s", config.DataDir)
	log.Printf("  Rate limit: %.1f req/s (burst: %d)", config.RateLimit, config.RateBurst)
	log.Printf("  Workers: %d", config.NumWorkers)
	log.Printf("  Max consecutive failures: %d", config.MaxFailures)

	if err := RunSync(config); err != nil {
		log.Fatalf("Sync failed: %v", err)
	}

	log.Println("Sync complete")
}
