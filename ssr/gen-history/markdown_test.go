package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestEscapeYAMLString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    `Simple text`,
			expected: `Simple text`,
		},
		{
			input:    `Text with "quotes" inside`,
			expected: `Text with \"quotes\" inside`,
		},
		{
			input:    `Text with \ backslash`,
			expected: `Text with \\ backslash`,
		},
		{
			input:    `Text with both "quotes" and \ backslash`,
			expected: `Text with both \"quotes\" and \\ backslash`,
		},
	}

	for _, tt := range tests {
		result := escapeYAMLString(tt.input)
		if result != tt.expected {
			t.Errorf("escapeYAMLString(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestGenerateTimelineMarkdown_SameDateMultipleCommits(t *testing.T) {
	// This test verifies that when multiple commits occur on the same date,
	// the diff is calculated against the previous date's change, not another
	// change from the same date.

	// Create test data: 3 changes on 2 different dates
	// - 2025-11-13: Two changes to roller.json (different commits)
	// - 2025-11-10: One change to roller.json

	date1 := time.Date(2025, 11, 10, 10, 0, 0, 0, time.UTC)
	date2a := time.Date(2025, 11, 13, 14, 0, 0, 0, time.UTC)
	date2b := time.Date(2025, 11, 13, 16, 0, 0, 0, time.UTC)

	changes := []EntityChange{
		// Most recent first (commit on 13.11 at 16:00)
		{
			CommitHash: "commit2b",
			Date:       date2b,
			Message:    "Update roller 2",
			FilePath:   "data/814/584/542/roller.json",
			Data:       json.RawMessage(`{"organisasjonsnummer":"814584542","rollegrupper":[{"type":"styre","roller":[{"person":{"navn":"Alice"}}]}]}`),
		},
		// Second commit on same date (13.11 at 14:00)
		{
			CommitHash: "commit2a",
			Date:       date2a,
			Message:    "Update roller 1",
			FilePath:   "data/814/584/542/roller.json",
			Data:       json.RawMessage(`{"organisasjonsnummer":"814584542","rollegrupper":[{"type":"styre","roller":[{"person":{"navn":"Bob"}}]}]}`),
		},
		// Original commit (10.11)
		{
			CommitHash: "commit1",
			Date:       date1,
			Message:    "Initial roller",
			FilePath:   "data/814/584/542/roller.json",
			Data:       json.RawMessage(`{"organisasjonsnummer":"814584542","rollegrupper":[]}`),
		},
	}

	markdown, err := GenerateTimelineMarkdown("814584542", changes)
	if err != nil {
		t.Fatalf("GenerateTimelineMarkdown failed: %v", err)
	}

	// Verify the markdown contains the date grouping
	if !strings.Contains(markdown, "13.11.2025") {
		t.Error("Expected markdown to contain '13.11.2025'")
	}

	if !strings.Contains(markdown, "10.11.2025") {
		t.Error("Expected markdown to contain '10.11.2025'")
	}

	// The key assertion: changes on 13.11 should be diffed against 10.11, not against each other
	// So we should NOT see changes that would only appear if comparing commit2b vs commit2a
	// Both commit2a and commit2b should be compared against commit1 (the 10.11 change)

	// Since both changes on 13.11 add roller entries (vs empty array on 10.11),
	// we should see "La til" (added) messages, not changes between Alice and Bob
	if strings.Contains(markdown, "Alice") && strings.Contains(markdown, "Bob") {
		// If both names appear, verify they're not being compared against each other
		// We expect to see additions, not name changes
		if strings.Contains(markdown, "endret fra") && strings.Contains(markdown, "Alice") && strings.Contains(markdown, "Bob") {
			t.Error("Expected changes to be compared against previous date, not within same date")
		}
	}

	// Verify we see role additions (since we're going from empty array to populated)
	if !strings.Contains(markdown, "Rolleendringer") {
		t.Error("Expected to see 'Rolleendringer' section")
	}
}

func TestGroupChangesByCommit(t *testing.T) {
	// Test that changes are properly grouped by date (not commit hash)

	date1 := time.Date(2025, 11, 10, 10, 0, 0, 0, time.UTC)
	date2a := time.Date(2025, 11, 13, 14, 0, 0, 0, time.UTC)
	date2b := time.Date(2025, 11, 13, 16, 0, 0, 0, time.UTC)

	changes := []EntityChange{
		{CommitHash: "c2b", Date: date2b, FilePath: "roller.json"},
		{CommitHash: "c2a", Date: date2a, FilePath: "enhet.json"},
		{CommitHash: "c1", Date: date1, FilePath: "roller.json"},
	}

	groups := groupChangesByCommit(changes)

	if len(groups) != 2 {
		t.Errorf("Expected 2 groups (2 dates), got %d", len(groups))
	}

	// First group should be 13.11 with 2 changes
	if groups[0].Date.Format("2006-01-02") != "2025-11-13" {
		t.Errorf("Expected first group to be 2025-11-13, got %s", groups[0].Date.Format("2006-01-02"))
	}

	if len(groups[0].Changes) != 2 {
		t.Errorf("Expected first group to have 2 changes, got %d", len(groups[0].Changes))
	}

	// Second group should be 10.11 with 1 change
	if groups[1].Date.Format("2006-01-02") != "2025-11-10" {
		t.Errorf("Expected second group to be 2025-11-10, got %s", groups[1].Date.Format("2006-01-02"))
	}

	if len(groups[1].Changes) != 1 {
		t.Errorf("Expected second group to have 1 change, got %d", len(groups[1].Changes))
	}
}
