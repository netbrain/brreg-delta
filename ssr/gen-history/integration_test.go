package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestIntegration_NoEntityMixing creates a real git repository with multiple entities
// and verifies that the HTML generation doesn't mix them up
func TestIntegration_NoEntityMixing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create temporary directory for test git repo
	tmpDir, err := os.MkdirTemp("", "gen-history-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize git repo
	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "config", "user.name", "Test User")
	runGit(t, tmpDir, "config", "user.email", "test@example.com")

	// Create data directory structure
	dataDir := filepath.Join(tmpDir, "data")

	// Entity 1: 814584542 at data/814/584/542/
	entity1Path := filepath.Join(dataDir, "814", "584", "542")
	os.MkdirAll(entity1Path, 0755)

	// Entity 2: 814539482 at data/814/539/482/
	entity2Path := filepath.Join(dataDir, "814", "539", "482")
	os.MkdirAll(entity2Path, 0755)

	// Commit 1: Initial creation of both entities (on 10.11.2025)
	writeJSON(t, filepath.Join(entity1Path, "enhet.json"), map[string]interface{}{
		"organisasjonsnummer": "814584542",
		"navn":                "TRIO FEDALA AS",
		"forretningsadresse": map[string]interface{}{
			"adresse": []string{"Kirkegata 15"},
		},
	})

	writeJSON(t, filepath.Join(entity2Path, "enhet.json"), map[string]interface{}{
		"organisasjonsnummer": "814539482",
		"navn":                "G.L. ETTERLATTE FORENING",
		"forretningsadresse": map[string]interface{}{
			"adresse": []string{"Berglyveien 20C"},
		},
	})

	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "Initial sync", "--date=2025-11-10T10:00:00")

	// Commit 2: Update entity 1 only (on 13.11.2025)
	writeJSON(t, filepath.Join(entity1Path, "enhet.json"), map[string]interface{}{
		"organisasjonsnummer": "814584542",
		"navn":                "TRIO FEDALA AS",
		"forretningsadresse": map[string]interface{}{
			"adresse": []string{"Kirkegata 15"},
		},
		"_links": map[string]interface{}{
			"self": map[string]interface{}{
				"href": "https://data.brreg.no/enhetsregisteret/api/enheter/814584542",
			},
		},
	})

	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "Update entity 814584542", "--date=2025-11-13T14:00:00")

	// Commit 3: Update entity 1 again (on 18.11.2025) - just address change
	writeJSON(t, filepath.Join(entity1Path, "enhet.json"), map[string]interface{}{
		"organisasjonsnummer": "814584542",
		"navn":                "TRIO FEDALA AS",
		"forretningsadresse": map[string]interface{}{
			"adresse": []string{"Markens gate 15"}, // Changed!
		},
		"_links": map[string]interface{}{
			"self": map[string]interface{}{
				"href": "https://data.brreg.no/enhetsregisteret/api/enheter/814584542",
			},
		},
	})

	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "Update address for 814584542", "--date=2025-11-18T16:00:00")

	// Commit 4: Update entity 2 separately (on 13.11.2025, different time)
	writeJSON(t, filepath.Join(entity2Path, "enhet.json"), map[string]interface{}{
		"organisasjonsnummer": "814539482",
		"navn":                "G.L. ETTERLATTE FORENING",
		"forretningsadresse": map[string]interface{}{
			"adresse": []string{"Updated Address"},
		},
		"_links": map[string]interface{}{
			"self": map[string]interface{}{
				"href": "https://data.brreg.no/enhetsregisteret/api/enheter/814539482",
			},
		},
	})

	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "Update entity 814539482", "--date=2025-11-13T15:00:00")

	// Now get the history for entity 814584542
	changes, err := GetEntityHistory(tmpDir, "814584542")
	if err != nil {
		t.Fatalf("GetEntityHistory failed: %v", err)
	}

	t.Logf("Found %d changes for entity 814584542", len(changes))

	// CRITICAL ASSERTION: All changes should be for entity 814584542 only
	for i, change := range changes {
		var obj map[string]interface{}
		if err := json.Unmarshal(change.Data, &obj); err != nil {
			t.Errorf("Change %d: failed to unmarshal: %v", i, err)
			continue
		}

		orgnum, ok := obj["organisasjonsnummer"].(string)
		if !ok {
			t.Errorf("Change %d: missing organisasjonsnummer", i)
			continue
		}

		if orgnum != "814584542" {
			t.Errorf("Change %d: ENTITY MIXING DETECTED! Got orgnum %s, expected 814584542", i, orgnum)
			t.Logf("  CommitHash: %s", change.CommitHash)
			t.Logf("  FilePath: %s", change.FilePath)
			t.Logf("  Date: %s", change.Date)
			t.Logf("  Message: %s", change.Message)
		}

		// Also check the name
		navn, _ := obj["navn"].(string)
		if navn == "G.L. ETTERLATTE FORENING" {
			t.Errorf("Change %d: Found entity 814539482's name in entity 814584542's history!", i)
		}
	}

	// Now generate markdown and verify no mixing there either
	markdown, err := GenerateTimelineMarkdown("814584542", changes)
	if err != nil {
		t.Fatalf("GenerateTimelineMarkdown failed: %v", err)
	}

	// Verify markdown doesn't contain entity 814539482 data
	if strings.Contains(markdown, "814539482") {
		t.Error("Markdown contains orgnum 814539482 - entities are being mixed in markdown generation!")
	}

	if strings.Contains(markdown, "G.L. ETTERLATTE FORENING") {
		t.Error("Markdown contains entity 814539482's name - entities are being mixed!")
	}

	// Verify markdown DOES contain entity 814584542 data
	if !strings.Contains(markdown, "814584542") {
		t.Error("Markdown missing orgnum 814584542")
	}

	if !strings.Contains(markdown, "TRIO FEDALA AS") {
		t.Error("Markdown missing entity 814584542's name")
	}

	// Verify the address change is captured correctly
	if !strings.Contains(markdown, "Kirkegata") || !strings.Contains(markdown, "Markens gate") {
		t.Error("Markdown missing the address change from Kirkegata to Markens gate")
	}

	// Verify we have the correct number of date sections (3 commits for entity 814584542)
	dateCount := strings.Count(markdown, "<summary><strong>")
	if dateCount != 3 {
		t.Errorf("Expected 3 date sections (10.11, 13.11, 18.11), got %d", dateCount)
	}

	t.Logf("Generated markdown length: %d bytes", len(markdown))
	t.Logf("Markdown preview:\n%s", markdown[:min(500, len(markdown))])
}

func runGit(t *testing.T, dir string, args ...string) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\nOutput: %s", args, err, output)
	}
}

func writeJSON(t *testing.T, path string, data interface{}) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	if err := os.WriteFile(path, jsonData, 0644); err != nil {
		t.Fatalf("Failed to write file %s: %v", path, err)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
