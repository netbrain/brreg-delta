package main

import (
	"encoding/json"
	"testing"
)

func TestUnderenhetDeletedEntityDetection(t *testing.T) {
	// Simulate the API response for a deleted underenhet
	deletedJSON := `{
		"organisasjonsnummer": "930802409",
		"navn": "LØNNEBLAD",
		"respons_klasse": "SlettetEnhet",
		"slettedato": "2025-11-10"
	}`

	var u Underenhet
	if err := json.Unmarshal([]byte(deletedJSON), &u); err != nil {
		t.Fatalf("Failed to parse deleted underenhet: %v", err)
	}

	if u.Organisasjonsnummer != "930802409" {
		t.Errorf("Expected organisasjonsnummer 930802409, got %s", u.Organisasjonsnummer)
	}

	if u.ResponsKlasse != "SlettetEnhet" {
		t.Errorf("Expected ResponsKlasse 'SlettetEnhet', got '%s'", u.ResponsKlasse)
	}

	if u.OverordnetEnhet != "" {
		t.Errorf("Expected empty OverordnetEnhet for deleted entity, got '%s'", u.OverordnetEnhet)
	}

	// Verify the logic that would be used in processUnderenhetUpdate
	if u.ResponsKlasse == "SlettetEnhet" {
		t.Log("✓ Deleted entity correctly detected via ResponsKlasse")
	} else if u.OverordnetEnhet == "" {
		t.Error("Would have triggered 'underenhet has no parent' error instead of detecting deletion")
	}
}

func TestEnhetDeletedEntityDetection(t *testing.T) {
	// Simulate the API response for a deleted enhet
	deletedJSON := `{
		"organisasjonsnummer": "123456789",
		"navn": "TEST COMPANY",
		"respons_klasse": "SlettetEnhet",
		"slettedato": "2025-11-10"
	}`

	var e Enhet
	if err := json.Unmarshal([]byte(deletedJSON), &e); err != nil {
		t.Fatalf("Failed to parse deleted enhet: %v", err)
	}

	if e.Organisasjonsnummer != "123456789" {
		t.Errorf("Expected organisasjonsnummer 123456789, got %s", e.Organisasjonsnummer)
	}

	if e.ResponsKlasse != "SlettetEnhet" {
		t.Errorf("Expected ResponsKlasse 'SlettetEnhet', got '%s'", e.ResponsKlasse)
	}

	if e.ResponsKlasse == "SlettetEnhet" {
		t.Log("✓ Deleted enhet correctly detected via ResponsKlasse")
	}
}

func TestUnderenhetActiveEntityWithParent(t *testing.T) {
	// Simulate a normal active underenhet with parent
	activeJSON := `{
		"organisasjonsnummer": "914930553",
		"navn": "ACTIVE SUB-UNIT",
		"overordnetEnhet": "810034882"
	}`

	var u Underenhet
	if err := json.Unmarshal([]byte(activeJSON), &u); err != nil {
		t.Fatalf("Failed to parse active underenhet: %v", err)
	}

	if u.Organisasjonsnummer != "914930553" {
		t.Errorf("Expected organisasjonsnummer 914930553, got %s", u.Organisasjonsnummer)
	}

	if u.OverordnetEnhet != "810034882" {
		t.Errorf("Expected OverordnetEnhet 810034882, got '%s'", u.OverordnetEnhet)
	}

	if u.ResponsKlasse != "" {
		t.Logf("ResponsKlasse present: '%s'", u.ResponsKlasse)
	}

	// Verify this would NOT be treated as deleted
	if u.ResponsKlasse == "SlettetEnhet" {
		t.Error("Active entity incorrectly flagged as deleted")
	}

	if u.OverordnetEnhet == "" {
		t.Error("Active entity missing parent reference")
	} else {
		t.Log("✓ Active entity has parent reference")
	}
}
