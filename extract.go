package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// ExtractCompanies extracts each company from bulk list into individual files
func ExtractCompanies(listFile, dataDir string) (int, error) {
	log.Printf("Loading company list from %s", listFile)
	client := NewClient()
	companies, err := client.LoadCompanyListFromFile(listFile)
	if err != nil {
		return 0, fmt.Errorf("failed to load company list: %w", err)
	}

	log.Printf("Extracting %d companies to %s", len(companies), dataDir)

	storage := NewStorage(dataDir)
	extracted := 0

	for i, company := range companies {
		// Save company data
		_, err := storage.SaveCompany(company.Organisasjonsnummer, company.RawData)
		if err != nil {
			log.Printf("Warning: failed to save company %s: %v", company.Organisasjonsnummer, err)
			continue
		}
		extracted++

		if (i+1)%10000 == 0 {
			log.Printf("Extracted %d/%d companies", i+1, len(companies))
		}
	}

	log.Printf("Extraction complete: %d companies", extracted)
	return extracted, nil
}

// CountExtractedCompanies counts directories in data folder
func CountExtractedCompanies(dataDir string) (int, error) {
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to read data directory: %w", err)
	}

	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			// Verify it has a company.json file
			companyFile := filepath.Join(dataDir, entry.Name(), "company.json")
			if _, err := os.Stat(companyFile); err == nil {
				count++
			}
		}
	}

	return count, nil
}

// GetCompanyOrgNums returns list of organization numbers from extracted data
func GetCompanyOrgNums(dataDir string) ([]string, error) {
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read data directory: %w", err)
	}

	orgnums := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			// Verify it has a company.json file
			companyFile := filepath.Join(dataDir, entry.Name(), "company.json")
			if _, err := os.Stat(companyFile); err == nil {
				orgnums = append(orgnums, entry.Name())
			}
		}
	}

	return orgnums, nil
}

// ReadCompanyData reads company data from disk
func ReadCompanyData(dataDir, orgnum string) (json.RawMessage, error) {
	path := filepath.Join(dataDir, orgnum, "company.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read company file: %w", err)
	}
	return json.RawMessage(data), nil
}
