package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {
	// Subcommands
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "extract":
			extractCmd := flag.NewFlagSet("extract", flag.ExitOnError)
			listFile := extractCmd.String("list", "", "Path to company list file")
			dataDir := extractCmd.String("data", "data", "Data directory")
			extractCmd.Parse(os.Args[2:])

			if *listFile == "" {
				log.Fatal("--list is required")
			}

			count, err := ExtractCompanies(*listFile, *dataDir)
			if err != nil {
				log.Fatalf("Error extracting companies: %v", err)
			}
			fmt.Printf("Extracted %d companies\n", count)
			return

		case "count":
			countCmd := flag.NewFlagSet("count", flag.ExitOnError)
			dataDir := countCmd.String("data", "data", "Data directory")
			countCmd.Parse(os.Args[2:])

			count, err := CountExtractedCompanies(*dataDir)
			if err != nil {
				log.Fatalf("Error counting companies: %v", err)
			}
			fmt.Printf("Total companies: %d\n", count)
			return
		}
	}

	// Main command flags
	workers := flag.Int("workers", 1, "Number of parallel workers")
	batchNum := flag.Int("batch", 0, "Batch number (0-indexed)")
	totalBatches := flag.Int("total", 256, "Total number of batches")
	dataDir := flag.String("data", "data", "Data directory")
	flag.Parse()

	if *batchNum < 0 || *batchNum >= *totalBatches {
		log.Fatalf("batch must be between 0 and %d", *totalBatches-1)
	}

	log.Printf("Starting batch %d/%d with %d workers", *batchNum+1, *totalBatches, *workers)

	processor := NewProcessor(*workers, *dataDir)

	if err := processor.Run(*batchNum, *totalBatches); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	log.Println("Batch complete")
}
