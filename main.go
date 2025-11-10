package main

import (
	"flag"
	"log"
)

func main() {
	// Single sync command
	dataDir := flag.String("data", "data", "Data directory")
	flag.Parse()

	log.Printf("Brreg Delta Sync - Data directory: %s", *dataDir)

	if err := RunSync(*dataDir); err != nil {
		log.Fatalf("Sync failed: %v", err)
	}

	log.Println("Sync complete")
}
