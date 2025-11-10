package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// GenerateEntityHTML generates HTML for a single entity using Hugo
func GenerateEntityHTML(orgnum, markdown, templateDir, outputDir string) error {
	// Create temporary directory for Hugo site
	tmpDir, err := os.MkdirTemp("", fmt.Sprintf("entity-%s-*", orgnum))
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Copy template to temp directory
	if err := copyDir(templateDir, tmpDir); err != nil {
		return fmt.Errorf("failed to copy template: %w", err)
	}

	// Write markdown content
	contentDir := filepath.Join(tmpDir, "content")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		return fmt.Errorf("failed to create content dir: %w", err)
	}

	mdPath := filepath.Join(contentDir, "_index.md")
	if err := os.WriteFile(mdPath, []byte(markdown), 0644); err != nil {
		return fmt.Errorf("failed to write markdown: %w", err)
	}

	// Run Hugo
	hugoOutputDir := filepath.Join(tmpDir, "public")
	if err := runHugo(tmpDir, hugoOutputDir); err != nil {
		return fmt.Errorf("hugo build failed: %w", err)
	}

	// Copy generated HTML to final location
	htmlSource := filepath.Join(hugoOutputDir, "index.html")
	htmlDest := filepath.Join(outputDir, orgnumToPath(orgnum)+".html")

	if err := os.MkdirAll(filepath.Dir(htmlDest), 0755); err != nil {
		return fmt.Errorf("failed to create output dir: %w", err)
	}

	if err := copyFile(htmlSource, htmlDest); err != nil {
		return fmt.Errorf("failed to copy HTML: %w", err)
	}

	// Copy assets (CSS, JS) if they exist
	if err := copyAssets(hugoOutputDir, filepath.Dir(outputDir)); err != nil {
		// Not fatal, just log
		fmt.Printf("Warning: failed to copy assets: %v\n", err)
	}

	return nil
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

// copyAssets copies CSS and JS files to the output directory
func copyAssets(hugoOutputDir, baseOutputDir string) error {
	// Copy CSS
	cssSrc := filepath.Join(hugoOutputDir, "css")
	cssDst := filepath.Join(baseOutputDir, "css")
	if _, err := os.Stat(cssSrc); err == nil {
		if err := os.MkdirAll(cssDst, 0755); err != nil {
			return err
		}
		if err := copyDir(cssSrc, cssDst); err != nil {
			return err
		}
	}

	// Copy JS
	jsSrc := filepath.Join(hugoOutputDir, "js")
	jsDst := filepath.Join(baseOutputDir, "js")
	if _, err := os.Stat(jsSrc); err == nil {
		if err := os.MkdirAll(jsDst, 0755); err != nil {
			return err
		}
		if err := copyDir(jsSrc, jsDst); err != nil {
			return err
		}
	}

	return nil
}
