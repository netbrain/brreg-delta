package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	var (
		command      string
		orgnum       string
		incremental  bool
		all          bool
		output       string
		workers      int
		templateDir  string
	)

	// Define subcommands
	generateCmd := flag.NewFlagSet("generate", flag.ExitOnError)
	generateCmd.StringVar(&orgnum, "orgnum", "", "Specific organization number to generate")
	generateCmd.BoolVar(&incremental, "incremental", false, "Generate only changed entities since last run")
	generateCmd.BoolVar(&all, "all", false, "Generate all entities (full rebuild)")
	generateCmd.StringVar(&output, "output", "./output", "Output directory for generated HTML files")
	generateCmd.IntVar(&workers, "workers", 10, "Number of parallel workers")
	generateCmd.StringVar(&templateDir, "template", "../entity-template", "Hugo template directory")

	allCmd := flag.NewFlagSet("all", flag.ExitOnError)
	allCmd.BoolVar(&incremental, "incremental", false, "Generate only changed entities")
	allCmd.StringVar(&output, "output", "./output", "Output directory")
	allCmd.IntVar(&workers, "workers", 10, "Number of parallel workers")
	allCmd.StringVar(&templateDir, "template", "../entity-template", "Hugo template directory")

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command = os.Args[1]

	switch command {
	case "generate":
		generateCmd.Parse(os.Args[2:])
		if err := runGenerate(orgnum, incremental, all, output, workers, templateDir); err != nil {
			log.Fatalf("Generate failed: %v", err)
		}

	case "all":
		allCmd.Parse(os.Args[2:])
		if err := runGenerate("", incremental, !incremental, output, workers, templateDir); err != nil {
			log.Fatalf("Generate failed: %v", err)
		}

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  gen-history generate [options]  - Generate entity history pages")
	fmt.Println("  gen-history all [options]       - Generate all entity pages")
	fmt.Println()
	fmt.Println("Generate options:")
	fmt.Println("  --orgnum <num>       Generate specific organization number")
	fmt.Println("  --incremental        Generate only changed entities")
	fmt.Println("  --all                Generate all entities (full rebuild)")
	fmt.Println("  --output <dir>       Output directory (default: ./output)")
	fmt.Println("  --workers <n>        Number of parallel workers (default: 10)")
	fmt.Println("  --template <dir>     Hugo template directory (default: ../entity-template)")
}

func runGenerate(orgnum string, incremental, all bool, output string, workers int, templateDir string) error {
	log.Println("Starting entity history generation...")

	// Ensure data directory exists (we're running from brreg-data directory)
	dataDir := "."

	// Single entity
	if orgnum != "" {
		return generateSingleEntity(dataDir, orgnum, templateDir, output)
	}

	// Incremental (changed entities)
	if incremental {
		return generateIncremental(dataDir, templateDir, output, workers)
	}

	// All entities
	if all {
		return generateAll(dataDir, templateDir, output, workers)
	}

	return fmt.Errorf("must specify --orgnum, --incremental, or --all")
}

func generateSingleEntity(dataDir, orgnum, templateDir, output string) error {
	log.Printf("Generating history for entity: %s", orgnum)

	// Get history
	changes, err := GetEntityHistory(dataDir, orgnum)
	if err != nil {
		return fmt.Errorf("failed to get history: %w", err)
	}

	log.Printf("Found %d changes for entity %s", len(changes), orgnum)

	// Generate markdown
	markdown, err := GenerateTimelineMarkdown(orgnum, changes)
	if err != nil {
		return fmt.Errorf("failed to generate markdown: %w", err)
	}

	// Generate HTML
	if err := GenerateEntityHTML(orgnum, markdown, templateDir, output); err != nil {
		return fmt.Errorf("failed to generate HTML: %w", err)
	}

	log.Printf("Successfully generated history for %s", orgnum)
	return nil
}

func generateIncremental(dataDir, templateDir, output string, workers int) error {
	log.Println("Generating histories for changed entities (incremental mode)")

	// Get last commit from state file
	lastCommit := "HEAD~1" // Default fallback
	stateFile := filepath.Join(output, ".last-history-commit")
	if data, err := os.ReadFile(stateFile); err == nil {
		lastCommit = strings.TrimSpace(string(data))
		log.Printf("Found previous state: %s", lastCommit)
	} else {
		log.Printf("No state file found, using default: %s", lastCommit)
	}

	// Get changed entities
	orgnums, err := GetChangedEntities(dataDir, lastCommit)
	if err != nil {
		return fmt.Errorf("failed to get changed entities: %w", err)
	}

	log.Printf("Found %d changed entities since %s", len(orgnums), lastCommit)

	if len(orgnums) == 0 {
		log.Println("No entities changed")
		return nil
	}

	// Process in parallel
	return ProcessInParallel(orgnums, workers, func(orgnum string) error {
		changes, err := GetEntityHistory(dataDir, orgnum)
		if err != nil {
			return err
		}

		markdown, err := GenerateTimelineMarkdown(orgnum, changes)
		if err != nil {
			return err
		}

		return GenerateEntityHTML(orgnum, markdown, templateDir, output)
	})
}

func generateAll(dataDir, templateDir, output string, workers int) error {
	log.Printf("Generating histories for ALL entities with %d workers", workers)
	log.Println("WARNING: This will take a very long time (hours to days)")

	// Get all entities
	orgnums, err := GetAllEntities(dataDir)
	if err != nil {
		return fmt.Errorf("failed to get all entities: %w", err)
	}

	log.Printf("Found %d total entities", len(orgnums))

	// Process in parallel
	return ProcessInParallel(orgnums, workers, func(orgnum string) error {
		changes, err := GetEntityHistory(dataDir, orgnum)
		if err != nil {
			return err
		}

		markdown, err := GenerateTimelineMarkdown(orgnum, changes)
		if err != nil {
			return err
		}

		return GenerateEntityHTML(orgnum, markdown, templateDir, output)
	})
}
