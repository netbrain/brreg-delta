package main

import (
	"testing"
)

func TestBatchDistribution(t *testing.T) {
	tests := []struct {
		name          string
		totalCompanies int
		totalBatches  int
		batchNum      int
		expectedStart int
		expectedEnd   int
	}{
		{
			name:          "Even distribution",
			totalCompanies: 1000,
			totalBatches:  10,
			batchNum:      0,
			expectedStart: 0,
			expectedEnd:   100,
		},
		{
			name:          "Even distribution - middle batch",
			totalCompanies: 1000,
			totalBatches:  10,
			batchNum:      5,
			expectedStart: 500,
			expectedEnd:   600,
		},
		{
			name:          "Even distribution - last batch",
			totalCompanies: 1000,
			totalBatches:  10,
			batchNum:      9,
			expectedStart: 900,
			expectedEnd:   1000,
		},
		{
			name:          "Uneven distribution - last batch gets remainder",
			totalCompanies: 1000,
			totalBatches:  3,
			batchNum:      2,
			expectedStart: 668,
			expectedEnd:   1000,
		},
		{
			name:          "256 batches - first",
			totalCompanies: 2000000,
			totalBatches:  256,
			batchNum:      0,
			expectedStart: 0,
			expectedEnd:   7813,
		},
		{
			name:          "256 batches - last",
			totalCompanies: 2000000,
			totalBatches:  256,
			batchNum:      255,
			expectedStart: 1992315,
			expectedEnd:   2000000,
		},
		{
			name:          "Small dataset",
			totalCompanies: 10,
			totalBatches:  256,
			batchNum:      0,
			expectedStart: 0,
			expectedEnd:   1,
		},
		{
			name:          "Small dataset - empty batch",
			totalCompanies: 10,
			totalBatches:  256,
			batchNum:      15,
			expectedStart: 15,
			expectedEnd:   10, // Capped at totalCompanies
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the batch calculation from processor.go
			batchSize := (tt.totalCompanies + tt.totalBatches - 1) / tt.totalBatches
			start := tt.batchNum * batchSize
			end := start + batchSize
			if end > tt.totalCompanies {
				end = tt.totalCompanies
			}

			if start != tt.expectedStart {
				t.Errorf("Expected start %d, got %d", tt.expectedStart, start)
			}
			if end != tt.expectedEnd {
				t.Errorf("Expected end %d, got %d", tt.expectedEnd, end)
			}
		})
	}
}

func TestBatchCoverage(t *testing.T) {
	// Ensure all companies are covered and no duplicates
	totalCompanies := 2000000
	totalBatches := 256

	coverage := make(map[int]bool)

	for batchNum := 0; batchNum < totalBatches; batchNum++ {
		batchSize := (totalCompanies + totalBatches - 1) / totalBatches
		start := batchNum * batchSize
		end := start + batchSize
		if end > totalCompanies {
			end = totalCompanies
		}

		for i := start; i < end; i++ {
			if coverage[i] {
				t.Errorf("Company %d covered multiple times", i)
			}
			coverage[i] = true
		}
	}

	// Check all companies are covered
	for i := 0; i < totalCompanies; i++ {
		if !coverage[i] {
			t.Errorf("Company %d not covered by any batch", i)
		}
	}
}

func TestStatsTracking(t *testing.T) {
	stats := &Stats{}

	if stats.processed != 0 {
		t.Error("Initial processed count should be 0")
	}

	// Simulate some stats
	stats.processed = 100
	stats.companyChanged = 10
	stats.rolesChanged = 5
	stats.errors = 2

	expected := "Processed: 100, Company changes: 10, Roles changes: 5, Errors: 2"
	if stats.String() != expected {
		t.Errorf("Expected stats string: %s, got: %s", expected, stats.String())
	}
}
