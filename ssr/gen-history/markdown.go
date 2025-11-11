package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// GenerateTimelineMarkdown creates markdown content for an entity's history
func GenerateTimelineMarkdown(orgnum string, changes []EntityChange) (string, error) {
	if len(changes) == 0 {
		return "", fmt.Errorf("no changes to generate markdown for")
	}

	// Get entity name from most recent enhet change
	navn := extractNavn(changes[0].Data)
	if navn == "" {
		// Try to find name from other changes
		for _, change := range changes {
			if strings.Contains(change.FilePath, "enhet.json") || strings.Contains(change.FilePath, "underenhet.json") {
				navn = extractNavn(change.Data)
				if navn != "" {
					break
				}
			}
		}
		if navn == "" {
			navn = "Ukjent enhet"
		}
	}

	var sb strings.Builder

	// Frontmatter
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("title: \"%s\"\n", escapeYAMLString(navn)))
	sb.WriteString(fmt.Sprintf("organisasjonsnummer: \"%s\"\n", orgnum))
	sb.WriteString(fmt.Sprintf("lastModified: \"%s\"\n", changes[0].Date.Format("2006-01-02")))
	sb.WriteString("---\n\n")

	// Title
	sb.WriteString(fmt.Sprintf("# %s (%s)\n\n", navn, orgnum))

	// Timeline
	sb.WriteString("## Tidslinje\n\n")

	// Process changes from newest to oldest
	for i, change := range changes {
		// Determine change type
		changeType := determineChangeType(change)

		// Write change header
		sb.WriteString(fmt.Sprintf("### %s\n", change.Date.Format("2006-01-02 15:04 MST")))
		sb.WriteString(fmt.Sprintf("**%s**", changeType))

		// Determine file type for labeling
		var fileLabel string
		if strings.Contains(change.FilePath, "roller.json") {
			fileLabel = "Rolleendringer"
		} else if strings.Contains(change.FilePath, "enhet.json") {
			fileLabel = "Enhetsendringer"
		} else if strings.Contains(change.FilePath, "underenhet.json") {
			fileLabel = "Underenhetsendringer"
		}

		// Compare with previous change if not the first (newest) change
		if i < len(changes)-1 {
			// Find previous change of the same file type
			var prevChange *EntityChange
			for j := i + 1; j < len(changes); j++ {
				if changes[j].FilePath == change.FilePath {
					prevChange = &changes[j]
					break
				}
			}

			if prevChange == nil {
				// No previous change of same type, skip diff
				sb.WriteString("\n\n")
				sb.WriteString("Opprettet i Enhetsregisteret\n\n")

				// Show initial data with git diff
				sb.WriteString("<details>\n")
				sb.WriteString("<summary>Vis git diff</summary>\n\n")

				diff, err := GetCommitDiff(".", change.CommitHash, change.FilePath)
				if err == nil && diff != "" {
					sb.WriteString("```diff\n")
					sb.WriteString(diff)
					sb.WriteString("\n```\n\n")
				} else {
					sb.WriteString("```json\n")
					prettyJSON, _ := json.MarshalIndent(json.RawMessage(change.Data), "", "  ")
					sb.WriteString(string(prettyJSON))
					sb.WriteString("\n```\n\n")
				}

				sb.WriteString("</details>\n\n")
				sb.WriteString("---\n\n")
				continue
			}

			// Calculate diff
			diffs, err := DiffJSON(prevChange.Data, change.Data)
			if err == nil && len(diffs) > 0 {
				sb.WriteString(fmt.Sprintf(" • %d felt endret", len(diffs)))
				sb.WriteString("\n\n")

				// Add file type label if present
				if fileLabel != "" {
					sb.WriteString(fmt.Sprintf("**%s**\n\n", fileLabel))
				}

				// Write changes as human-readable list
				for _, diff := range diffs {
					// Try to get human-readable translation
					humanText := TranslateChange(diff.Field, diff.OldValue, diff.NewValue)

					if humanText != "" {
						sb.WriteString(fmt.Sprintf("- %s\n", humanText))
					} else {
						// Fallback: just show translated field name without values
						fieldName := translateFieldName(diff.Field)
						sb.WriteString(fmt.Sprintf("- %s endret\n", fieldName))
					}
				}

				sb.WriteString("\n")

				// Add expandable section with git diff
				sb.WriteString("<details>\n")
				sb.WriteString("<summary>Vis git diff</summary>\n\n")

				// Get git diff for this change
				diff, err := GetCommitDiff(".", change.CommitHash, change.FilePath)
				if err == nil && diff != "" {
					sb.WriteString("```diff\n")
					sb.WriteString(diff)
					sb.WriteString("\n```\n\n")
				} else {
					// Fallback to full JSON if diff fails
					sb.WriteString("```json\n")
					prettyJSON, _ := json.MarshalIndent(json.RawMessage(change.Data), "", "  ")
					sb.WriteString(string(prettyJSON))
					sb.WriteString("\n```\n\n")
				}

				sb.WriteString("</details>\n\n")
			} else {
				sb.WriteString("\n\n")
			}
		} else {
			// First commit (creation)
			sb.WriteString("\n\n")
			sb.WriteString("Opprettet i Enhetsregisteret\n\n")

			// Show initial data with git diff
			sb.WriteString("<details>\n")
			sb.WriteString("<summary>Vis git diff</summary>\n\n")

			// Get git diff for this change
			diff, err := GetCommitDiff(".", change.CommitHash, change.FilePath)
			if err == nil && diff != "" {
				sb.WriteString("```diff\n")
				sb.WriteString(diff)
				sb.WriteString("\n```\n\n")
			} else {
				// Fallback to full JSON if diff fails
				sb.WriteString("```json\n")
				prettyJSON, _ := json.MarshalIndent(json.RawMessage(change.Data), "", "  ")
				sb.WriteString(string(prettyJSON))
				sb.WriteString("\n```\n\n")
			}

			sb.WriteString("</details>\n\n")
		}

		sb.WriteString("---\n\n")
	}

	return sb.String(), nil
}

// extractNavn gets the entity name from JSON data
func extractNavn(data json.RawMessage) string {
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return ""
	}

	if navn, ok := obj["navn"].(string); ok {
		return navn
	}

	return ""
}

// determineChangeType categorizes the type of change
func determineChangeType(change EntityChange) string {
	// Check if entity was deleted
	var obj map[string]interface{}
	if err := json.Unmarshal(change.Data, &obj); err == nil {
		if responsKlasse, ok := obj["respons_klasse"].(string); ok {
			if responsKlasse == "SlettetEnhet" {
				return "Slettet"
			}
		}
	}

	// Check commit message for hints
	msg := strings.ToLower(change.Message)
	if strings.Contains(msg, "initial") || strings.Contains(msg, "opprett") {
		return "Opprettet"
	}

	return "Endret"
}

// FormatTimestamp formats a timestamp for display
func FormatTimestamp(t time.Time) string {
	return t.Format("2006-01-02 15:04 MST")
}

// isComplexValue checks if a value is a complex type (object or array)
func isComplexValue(val interface{}) bool {
	if val == nil {
		return false
	}
	switch val.(type) {
	case map[string]interface{}, []interface{}:
		return true
	default:
		return false
	}
}

// escapeYAMLString escapes special characters in YAML strings
func escapeYAMLString(s string) string {
	// Replace double quotes with escaped quotes
	s = strings.ReplaceAll(s, "\"", "\\\"")
	// Replace backslashes
	s = strings.ReplaceAll(s, "\\", "\\\\")
	return s
}
