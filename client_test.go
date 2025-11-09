package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"os"
	"testing"
)

// Sample data matching the actual Brreg.no format
const sampleCompanyJSON = `[
  {
    "organisasjonsnummer": "810034882",
    "navn": "SANDNES ELEKTRISKE AS",
    "organisasjonsform": {
      "kode": "AS",
      "beskrivelse": "Aksjeselskap",
      "links": []
    },
    "postadresse": {
      "land": "Norge",
      "landkode": "NO",
      "postnummer": "4301",
      "poststed": "SANDNES",
      "adresse": ["Postboks 32"],
      "kommune": "SANDNES",
      "kommunenummer": "1108"
    },
    "registreringsdatoEnhetsregisteret": "1995-02-19",
    "registrertIMvaregisteret": true,
    "naeringskode1": {
      "kode": "43.210",
      "beskrivelse": "Elektrisk installasjonsarbeid"
    },
    "antallAnsatte": 8,
    "konkurs": false,
    "underAvvikling": false
  },
  {
    "organisasjonsnummer": "810059672",
    "navn": "AASEN & FARSTAD AS",
    "organisasjonsform": {
      "kode": "AS",
      "beskrivelse": "Aksjeselskap",
      "links": []
    },
    "registreringsdatoEnhetsregisteret": "1995-02-19",
    "registrertIMvaregisteret": false,
    "naeringskode1": {
      "kode": "68.200",
      "beskrivelse": "Utleie av egen eller leid fast eiendom"
    },
    "konkurs": false,
    "underAvvikling": false
  },
  {
    "organisasjonsnummer": "123456789",
    "navn": "TEST COMPANY AS",
    "organisasjonsform": {
      "kode": "AS",
      "beskrivelse": "Aksjeselskap",
      "links": []
    }
  }
]`

func TestParseCompanyList(t *testing.T) {
	var companies []Company
	err := json.Unmarshal([]byte(sampleCompanyJSON), &companies)
	if err != nil {
		t.Fatalf("Failed to parse company list: %v", err)
	}

	if len(companies) != 3 {
		t.Errorf("Expected 3 companies, got %d", len(companies))
	}

	// Check first company
	if companies[0].Organisasjonsnummer != "810034882" {
		t.Errorf("Expected orgnum 810034882, got %s", companies[0].Organisasjonsnummer)
	}

	// Check that RawData is populated
	if len(companies[0].RawData) == 0 {
		t.Error("RawData should be populated")
	}

	// Verify RawData contains the full company data
	var parsed map[string]interface{}
	if err := json.Unmarshal(companies[0].RawData, &parsed); err != nil {
		t.Errorf("RawData should be valid JSON: %v", err)
	}

	if parsed["navn"] != "SANDNES ELEKTRISKE AS" {
		t.Errorf("RawData should contain company name")
	}
}

func TestLoadCompanyListFromGzipFile(t *testing.T) {
	// Create temporary gzipped file
	tmpFile, err := os.CreateTemp("", "companies-*.json.gz")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write gzipped data
	gzWriter := gzip.NewWriter(tmpFile)
	_, err = gzWriter.Write([]byte(sampleCompanyJSON))
	if err != nil {
		t.Fatalf("Failed to write gzip data: %v", err)
	}
	gzWriter.Close()
	tmpFile.Close()

	// Test loading
	client := NewClient()
	companies, err := client.LoadCompanyListFromFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load company list from gzip file: %v", err)
	}

	if len(companies) != 3 {
		t.Errorf("Expected 3 companies, got %d", len(companies))
	}

	if companies[1].Organisasjonsnummer != "810059672" {
		t.Errorf("Expected orgnum 810059672, got %s", companies[1].Organisasjonsnummer)
	}
}

func TestLoadCompanyListFromPlainFile(t *testing.T) {
	// Create temporary plain JSON file
	tmpFile, err := os.CreateTemp("", "companies-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.Write([]byte(sampleCompanyJSON))
	if err != nil {
		t.Fatalf("Failed to write data: %v", err)
	}
	tmpFile.Close()

	// Test loading
	client := NewClient()
	companies, err := client.LoadCompanyListFromFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load company list from plain file: %v", err)
	}

	if len(companies) != 3 {
		t.Errorf("Expected 3 companies, got %d", len(companies))
	}
}

func TestGzipDetection(t *testing.T) {
	tests := []struct {
		name       string
		data       []byte
		shouldGzip bool
	}{
		{
			name:       "Plain JSON",
			data:       []byte(sampleCompanyJSON),
			shouldGzip: false,
		},
		{
			name:       "Gzipped JSON",
			data:       gzipData([]byte(sampleCompanyJSON)),
			shouldGzip: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "test-*.json")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			_, err = tmpFile.Write(tt.data)
			if err != nil {
				t.Fatalf("Failed to write data: %v", err)
			}
			tmpFile.Close()

			client := NewClient()
			companies, err := client.LoadCompanyListFromFile(tmpFile.Name())
			if err != nil {
				t.Fatalf("Failed to load file: %v", err)
			}

			if len(companies) != 3 {
				t.Errorf("Expected 3 companies, got %d", len(companies))
			}
		})
	}
}

func gzipData(data []byte) []byte {
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	gzWriter.Write(data)
	gzWriter.Close()
	return buf.Bytes()
}

func TestCompanyDataPreservation(t *testing.T) {
	// Ensure that when we unmarshal, we preserve all fields in RawData
	singleCompany := `{
		"organisasjonsnummer": "810034882",
		"navn": "SANDNES ELEKTRISKE AS",
		"antallAnsatte": 8,
		"konkurs": false,
		"custom_field": "should be preserved"
	}`

	var company Company
	err := json.Unmarshal([]byte(singleCompany), &company)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify RawData contains everything
	var rawMap map[string]interface{}
	err = json.Unmarshal(company.RawData, &rawMap)
	if err != nil {
		t.Fatalf("RawData is not valid JSON: %v", err)
	}

	if rawMap["custom_field"] != "should be preserved" {
		t.Error("RawData should preserve all fields")
	}

	if rawMap["antallAnsatte"] != float64(8) {
		t.Error("RawData should preserve numeric fields")
	}
}
