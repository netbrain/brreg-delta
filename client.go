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

// EntityDeletedError indicates an entity was deleted (410 Gone)
type EntityDeletedError struct {
	Orgnum string
}

func (e *EntityDeletedError) Error() string {
	return fmt.Sprintf("entity deleted: %s", e.Orgnum)
}

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

// Enhet represents an entity (company)
type Enhet struct {
	Organisasjonsnummer string          `json:"organisasjonsnummer"`
	RawData             json.RawMessage `json:"-"`
}

func (e *Enhet) UnmarshalJSON(data []byte) error {
	type Alias Enhet
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(e),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	e.RawData = data
	return nil
}

// Underenhet represents a sub-unit
type Underenhet struct {
	Organisasjonsnummer string          `json:"organisasjonsnummer"`
	OverordnetEnhet     string          `json:"overordnetEnhet"`
	RawData             json.RawMessage `json:"-"`
}

func (u *Underenhet) UnmarshalJSON(data []byte) error {
	type Alias Underenhet
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(u),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	u.RawData = data
	return nil
}

// OppdaterteEnheter represents paginated updates response for enheter
type OppdaterteEnheter struct {
	Embedded struct {
		OppdaterteEnheter []OppdateringEnhet `json:"oppdaterteEnheter"`
	} `json:"_embedded"`
	Page struct {
		Size          int `json:"size"`
		TotalElements int `json:"totalElements"`
		TotalPages    int `json:"totalPages"`
		Number        int `json:"number"`
	} `json:"page"`
}

// OppdateringEnhet represents a single entity update
type OppdateringEnhet struct {
	Oppdateringsid      int64  `json:"oppdateringsid"`
	Dato                string `json:"dato"`
	Organisasjonsnummer string `json:"organisasjonsnummer"`
	Endringstype        string `json:"endringstype"`
}

// OppdaterteUnderenheter represents paginated updates response for underenheter
type OppdaterteUnderenheter struct {
	Embedded struct {
		OppdaterteUnderenheter []OppdateringUnderenhet `json:"oppdaterteUnderenheter"`
	} `json:"_embedded"`
	Page struct {
		Size          int `json:"size"`
		TotalElements int `json:"totalElements"`
		TotalPages    int `json:"totalPages"`
		Number        int `json:"number"`
	} `json:"page"`
}

// OppdateringUnderenhet represents a single sub-unit update
type OppdateringUnderenhet struct {
	Oppdateringsid      int64  `json:"oppdateringsid"`
	Dato                string `json:"dato"`
	Organisasjonsnummer string `json:"organisasjonsnummer"`
	Endringstype        string `json:"endringstype"`
}

// Backwards compatibility
type Company = Enhet

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

// FetchRoles fetches roles/representatives for a specific company (backwards compatibility)
func (c *Client) FetchRoles(orgnum string) (json.RawMessage, error) {
	return c.FetchEnhetRoller(orgnum)
}

// FetchEnhet fetches a single enhet by organization number
func (c *Client) FetchEnhet(orgnum string) (json.RawMessage, error) {
	url := fmt.Sprintf("%s/enheter/%s", baseURL, orgnum)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch enhet %s: %w", orgnum, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusGone {
		// Entity deleted (410 Gone)
		return nil, &EntityDeletedError{Orgnum: orgnum}
	}

	if resp.StatusCode == http.StatusNotFound {
		// Entity not found (404)
		return nil, fmt.Errorf("enhet not found: %s", orgnum)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code for enhet %s: %d", orgnum, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return body, nil
}

// FetchUnderenhet fetches a single underenhet by organization number
func (c *Client) FetchUnderenhet(orgnum string) (json.RawMessage, error) {
	url := fmt.Sprintf("%s/underenheter/%s", baseURL, orgnum)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch underenhet %s: %w", orgnum, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusGone {
		// Entity deleted (410 Gone)
		return nil, &EntityDeletedError{Orgnum: orgnum}
	}

	if resp.StatusCode == http.StatusNotFound {
		// Entity not found (404)
		return nil, fmt.Errorf("underenhet not found: %s", orgnum)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code for underenhet %s: %d", orgnum, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return body, nil
}

// FetchEnhetRoller fetches roles for an enhet
func (c *Client) FetchEnhetRoller(orgnum string) (json.RawMessage, error) {
	url := fmt.Sprintf("%s/enheter/%s/roller", baseURL, orgnum)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch roles for enhet %s: %w", orgnum, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// No roles available, return empty JSON
		return json.RawMessage("{}"), nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code for enhet roles %s: %d", orgnum, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return body, nil
}

// FetchUnderenhetRoller fetches roles for an underenhet
func (c *Client) FetchUnderenhetRoller(orgnum string) (json.RawMessage, error) {
	url := fmt.Sprintf("%s/underenheter/%s/roller", baseURL, orgnum)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch roles for underenhet %s: %w", orgnum, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// No roles available, return empty JSON
		return json.RawMessage("{}"), nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code for underenhet roles %s: %d", orgnum, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return body, nil
}

// FetchEnheterUpdates fetches enheter updates from a specific oppdateringsid
func (c *Client) FetchEnheterUpdates(fraOppdateringsid int64, size int) (*OppdaterteEnheter, error) {
	url := fmt.Sprintf("%s/oppdateringer/enheter?oppdateringsid=%d&size=%d", baseURL, fraOppdateringsid, size)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch enheter updates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code for enheter updates: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var updates OppdaterteEnheter
	if err := json.Unmarshal(body, &updates); err != nil {
		return nil, fmt.Errorf("failed to unmarshal enheter updates: %w", err)
	}

	return &updates, nil
}

// FetchUnderenheterUpdates fetches underenheter updates from a specific oppdateringsid
func (c *Client) FetchUnderenheterUpdates(fraOppdateringsid int64, size int) (*OppdaterteUnderenheter, error) {
	url := fmt.Sprintf("%s/oppdateringer/underenheter?oppdateringsid=%d&size=%d", baseURL, fraOppdateringsid, size)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch underenheter updates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code for underenheter updates: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var updates OppdaterteUnderenheter
	if err := json.Unmarshal(body, &updates); err != nil {
		return nil, fmt.Errorf("failed to unmarshal underenheter updates: %w", err)
	}

	return &updates, nil
}

// GetLatestEnheterOppdateringsid fetches the most recent oppdateringsid for enheter
// Uses a high starting point and fetches a batch to find the latest
func (c *Client) GetLatestEnheterOppdateringsid() (int64, error) {
	// Start from a high oppdateringsid (14M as of 2025)
	startId := int64(13900000)
	batchSize := 1000

	for {
		updates, err := c.FetchEnheterUpdates(startId, batchSize)
		if err != nil {
			return 0, fmt.Errorf("failed to fetch updates: %w", err)
		}

		if len(updates.Embedded.OppdaterteEnheter) == 0 {
			// No more updates, the last batch had the max
			return startId - 1, nil
		}

		// Get the last oppdateringsid from this batch
		lastId := updates.Embedded.OppdaterteEnheter[len(updates.Embedded.OppdaterteEnheter)-1].Oppdateringsid

		// If we got fewer than requested, this is the last batch
		if len(updates.Embedded.OppdaterteEnheter) < batchSize {
			return lastId, nil
		}

		// Move to next batch
		startId = lastId + 1
	}
}

// GetLatestUnderenheterOppdateringsid fetches the most recent oppdateringsid for underenheter
func (c *Client) GetLatestUnderenheterOppdateringsid() (int64, error) {
	// Start from a high oppdateringsid (4M as of 2025)
	startId := int64(3900000)
	batchSize := 1000

	for {
		updates, err := c.FetchUnderenheterUpdates(startId, batchSize)
		if err != nil {
			return 0, fmt.Errorf("failed to fetch updates: %w", err)
		}

		if len(updates.Embedded.OppdaterteUnderenheter) == 0 {
			// No more updates, the last batch had the max
			return startId - 1, nil
		}

		// Get the last oppdateringsid from this batch
		lastId := updates.Embedded.OppdaterteUnderenheter[len(updates.Embedded.OppdaterteUnderenheter)-1].Oppdateringsid

		// If we got fewer than requested, this is the last batch
		if len(updates.Embedded.OppdaterteUnderenheter) < batchSize {
			return lastId, nil
		}

		// Move to next batch
		startId = lastId + 1
	}
}

// LoadEnheterFromFile loads enheter from bulk download file (handles gzip)
func (c *Client) LoadEnheterFromFile(path string) ([]Enhet, error) {
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

	var enheter []Enhet
	if err := json.Unmarshal(body, &enheter); err != nil {
		return nil, fmt.Errorf("failed to parse enheter list: %w", err)
	}

	return enheter, nil
}

// LoadUnderenheterFromFile loads underenheter from bulk download file (handles gzip)
func (c *Client) LoadUnderenheterFromFile(path string) ([]Underenhet, error) {
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

	var underenheter []Underenhet
	if err := json.Unmarshal(body, &underenheter); err != nil {
		return nil, fmt.Errorf("failed to parse underenheter list: %w", err)
	}

	return underenheter, nil
}

// RollerEntry represents a single entry in the roller bulk file
type RollerEntry struct {
	Organisasjonsnummer string          `json:"organisasjonsnummer"`
	RawData             json.RawMessage `json:"-"`
}

func (r *RollerEntry) UnmarshalJSON(data []byte) error {
	type Alias RollerEntry
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(r),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	r.RawData = data
	return nil
}

// LoadRollerFromFile loads roller from bulk download file (handles gzip)
func (c *Client) LoadRollerFromFile(path string) ([]RollerEntry, error) {
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

	var roller []RollerEntry
	if err := json.Unmarshal(body, &roller); err != nil {
		return nil, fmt.Errorf("failed to parse roller list: %w", err)
	}

	return roller, nil
}
