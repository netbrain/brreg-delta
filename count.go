package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// CountOrganizations counts organization numbers in a company list file
func CountOrganizations(listFile string) (int, error) {
	client := NewClient()
	companies, err := client.LoadCompanyListFromFile(listFile)
	if err != nil {
		return 0, fmt.Errorf("failed to load company list: %w", err)
	}
	return len(companies), nil
}

// ExtractOrganizationNumbers extracts all organization numbers to a separate file
func ExtractOrganizationNumbers(listFile, outputFile string) (int, error) {
	client := NewClient()
	companies, err := client.LoadCompanyListFromFile(listFile)
	if err != nil {
		return 0, fmt.Errorf("failed to load company list: %w", err)
	}

	orgnums := make([]string, len(companies))
	for i, company := range companies {
		orgnums[i] = company.Organisasjonsnummer
	}

	// Write organization numbers as JSON array
	data, err := json.MarshalIndent(map[string]interface{}{
		"total": len(orgnums),
		"orgnums": orgnums,
	}, "", "  ")
	if err != nil {
		return 0, fmt.Errorf("failed to marshal orgnums: %w", err)
	}

	if err := os.WriteFile(outputFile, data, 0644); err != nil {
		return 0, fmt.Errorf("failed to write orgnums file: %w", err)
	}

	return len(orgnums), nil
}
