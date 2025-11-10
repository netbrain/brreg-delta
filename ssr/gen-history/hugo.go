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

// BuildAllWithHugo runs Hugo once to build all entity pages
func BuildAllWithHugo(templateDir, contentDir, outputDir string) error {
	// Create temporary Hugo site
	tmpDir, err := os.MkdirTemp("", "hugo-build-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Copy template to temp directory
	if err := copyDir(templateDir, tmpDir); err != nil {
		return fmt.Errorf("failed to copy template: %w", err)
	}

	// Copy content directory to Hugo site
	hugoContentDir := filepath.Join(tmpDir, "content")
	if err := os.RemoveAll(hugoContentDir); err != nil {
		return fmt.Errorf("failed to remove default content: %w", err)
	}
	if err := copyDir(contentDir, hugoContentDir); err != nil {
		return fmt.Errorf("failed to copy content: %w", err)
	}

	// Run Hugo
	hugoOutputDir := filepath.Join(tmpDir, "public")
	fmt.Println("Running Hugo to generate all pages...")
	if err := runHugo(tmpDir, hugoOutputDir); err != nil {
		return fmt.Errorf("hugo build failed: %w", err)
	}

	// Copy all generated HTML files to output directory
	fmt.Println("Copying generated HTML files to output...")
	if err := copyDir(hugoOutputDir, outputDir); err != nil {
		return fmt.Errorf("failed to copy output: %w", err)
	}

	return nil
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
