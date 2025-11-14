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
	MaxDurationMinutes int
}

func main() {
	// Configuration flags
	dataDir := flag.String("data", "data", "Data directory")
	rateLimit := flag.Float64("rate-limit", 20.0, "API requests per second (default: 20)")
	rateBurst := flag.Int("rate-burst", 40, "Maximum burst size for rate limiter (default: 40)")
	numWorkers := flag.Int("workers", 10, "Number of concurrent workers (default: 10)")
	maxFailures := flag.Int("max-failures", 50, "Maximum consecutive failures before exit (default: 50)")
	maxDuration := flag.Int("max-duration", 0, "Maximum sync duration in minutes, 0 = unlimited (default: 0)")

	flag.Parse()

	config := &SyncConfig{
		DataDir:            *dataDir,
		RateLimit:          *rateLimit,
		RateBurst:          *rateBurst,
		NumWorkers:         *numWorkers,
		MaxFailures:        *maxFailures,
		MaxDurationMinutes: *maxDuration,
	}

	log.Printf("Brreg Delta Sync - Configuration:")
	log.Printf("  Data directory: %s", config.DataDir)
	log.Printf("  Rate limit: %.1f req/s (burst: %d)", config.RateLimit, config.RateBurst)
	log.Printf("  Workers: %d", config.NumWorkers)
	log.Printf("  Max consecutive failures: %d", config.MaxFailures)
	if config.MaxDurationMinutes > 0 {
		log.Printf("  Max duration: %d minutes", config.MaxDurationMinutes)
	} else {
		log.Printf("  Max duration: unlimited")
	}

	if err := RunSync(config); err != nil {
		log.Fatalf("Sync failed: %v", err)
	}

	log.Println("Sync complete")
}
