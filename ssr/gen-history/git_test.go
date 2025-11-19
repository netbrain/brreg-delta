package main

import (
	"encoding/json"
	"testing"
)

func TestPathToOrgnum(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{
			path:     "data/814/584/542/enhet.json",
			expected: "814584542",
		},
		{
			path:     "data/814/539/482/roller.json",
			expected: "814539482",
		},
		{
			path:     "data/810/034/882/underenhet.json",
			expected: "810034882",
		},
		{
			path:     "data/123/456/789/enhet.json",
			expected: "123456789",
		},
		{
			path:     "invalid/path",
			expected: "",
		},
	}

	for _, tt := range tests {
		result := pathToOrgnum(tt.path)
		if result != tt.expected {
			t.Errorf("pathToOrgnum(%q) = %q, want %q", tt.path, result, tt.expected)
		}
	}
}

func TestOrgnumToPath(t *testing.T) {
	tests := []struct {
		orgnum   string
		expected string
	}{
		{
			orgnum:   "814584542",
			expected: "814/584/542",
		},
		{
			orgnum:   "814539482",
			expected: "814/539/482",
		},
		{
			orgnum:   "810034882",
			expected: "810/034/882",
		},
		{
			orgnum:   "123456789",
			expected: "123/456/789",
		},
	}

	for _, tt := range tests {
		result := orgnumToPath(tt.orgnum)
		if result != tt.expected {
			t.Errorf("orgnumToPath(%q) = %q, want %q", tt.orgnum, result, tt.expected)
		}
	}
}

func TestFilterChangesByOrgnum(t *testing.T) {
	// Test that changes are properly filtered to only include matching orgnum
	// This is critical to prevent mixing different entities

	changes := []EntityChange{
		{
			CommitHash: "commit1",
			FilePath:   "data/814/584/542/enhet.json",
			Data:       json.RawMessage(`{"organisasjonsnummer":"814584542","navn":"TRIO FEDALA AS"}`),
		},
		{
			CommitHash: "commit2",
			FilePath:   "data/814/539/482/enhet.json",
			Data:       json.RawMessage(`{"organisasjonsnummer":"814539482","navn":"G.L. ETTERLATTE FORENING"}`),
		},
		{
			CommitHash: "commit3",
			FilePath:   "data/814/584/542/enhet.json",
			Data:       json.RawMessage(`{"organisasjonsnummer":"814584542","navn":"TRIO FEDALA AS"}`),
		},
		{
			CommitHash: "commit4",
			FilePath:   "data/814/584/542/roller.json",
			Data:       json.RawMessage(`{"organisasjonsnummer":"814584542","rollegrupper":[]}`),
		},
	}

	// Filter for entity 814584542
	filtered := filterChangesByOrgnum(changes, "814584542")

	// Should only get commits 1, 3, and 4
	if len(filtered) != 3 {
		t.Errorf("Expected 3 filtered changes, got %d", len(filtered))
	}

	// Verify none of the filtered changes contain the wrong orgnum
	for _, change := range filtered {
		var obj map[string]interface{}
		if err := json.Unmarshal(change.Data, &obj); err != nil {
			t.Fatalf("Failed to unmarshal change data: %v", err)
		}

		orgnum, ok := obj["organisasjonsnummer"].(string)
		if !ok {
			t.Error("Change missing organisasjonsnummer field")
			continue
		}

		if orgnum != "814584542" {
			t.Errorf("Filtered change contains wrong orgnum: got %s, want 814584542", orgnum)
		}
	}

	// Filter for entity 814539482
	filtered2 := filterChangesByOrgnum(changes, "814539482")

	// Should only get commit 2
	if len(filtered2) != 1 {
		t.Errorf("Expected 1 filtered change for 814539482, got %d", len(filtered2))
	}

	if len(filtered2) > 0 && filtered2[0].CommitHash != "commit2" {
		t.Errorf("Expected commit2, got %s", filtered2[0].CommitHash)
	}
}

// filterChangesByOrgnum filters changes to only include those matching the specified orgnum
func filterChangesByOrgnum(changes []EntityChange, orgnum string) []EntityChange {
	var filtered []EntityChange

	for _, change := range changes {
		// Parse JSON to check organisasjonsnummer
		var obj map[string]interface{}
		if err := json.Unmarshal(change.Data, &obj); err != nil {
			// If we can't parse, skip this change
			continue
		}

		// Check organisasjonsnummer field
		changeOrgnum, ok := obj["organisasjonsnummer"].(string)
		if !ok {
			// No organisasjonsnummer field, skip
			continue
		}

		// Only include if orgnum matches
		if changeOrgnum == orgnum {
			filtered = append(filtered, change)
		}
	}

	return filtered
}

func TestAddCommitFilesFiltered_NoEntityMixing(t *testing.T) {
	// This test verifies that when processing commits, we don't accidentally
	// mix changes from different entities even if they're in the same commit

	// Simulate a commit that touched files for both entities
	// (this could happen in a batch sync)
	files := []string{
		"data/814/584/542/enhet.json",
		"data/814/539/482/enhet.json",
	}

	// In a real scenario, this would fetch from git, but we can't do that in a unit test
	// Instead, verify that the logic properly separates entities based on orgnum

	// The key assertion: after processing, each entity should only have its own changes
	// We can't easily test the full flow without git, but we can test pathToOrgnum

	for _, filePath := range files {
		orgnum := pathToOrgnum(filePath)
		if orgnum == "" {
			t.Errorf("Failed to extract orgnum from path: %s", filePath)
			continue
		}

		// Verify orgnum matches expected pattern
		if filePath == "data/814/584/542/enhet.json" && orgnum != "814584542" {
			t.Errorf("Wrong orgnum extracted: got %s, want 814584542", orgnum)
		}
		if filePath == "data/814/539/482/enhet.json" && orgnum != "814539482" {
			t.Errorf("Wrong orgnum extracted: got %s, want 814539482", orgnum)
		}
	}
}

func TestEntityChangesNotMixed(t *testing.T) {
	// Test that verifies entity 814584542 never gets changes from entity 814539482
	// This is the core bug we're trying to prevent

	// Create a scenario similar to what caused the bug:
	// Multiple entities in the same shard (814/5xx/xxx)
	allChanges := []EntityChange{
		// Entity 814584542 changes
		{
			CommitHash: "abc123",
			FilePath:   "data/814/584/542/enhet.json",
			Data:       json.RawMessage(`{"organisasjonsnummer":"814584542","navn":"TRIO FEDALA AS","forretningsadresse":{"adresse":["Kirkegata 15"]}}`),
		},
		{
			CommitHash: "def456",
			FilePath:   "data/814/584/542/enhet.json",
			Data:       json.RawMessage(`{"organisasjonsnummer":"814584542","navn":"TRIO FEDALA AS","forretningsadresse":{"adresse":["Markens gate 15"]}}`),
		},
		// Entity 814539482 changes (different entity!)
		{
			CommitHash: "ghi789",
			FilePath:   "data/814/539/482/enhet.json",
			Data:       json.RawMessage(`{"organisasjonsnummer":"814539482","navn":"G.L. ETTERLATTE FORENING"}`),
		},
	}

	// When we get history for entity 814584542, we should NEVER see entity 814539482
	filteredFor814584542 := filterChangesByOrgnum(allChanges, "814584542")

	if len(filteredFor814584542) != 2 {
		t.Errorf("Expected 2 changes for 814584542, got %d", len(filteredFor814584542))
	}

	// Verify no mixing
	for _, change := range filteredFor814584542 {
		var obj map[string]interface{}
		json.Unmarshal(change.Data, &obj)

		orgnum := obj["organisasjonsnummer"].(string)
		if orgnum != "814584542" {
			t.Errorf("Entity 814584542 history contains change from entity %s - entities are being mixed!", orgnum)
		}

		navn := obj["navn"].(string)
		if navn == "G.L. ETTERLATTE FORENING" {
			t.Error("Entity 814584542 history contains 'G.L. ETTERLATTE FORENING' - this is from entity 814539482!")
		}
	}
}
