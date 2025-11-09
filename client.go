package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	baseURL = "https://data.brreg.no/enhetsregisteret/api"
)

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Company represents a company entity
type Company struct {
	Organisasjonsnummer string          `json:"organisasjonsnummer"`
	RawData             json.RawMessage `json:"-"`
}

func (c *Company) UnmarshalJSON(data []byte) error {
	type Alias Company
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(c),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	c.RawData = data
	return nil
}

// Roles represents roles/representatives for a company
type Roles struct {
	RawData json.RawMessage
}

// LoadCompanyListFromFile loads company list from a JSON file (handles gzip)
func (c *Client) LoadCompanyListFromFile(path string) ([]Company, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var reader io.Reader = file

	// Try to detect gzip by reading magic bytes
	magic := make([]byte, 2)
	if _, err := file.Read(magic); err == nil {
		file.Seek(0, 0) // Reset file position
		if magic[0] == 0x1f && magic[1] == 0x8b {
			gzReader, err := gzip.NewReader(file)
			if err != nil {
				return nil, fmt.Errorf("failed to create gzip reader: %w", err)
			}
			defer gzReader.Close()
			reader = gzReader
		}
	} else {
		file.Seek(0, 0) // Reset on error
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var companies []Company
	if err := json.Unmarshal(body, &companies); err != nil {
		return nil, fmt.Errorf("failed to parse company list: %w", err)
	}

	return companies, nil
}

// FetchRoles fetches roles/representatives for a specific company
func (c *Client) FetchRoles(orgnum string) (json.RawMessage, error) {
	url := fmt.Sprintf("%s/enheter/%s/roller", baseURL, orgnum)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch roles for %s: %w", orgnum, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// No roles available, return empty JSON
		return json.RawMessage("{}"), nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code for roles %s: %d", orgnum, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return body, nil
}
