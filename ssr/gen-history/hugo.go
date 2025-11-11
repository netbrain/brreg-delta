package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// WriteEntityMarkdown writes markdown for an entity to the Hugo content directory
func WriteEntityMarkdown(orgnum, markdown, contentDir string) error {
	// Create section directory: content/810/034/882/
	sectionPath := filepath.Join(contentDir, orgnumToPath(orgnum))
	if err := os.MkdirAll(sectionPath, 0755); err != nil {
		return fmt.Errorf("failed to create section dir: %w", err)
	}

	// Write _index.md
	mdPath := filepath.Join(sectionPath, "_index.md")
	if err := os.WriteFile(mdPath, []byte(markdown), 0644); err != nil {
		return fmt.Errorf("failed to write markdown: %w", err)
	}

	return nil
}

// BuildAllWithHugo builds Hugo site (template + content) and generates HTML
func BuildAllWithHugo(templateDir, contentDir, outputDir string) error {
	// Create temporary Hugo build directory
	tmpHugoSite, err := os.MkdirTemp("", "hugo-incremental-*")
	if err != nil {
		return fmt.Errorf("failed to create temp hugo site: %w", err)
	}
	defer os.RemoveAll(tmpHugoSite)

	// Copy template to temp site
	fmt.Println("Copying Hugo template to temp directory...")
	if err := copyDir(templateDir, tmpHugoSite); err != nil {
		return fmt.Errorf("failed to copy template: %w", err)
	}

	// Move content directory to Hugo site (rename is much faster than copy)
	hugoContentDir := filepath.Join(tmpHugoSite, "content")
	fmt.Println("Moving generated markdown content...")
	if err := os.RemoveAll(hugoContentDir); err != nil {
		return fmt.Errorf("failed to remove default content: %w", err)
	}
	if err := os.Rename(contentDir, hugoContentDir); err != nil {
		return fmt.Errorf("failed to move content: %w", err)
	}

	// Run Hugo to generate HTML
	fmt.Println("Running Hugo to generate HTML...")
	publicDir := filepath.Join(tmpHugoSite, "public")
	if err := runHugo(tmpHugoSite, publicDir); err != nil {
		return fmt.Errorf("hugo build failed: %w", err)
	}

	// Move generated HTML to final output (merge, don't overwrite entire directory)
	fmt.Println("Moving generated HTML to output directory...")
	if err := moveGeneratedHTML(publicDir, outputDir); err != nil {
		return fmt.Errorf("failed to move HTML output: %w", err)
	}

	fmt.Println("Incremental build complete")
	return nil
}

// BuildShardWithHugo processes a single shard: markdown -> Hugo -> move HTML to output
func BuildShardWithHugo(templateDir, contentDir, finalOutputDir string) error {
	// Create temporary Hugo build directory
	tmpHugoSite, err := os.MkdirTemp("", "hugo-shard-*")
	if err != nil {
		return fmt.Errorf("failed to create temp hugo site: %w", err)
	}
	defer os.RemoveAll(tmpHugoSite)

	// Copy template to temp site
	if err := copyDir(templateDir, tmpHugoSite); err != nil {
		return fmt.Errorf("failed to copy template: %w", err)
	}

	// Move content to temp site
	hugoContentDir := filepath.Join(tmpHugoSite, "content")
	if err := os.RemoveAll(hugoContentDir); err != nil {
		return fmt.Errorf("failed to remove default content: %w", err)
	}
	if err := os.Rename(contentDir, hugoContentDir); err != nil {
		return fmt.Errorf("failed to move content: %w", err)
	}

	// Run Hugo to generate HTML
	publicDir := filepath.Join(tmpHugoSite, "public")
	if err := runHugo(tmpHugoSite, publicDir); err != nil {
		return fmt.Errorf("hugo build failed: %w", err)
	}

	// Move generated HTML to final output (merge, don't overwrite entire directory)
	if err := moveGeneratedHTML(publicDir, finalOutputDir); err != nil {
		return fmt.Errorf("failed to move HTML output: %w", err)
	}

	return nil
}

// moveGeneratedHTML moves HTML files from Hugo's public/ to final output, merging directories
func moveGeneratedHTML(publicDir, finalOutputDir string) error {
	return filepath.Walk(publicDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path
		relPath, err := filepath.Rel(publicDir, path)
		if err != nil {
			return err
		}

		// Skip the root directory
		if relPath == "." {
			return nil
		}

		dstPath := filepath.Join(finalOutputDir, relPath)

		if info.IsDir() {
			// Create directory in destination
			return os.MkdirAll(dstPath, info.Mode())
		}

		// Move file (use rename if possible, otherwise copy+delete)
		if err := os.Rename(path, dstPath); err != nil {
			// If rename fails (cross-device), copy then delete
			if err := copyFile(path, dstPath); err != nil {
				return err
			}
			return os.Remove(path)
		}

		return nil
	})
}

// GenerateEntityHTML generates HTML for a single entity using Hugo (legacy single-entity mode)
func GenerateEntityHTML(orgnum, markdown, templateDir, outputDir string) error {
	// Create temporary content directory
	tmpContentDir, err := os.MkdirTemp("", "hugo-content-*")
	if err != nil {
		return fmt.Errorf("failed to create temp content dir: %w", err)
	}
	defer os.RemoveAll(tmpContentDir)

	// Write markdown
	if err := WriteEntityMarkdown(orgnum, markdown, tmpContentDir); err != nil {
		return err
	}

	// Build with Hugo
	return BuildAllWithHugo(templateDir, tmpContentDir, outputDir)
}

// runHugo executes Hugo to build the site
func runHugo(sourceDir, outputDir string) error {
	cmd := exec.Command("hugo", "--minify", "--destination", outputDir)
	cmd.Dir = sourceDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("hugo command failed: %w", err)
	}

	return nil
}

// copyDir recursively copies a directory
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate destination path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		// Skip .git directories and other hidden files
		if info.Name() != "." && info.Name()[0] == '.' {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		return copyFile(path, dstPath)
	})
}

// copyFile copies a single file
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	return dstFile.Sync()
}

// BuildNavigationSite builds the navigation-site and copies index.html to output
func BuildNavigationSite(navSiteDir, outputDir string) error {
	// Run Hugo in the navigation site directory
	publicDir := filepath.Join(navSiteDir, "public")
	if err := runHugo(navSiteDir, publicDir); err != nil {
		return fmt.Errorf("failed to build navigation site: %w", err)
	}

	// Copy index.html to output directory
	indexSrc := filepath.Join(publicDir, "index.html")
	indexDst := filepath.Join(outputDir, "index.html")
	if err := copyFile(indexSrc, indexDst); err != nil {
		return fmt.Errorf("failed to copy index.html: %w", err)
	}

	fmt.Println("Navigation site built and index.html copied to output")
	return nil
}
