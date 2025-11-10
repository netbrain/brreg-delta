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

	// Create temporary content directory
	contentDir, err := os.MkdirTemp("", "hugo-content-*")
	if err != nil {
		return fmt.Errorf("failed to create content dir: %w", err)
	}
	defer os.RemoveAll(contentDir)

	// For incremental mode with small number of entities, use per-entity git log
	log.Printf("Generating markdown for %d entities...", len(orgnums))
	err = ProcessInParallel(orgnums, workers, func(orgnum string) error {
		// Get full history for this entity
		changes, err := GetEntityHistory(dataDir, orgnum)
		if err != nil {
			return err
		}

		// Generate markdown
		markdown, err := GenerateTimelineMarkdown(orgnum, changes)
		if err != nil {
			return err
		}

		return WriteEntityMarkdown(orgnum, markdown, contentDir)
	})
	if err != nil {
		return fmt.Errorf("markdown generation failed: %w", err)
	}

	log.Println("\n=== Running Hugo ===")

	// Run Hugo once to build all pages
	return BuildAllWithHugo(templateDir, contentDir, output)
}

func generateAll(dataDir, templateDir, output string, workers int) error {
	log.Printf("Generating histories for ALL entities with %d workers", workers)
	log.Println("WARNING: This will take a very long time (hours to days)")

	// Get all top-level shard directories (000-999)
	shardDirs, err := getShardDirectories(dataDir)
	if err != nil {
		return fmt.Errorf("failed to get shard directories: %w", err)
	}

	log.Printf("Found %d shard directories to process", len(shardDirs))

	// Create temporary content directory
	contentDir, err := os.MkdirTemp("", "hugo-content-*")
	if err != nil {
		return fmt.Errorf("failed to create content dir: %w", err)
	}
	defer os.RemoveAll(contentDir)

	// Process each shard directory
	return processAllShardDirs(dataDir, shardDirs, contentDir, workers, templateDir, output)
}

// getShardDirectories returns all top-level shard directories in data/
func getShardDirectories(dataDir string) ([]string, error) {
	dataPath := filepath.Join(dataDir, "data")
	entries, err := os.ReadDir(dataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read data directory: %w", err)
	}

	var shardDirs []string
	for _, entry := range entries {
		if entry.IsDir() && len(entry.Name()) == 3 {
			shardDirs = append(shardDirs, entry.Name())
		}
	}

	return shardDirs, nil
}


// processAllShardDirs processes all shard directories for full generation
func processAllShardDirs(dataDir string, shardDirs []string, contentDir string, workers int, templateDir, output string) error {
	total := len(shardDirs)

	log.Printf("Processing %d shards with up to %d parallel shard workers", total, workers)

	// Process shards in parallel up to workers limit with progress tracking
	err := ProcessInParallelWithProgress(shardDirs, workers, total, func(shard string) error {
		shardPath := filepath.Join("data", shard)

		log.Printf("Shard %s: Starting", shard)

		// Build git history cache for this shard directory
		cache, err := BuildGitHistoryCacheByShard(dataDir, shardPath)
		if err != nil {
			return fmt.Errorf("failed to build git history cache: %w", err)
		}

		// Get all entities in this shard from the cache
		shardEntities := cache.GetAllEntities()

		log.Printf("Shard %s: Processing %d entities", shard, len(shardEntities))

		// Generate markdown for entities in this shard sequentially (to avoid spawning workers*workers goroutines)
		for _, orgnum := range shardEntities {
			changes, err := cache.GetEntityHistory(orgnum)
			if err != nil {
				return err
			}

			markdown, err := GenerateTimelineMarkdown(orgnum, changes)
			if err != nil {
				return err
			}

			if err := WriteEntityMarkdown(orgnum, markdown, contentDir); err != nil {
				return err
			}
		}

		log.Printf("Shard %s: Complete (%d entities)", shard, len(shardEntities))
		return nil
	})

	if err != nil {
		return fmt.Errorf("shard processing failed: %w", err)
	}

	log.Println("\n=== All shards complete, running Hugo ===")

	// Run Hugo once to build all pages
	return BuildAllWithHugo(templateDir, contentDir, output)
}

// Old batch-based function - keeping for reference but unused
func generateInBatches(dataDir string, orgnums []string, contentDir string, workers, batchSize int, templateDir, output string) error {
	total := len(orgnums)
	numBatches := (total + batchSize - 1) / batchSize

	log.Printf("Processing %d entities in %d batches of %d", total, numBatches, batchSize)

	for i := 0; i < total; i += batchSize {
		end := i + batchSize
		if end > total {
			end = total
		}

		batch := orgnums[i:end]
		batchNum := i/batchSize + 1

		log.Printf("\n=== Batch %d/%d: Processing %d entities ===", batchNum, numBatches, len(batch))

		// Build git history cache for this batch only
		cache, err := BuildGitHistoryCache(dataDir, batch)
		if err != nil {
			return fmt.Errorf("batch %d: failed to build git history cache: %w", batchNum, err)
		}

		// Generate markdown for this batch
		log.Printf("Batch %d/%d: Generating markdown files...", batchNum, numBatches)
		err = ProcessInParallel(batch, workers, func(orgnum string) error {
			changes, err := cache.GetEntityHistory(orgnum)
			if err != nil {
				return err
			}

			markdown, err := GenerateTimelineMarkdown(orgnum, changes)
			if err != nil {
				return err
			}

			return WriteEntityMarkdown(orgnum, markdown, contentDir)
		})
		if err != nil {
			return fmt.Errorf("batch %d: markdown generation failed: %w", batchNum, err)
		}

		// Clear cache to free memory
		cache = nil

		log.Printf("Batch %d/%d: Complete (%.1f%% total progress)",
			batchNum, numBatches, float64(end)/float64(total)*100)
	}

	log.Println("\n=== All batches complete, running Hugo ===")

	// Run Hugo once to build all pages
	return BuildAllWithHugo(templateDir, contentDir, output)
}
