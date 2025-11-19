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
		shard        string
		incremental  bool
		all          bool
		output       string
		workers      int
		templatesDir string
	)

	// Define subcommands
	generateCmd := flag.NewFlagSet("generate", flag.ExitOnError)
	generateCmd.StringVar(&orgnum, "orgnum", "", "Specific organization number to generate")
	generateCmd.BoolVar(&incremental, "incremental", false, "Generate only changed entities since last run")
	generateCmd.BoolVar(&all, "all", false, "Generate all entities (full rebuild)")
	generateCmd.StringVar(&output, "output", "./output", "Output directory for generated HTML files")
	generateCmd.IntVar(&workers, "workers", 10, "Number of parallel workers")
	generateCmd.StringVar(&templatesDir, "templates", "..", "Templates directory (expects entity-template and navigation-site subdirs)")

	allCmd := flag.NewFlagSet("all", flag.ExitOnError)
	allCmd.BoolVar(&incremental, "incremental", false, "Generate only changed entities")
	allCmd.StringVar(&output, "output", "./output", "Output directory")
	allCmd.IntVar(&workers, "workers", 10, "Number of parallel workers")
	allCmd.StringVar(&templatesDir, "templates", "..", "Templates directory (expects entity-template and navigation-site subdirs)")

	shardCmd := flag.NewFlagSet("shard", flag.ExitOnError)
	shardCmd.StringVar(&shard, "shard", "", "Shard directory to process (e.g., 810)")
	shardCmd.StringVar(&output, "output", "./output", "Output directory")
	shardCmd.IntVar(&workers, "workers", 10, "Number of parallel workers")
	shardCmd.StringVar(&templatesDir, "templates", "..", "Templates directory (expects entity-template and navigation-site subdirs)")

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command = os.Args[1]

	switch command {
	case "generate":
		generateCmd.Parse(os.Args[2:])
		if err := runGenerate(orgnum, incremental, all, output, workers, templatesDir); err != nil {
			log.Fatalf("Generate failed: %v", err)
		}

	case "all":
		allCmd.Parse(os.Args[2:])
		if err := runGenerate("", incremental, !incremental, output, workers, templatesDir); err != nil {
			log.Fatalf("Generate failed: %v", err)
		}

	case "shard":
		shardCmd.Parse(os.Args[2:])
		if shard == "" {
			log.Fatal("--shard is required")
		}
		if err := runShard(shard, output, workers, templatesDir); err != nil {
			log.Fatalf("Shard generation failed: %v", err)
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
	fmt.Println("  gen-history shard [options]     - Generate single shard")
	fmt.Println()
	fmt.Println("Generate options:")
	fmt.Println("  --orgnum <num>       Generate specific organization number")
	fmt.Println("  --incremental        Generate only changed entities")
	fmt.Println("  --all                Generate all entities (full rebuild)")
	fmt.Println("  --output <dir>       Output directory (default: ./output)")
	fmt.Println("  --workers <n>        Number of parallel workers (default: 10)")
	fmt.Println("  --templates <dir>    Templates directory with entity-template and navigation-site subdirs (default: ..)")
	fmt.Println()
	fmt.Println("Shard options:")
	fmt.Println("  --shard <num>        Shard directory to process (e.g., 810)")
	fmt.Println("  --output <dir>       Output directory (default: ./output)")
	fmt.Println("  --workers <n>        Number of parallel workers (default: 10)")
	fmt.Println("  --templates <dir>    Templates directory (default: ..)")
}

func runGenerate(orgnum string, incremental, all bool, output string, workers int, templatesDir string) error {
	log.Println("Starting entity history generation...")

	// Ensure data directory exists (we're running from brreg-data directory)
	dataDir := "."

	// Derive template paths
	entityTemplateDir := filepath.Join(templatesDir, "entity-template")
	navSiteDir := filepath.Join(templatesDir, "navigation-site")

	var err error

	// Single entity
	if orgnum != "" {
		err = generateSingleEntity(dataDir, orgnum, entityTemplateDir, output)
	} else if incremental {
		// Incremental (changed entities)
		err = generateIncremental(dataDir, entityTemplateDir, output, workers)
	} else if all {
		// All entities
		err = generateAll(dataDir, entityTemplateDir, output, workers)
	} else {
		return fmt.Errorf("must specify --orgnum, --incremental, or --all")
	}

	if err != nil {
		return err
	}

	// Build navigation site and copy index.html to output last (to overwrite entity-generated index)
	log.Println("\n=== Building navigation site ===")
	if err := BuildNavigationSite(navSiteDir, output); err != nil {
		return fmt.Errorf("failed to build navigation site: %w", err)
	}

	return nil
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

	// Get last commit from state file (stored in data directory, not output)
	lastCommit := "HEAD~1" // Default fallback
	stateFile := filepath.Join(dataDir, ".last-history-commit")
	if data, err := os.ReadFile(stateFile); err == nil {
		lastCommit = strings.TrimSpace(string(data))
		log.Printf("Found previous state: %s", lastCommit)
	} else {
		log.Printf("No state file found, using default: %s", lastCommit)
	}

	// Get changed entities with their file paths
	result, err := GetChangedEntitiesWithPaths(dataDir, lastCommit)
	if err != nil {
		return fmt.Errorf("failed to get changed entities: %w", err)
	}

	orgnums := result.Orgnums
	filePaths := result.FilePaths

	log.Printf("Found %d changed entities (%d files) since %s", len(orgnums), len(filePaths), lastCommit)

	if len(orgnums) == 0 {
		log.Println("No entities changed")
		return nil
	}

	// Create temporary content directory
	contentDir, err := os.MkdirTemp("", "hugo-content-*")
	if err != nil {
		return fmt.Errorf("failed to create content dir: %w", err)
	}
	// Note: contentDir will be moved (not copied) to output, so no cleanup needed

	// Build git history cache using specific file paths (much faster than scanning entire repo)
	log.Printf("Building git history cache for %d entities using %d specific file paths...", len(orgnums), len(filePaths))
	cache, err := BuildGitHistoryCacheWithPaths(dataDir, orgnums, filePaths)
	if err != nil {
		return fmt.Errorf("failed to build git history cache: %w", err)
	}

	log.Printf("Generating markdown for %d entities...", len(orgnums))
	err = ProcessInParallel(orgnums, workers, func(orgnum string) error {
		// Get history from cache
		changes, err := cache.GetEntityHistory(orgnum)
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

	log.Println("\n=== Preparing Hugo site ===")

	// Prepare Hugo site structure (template + content)
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

	// Process each shard directory (each shard manages its own temp dirs)
	return processAllShardDirs(dataDir, shardDirs, "", workers, templateDir, output)
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

	log.Printf("Processing %d shards sequentially (markdown -> Hugo -> output per shard)", total)

	// Process shards sequentially (one at a time) to keep memory usage constant
	for i, shard := range shardDirs {
		shardPath := filepath.Join("data", shard)

		log.Printf("Shard %s (%d/%d): Starting", shard, i+1, total)

		// Build git history cache for this shard directory
		cache, err := BuildGitHistoryCacheByShard(dataDir, shardPath)
		if err != nil {
			return fmt.Errorf("shard %s: failed to build git history cache: %w", shard, err)
		}

		// Get all entities in this shard from the cache
		shardEntities := cache.GetAllEntities()

		log.Printf("Shard %s: Generating markdown for %d entities", shard, len(shardEntities))

		// Create temporary content directory for this shard
		shardContentDir, err := os.MkdirTemp("", fmt.Sprintf("hugo-content-shard-%s-*", shard))
		if err != nil {
			return fmt.Errorf("shard %s: failed to create content dir: %w", shard, err)
		}

		// Generate markdown for entities in this shard
		for _, orgnum := range shardEntities {
			changes, err := cache.GetEntityHistory(orgnum)
			if err != nil {
				return fmt.Errorf("shard %s: entity %s: %w", shard, orgnum, err)
			}

			markdown, err := GenerateTimelineMarkdown(orgnum, changes)
			if err != nil {
				return fmt.Errorf("shard %s: entity %s: %w", shard, orgnum, err)
			}

			if err := WriteEntityMarkdown(orgnum, markdown, shardContentDir); err != nil {
				return fmt.Errorf("shard %s: entity %s: %w", shard, orgnum, err)
			}
		}

		log.Printf("Shard %s: Running Hugo and moving to output", shard)

		// Build this shard with Hugo and move HTML to output
		if err := BuildShardWithHugo(templateDir, shardContentDir, output); err != nil {
			return fmt.Errorf("shard %s: hugo build failed: %w", shard, err)
		}

		log.Printf("Shard %s: Complete (%d entities) - %.1f%% total progress", shard, len(shardEntities), float64(i+1)/float64(total)*100)
	}

	log.Println("\n=== All shards complete ===")

	return nil
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

func runShard(shard, output string, workers int, templatesDir string) error {
	log.Printf("Generating history for shard: %s", shard)

	dataDir := "."
	entityTemplateDir := filepath.Join(templatesDir, "entity-template")
	shardPath := filepath.Join("data", shard)

	// Build git history cache for this shard directory
	cache, err := BuildGitHistoryCacheByShard(dataDir, shardPath)
	if err != nil {
		return fmt.Errorf("failed to build git history cache: %w", err)
	}

	// Get all entities in this shard
	shardEntities := cache.GetAllEntities()
	log.Printf("Found %d entities in shard %s", len(shardEntities), shard)

	if len(shardEntities) == 0 {
		log.Printf("No entities in shard %s, skipping", shard)
		return nil
	}

	// Create temporary content directory
	contentDir, err := os.MkdirTemp("", fmt.Sprintf("hugo-content-shard-%s-*", shard))
	if err != nil {
		return fmt.Errorf("failed to create content dir: %w", err)
	}

	// Generate markdown for entities in this shard
	log.Printf("Generating markdown for %d entities...", len(shardEntities))
	for _, orgnum := range shardEntities {
		changes, err := cache.GetEntityHistory(orgnum)
		if err != nil {
			return fmt.Errorf("entity %s: %w", orgnum, err)
		}

		markdown, err := GenerateTimelineMarkdown(orgnum, changes)
		if err != nil {
			return fmt.Errorf("entity %s: %w", orgnum, err)
		}

		if err := WriteEntityMarkdown(orgnum, markdown, contentDir); err != nil {
			return fmt.Errorf("entity %s: %w", orgnum, err)
		}
	}

	log.Println("Running Hugo to generate HTML...")

	// Build this shard with Hugo and move HTML to output
	if err := BuildShardWithHugo(entityTemplateDir, contentDir, output); err != nil {
		return fmt.Errorf("hugo build failed: %w", err)
	}

	log.Printf("Shard %s complete (%d entities)", shard, len(shardEntities))
	return nil
}
