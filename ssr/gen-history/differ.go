package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
)

// FieldDiff represents a change to a single field
type FieldDiff struct {
	Field    string
	OldValue interface{}
	NewValue interface{}
}

// DiffJSON compares two JSON objects and returns field-level differences
func DiffJSON(oldData, newData json.RawMessage) ([]FieldDiff, error) {
	var oldMap, newMap map[string]interface{}

	if err := json.Unmarshal(oldData, &oldMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal old data: %w", err)
	}

	if err := json.Unmarshal(newData, &newMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal new data: %w", err)
	}

	return compareObjects("", oldMap, newMap), nil
}

// compareObjects recursively compares two objects and returns differences
func compareObjects(prefix string, old, new map[string]interface{}) []FieldDiff {
	var diffs []FieldDiff

	// Check all fields in new object
	for key, newVal := range new {
		fieldPath := key
		if prefix != "" {
			fieldPath = prefix + "." + key
		}

		oldVal, existedBefore := old[key]

		// Field added
		if !existedBefore {
			diffs = append(diffs, FieldDiff{
				Field:    fieldPath,
				OldValue: nil,
				NewValue: newVal,
			})
			continue
		}

		// Compare values
		if !reflect.DeepEqual(oldVal, newVal) {
			// If both are objects, recurse
			oldMap, oldIsMap := oldVal.(map[string]interface{})
			newMap, newIsMap := newVal.(map[string]interface{})

			if oldIsMap && newIsMap {
				nestedDiffs := compareObjects(fieldPath, oldMap, newMap)
				diffs = append(diffs, nestedDiffs...)
			} else {
				// Simple value change
				diffs = append(diffs, FieldDiff{
					Field:    fieldPath,
					OldValue: oldVal,
					NewValue: newVal,
				})
			}
		}
	}

	// Check for removed fields
	for key, oldVal := range old {
		fieldPath := key
		if prefix != "" {
			fieldPath = prefix + "." + key
		}

		if _, existsInNew := new[key]; !existsInNew {
			diffs = append(diffs, FieldDiff{
				Field:    fieldPath,
				OldValue: oldVal,
				NewValue: nil,
			})
		}
	}

	// Sort diffs by field name for consistent output
	sort.Slice(diffs, func(i, j int) bool {
		return diffs[i].Field < diffs[j].Field
	})

	return diffs
}

// FormatValue formats a value for display in markdown
func FormatValue(val interface{}) string {
	if val == nil {
		return "_null_"
	}

	switch v := val.(type) {
	case string:
		return fmt.Sprintf("\"%s\"", v)
	case float64:
		// Check if it's actually an integer
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%.2f", v)
	case bool:
		return fmt.Sprintf("%t", v)
	case map[string]interface{}:
		// Format nested objects
		b, _ := json.MarshalIndent(v, "", "  ")
		return string(b)
	case []interface{}:
		// Format arrays
		if len(v) == 0 {
			return "[]"
		}
		b, _ := json.MarshalIndent(v, "", "  ")
		return string(b)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// GetFieldValue extracts a specific field value from JSON data
func GetFieldValue(data json.RawMessage, fieldPath string) (interface{}, error) {
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}

	// Handle nested field paths (e.g., "postadresse.postnummer")
	parts := splitFieldPath(fieldPath)
	current := interface{}(obj)

	for _, part := range parts {
		currentMap, ok := current.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("field %s not found", fieldPath)
		}

		val, exists := currentMap[part]
		if !exists {
			return nil, fmt.Errorf("field %s not found", fieldPath)
		}

		current = val
	}

	return current, nil
}

// splitFieldPath splits a field path by dots, handling edge cases
func splitFieldPath(path string) []string {
	if path == "" {
		return []string{}
	}

	var parts []string
	current := ""

	for _, ch := range path {
		if ch == '.' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}

	if current != "" {
		parts = append(parts, current)
	}

	return parts
}
