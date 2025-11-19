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
	sb.WriteString(fmt.Sprintf("lastModified: \"%s\"\n", changes[0].Date.Format("02.01.2006")))
	sb.WriteString("---\n\n")

	// Title
	sb.WriteString(fmt.Sprintf("# %s (%s)\n\n", navn, orgnum))

	// Group changes by commit hash to avoid duplicate timestamps
	commitGroups := groupChangesByCommit(changes)

	// Process commit groups from newest to oldest
	for _, group := range commitGroups {
		// Start accordion (details/summary) with timestamp
		sb.WriteString("<details open>\n")
		sb.WriteString(fmt.Sprintf("<summary><strong>%s</strong></summary>\n\n", group.Date.Format("02.01.2006")))

		// Build content for each file type separately
		fileTypeContent := make(map[string]string)

		for _, change := range group.Changes {
			// Determine file type
			var fileType string
			if strings.Contains(change.FilePath, "enhet.json") {
				fileType = "enhet"
			} else if strings.Contains(change.FilePath, "underenhet.json") {
				fileType = "underenhet"
			} else if strings.Contains(change.FilePath, "roller.json") {
				fileType = "roller"
			}

			if fileType == "" {
				continue
			}

			// Build content for this file type
			var contentSb strings.Builder

			// Find change index in original changes array
			changeIdx := findChangeIndex(changes, change.CommitHash, change.FilePath)

			// Find previous change of the same file type
			var prevChange *EntityChange
			for j := changeIdx + 1; j < len(changes); j++ {
				if changes[j].FilePath == change.FilePath {
					prevChange = &changes[j]
					break
				}
			}

			if prevChange == nil {
				// No previous change of same type - this is creation
				if fileType == "enhet" {
					contentSb.WriteString(summarizeEnhetCreation(change.Data))
				} else if fileType == "underenhet" {
					contentSb.WriteString(summarizeUnderenhetCreation(change.Data))
				} else if fileType == "roller" {
					contentSb.WriteString(summarizeRollerCreation(change.Data))
				}
			} else {
				// Calculate diff
				diffs, err := DiffJSON(prevChange.Data, change.Data)
				if err == nil && len(diffs) > 0 {
					// Filter out _links and links fields (API metadata, not business data)
					var filteredDiffs []FieldDiff
					for _, diff := range diffs {
						if !isLinkField(diff.Field) {
							filteredDiffs = append(filteredDiffs, diff)
						}
					}

					if len(filteredDiffs) > 0 {
						contentSb.WriteString(fmt.Sprintf("%d felt endret\n\n", len(filteredDiffs)))

						// Write changes as human-readable list
						for _, diff := range filteredDiffs {
							humanText := TranslateChange(diff.Field, diff.OldValue, diff.NewValue)
							if humanText != "" {
								contentSb.WriteString(fmt.Sprintf("- %s\n", humanText))
							} else {
								fieldName := translateFieldName(diff.Field)
								contentSb.WriteString(fmt.Sprintf("- %s endret\n", fieldName))
							}
						}
					}

					contentSb.WriteString("\n")

					// Add expandable section with git diff in footer
					contentSb.WriteString("<footer>\n")
					contentSb.WriteString("<details>\n")
					contentSb.WriteString("<summary>Vis git diff</summary>\n\n")

					diff, err := GetCommitDiff(".", change.CommitHash, change.FilePath)
					if err == nil && diff != "" {
						contentSb.WriteString("```diff\n")
						contentSb.WriteString(diff)
						contentSb.WriteString("\n```\n\n")
					} else {
						contentSb.WriteString("```json\n")
						prettyJSON, _ := json.MarshalIndent(json.RawMessage(change.Data), "", "  ")
						contentSb.WriteString(string(prettyJSON))
						contentSb.WriteString("\n```\n\n")
					}

					contentSb.WriteString("</details>\n")
					contentSb.WriteString("</footer>\n\n")
				}
			}

			fileTypeContent[fileType] = contentSb.String()
		}

		// Determine grid layout based on how many file types we have
		fileTypes := []string{}
		if _, ok := fileTypeContent["enhet"]; ok {
			fileTypes = append(fileTypes, "enhet")
		}
		if _, ok := fileTypeContent["underenhet"]; ok {
			fileTypes = append(fileTypes, "underenhet")
		}
		if _, ok := fileTypeContent["roller"]; ok {
			fileTypes = append(fileTypes, "roller")
		}

		if len(fileTypes) > 1 {
			// Multiple file types - use grid with articles
			sb.WriteString("<div style=\"display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 1rem;\">\n")

			for _, ft := range fileTypes {
				sb.WriteString("<article style=\"display: flex; flex-direction: column;\">\n")
				sb.WriteString("<div style=\"flex: 1;\">\n")
				sb.WriteString(fmt.Sprintf("<h4>%s</h4>\n\n", getFileTypeLabel(ft)))
				sb.WriteString(fileTypeContent[ft])
				sb.WriteString("</div>\n")
				sb.WriteString("</article>\n")
			}

			sb.WriteString("</div>\n")
		} else if len(fileTypes) == 1 {
			// Single file type - wrap in article
			ft := fileTypes[0]
			sb.WriteString("<article style=\"display: flex; flex-direction: column;\">\n")
			sb.WriteString("<div style=\"flex: 1;\">\n")
			sb.WriteString(fmt.Sprintf("<h4>%s</h4>\n\n", getFileTypeLabel(ft)))
			sb.WriteString(fileTypeContent[ft])
			sb.WriteString("</div>\n")
			sb.WriteString("</article>\n")
		}

		sb.WriteString("</details>\n\n")
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
	// Escape backslashes FIRST (order matters!)
	s = strings.ReplaceAll(s, "\\", "\\\\")
	// Then escape double quotes
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}

// CommitGroup groups changes by commit hash
type CommitGroup struct {
	CommitHash string
	Date       time.Time
	Message    string
	Changes    []EntityChange
}

// groupChangesByCommit groups entity changes by date (not commit hash)
func groupChangesByCommit(changes []EntityChange) []CommitGroup {
	groupMap := make(map[string]*CommitGroup)
	var order []string

	for _, change := range changes {
		// Group by date only (YYYY-MM-DD)
		dateKey := change.Date.Format("2006-01-02")

		if _, exists := groupMap[dateKey]; !exists {
			groupMap[dateKey] = &CommitGroup{
				CommitHash: change.CommitHash, // Use first commit hash
				Date:       change.Date,
				Message:    change.Message,
				Changes:    []EntityChange{},
			}
			order = append(order, dateKey)
		}
		groupMap[dateKey].Changes = append(groupMap[dateKey].Changes, change)
	}

	// Maintain original order (newest first)
	result := make([]CommitGroup, 0, len(order))
	for _, dateKey := range order {
		result = append(result, *groupMap[dateKey])
	}

	return result
}

// findChangeIndex finds the index of a change in the changes array
func findChangeIndex(changes []EntityChange, commitHash, filePath string) int {
	for i, change := range changes {
		if change.CommitHash == commitHash && change.FilePath == filePath {
			return i
		}
	}
	return -1
}

// getFileTypeLabel returns the display label for a file type
func getFileTypeLabel(fileType string) string {
	switch fileType {
	case "enhet":
		return "Enhetsendringer"
	case "underenhet":
		return "Underenhetsendringer"
	case "roller":
		return "Rolleendringer"
	default:
		return "Endringer"
	}
}

// isLinkField checks if a field is a links/_links metadata field
func isLinkField(field string) bool {
	// Filter out _links and links fields at any level
	return strings.Contains(field, "._links") ||
	       strings.Contains(field, ".links") ||
	       field == "_links" ||
	       field == "links"
}

// summarizeEnhetCreation extracts key information from enhet creation
func summarizeEnhetCreation(data json.RawMessage) string {
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return "Registrert i Enhetsregisteret\n\n"
	}

	var sb strings.Builder
	sb.WriteString("Registrert i Enhetsregisteret\n\n")

	// Extract key fields
	if navn, ok := obj["navn"].(string); ok && navn != "" {
		sb.WriteString(fmt.Sprintf("- **Navn:** %s\n", navn))
	}

	if orgform, ok := obj["organisasjonsform"].(map[string]interface{}); ok {
		if beskrivelse, ok := orgform["beskrivelse"].(string); ok {
			kode, _ := orgform["kode"].(string)
			if kode != "" {
				sb.WriteString(fmt.Sprintf("- **Organisasjonsform:** %s (%s)\n", beskrivelse, kode))
			} else {
				sb.WriteString(fmt.Sprintf("- **Organisasjonsform:** %s\n", beskrivelse))
			}
		}
	}

	if naring1, ok := obj["naeringskode1"].(map[string]interface{}); ok {
		if beskrivelse, ok := naring1["beskrivelse"].(string); ok {
			kode, _ := naring1["kode"].(string)
			sb.WriteString(fmt.Sprintf("- **Næring:** %s (%s)\n", beskrivelse, kode))
		}
	}

	if forretningsadr, ok := obj["forretningsadresse"].(map[string]interface{}); ok {
		var addrParts []string
		if adresser, ok := forretningsadr["adresse"].([]interface{}); ok && len(adresser) > 0 {
			if addr, ok := adresser[0].(string); ok {
				addrParts = append(addrParts, addr)
			}
		}
		postnr, _ := forretningsadr["postnummer"].(string)
		poststed, _ := forretningsadr["poststed"].(string)
		if postnr != "" && poststed != "" {
			addrParts = append(addrParts, fmt.Sprintf("%s %s", postnr, poststed))
		}
		if len(addrParts) > 0 {
			sb.WriteString(fmt.Sprintf("- **Adresse:** %s\n", strings.Join(addrParts, ", ")))
		}
	}

	if stiftelsesdato, ok := obj["stiftelsesdato"].(string); ok && stiftelsesdato != "" {
		sb.WriteString(fmt.Sprintf("- **Stiftelsesdato:** %s\n", stiftelsesdato))
	}

	sb.WriteString("\n")
	return sb.String()
}

// summarizeUnderenhetCreation extracts key information from underenhet creation
func summarizeUnderenhetCreation(data json.RawMessage) string {
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return "Underenhet opprettet\n\n"
	}

	var sb strings.Builder
	sb.WriteString("Underenhet opprettet\n\n")

	if navn, ok := obj["navn"].(string); ok && navn != "" {
		sb.WriteString(fmt.Sprintf("- **Navn:** %s\n", navn))
	}

	if beliggenhetsadr, ok := obj["beliggenhetsadresse"].(map[string]interface{}); ok {
		var addrParts []string
		if adresser, ok := beliggenhetsadr["adresse"].([]interface{}); ok && len(adresser) > 0 {
			if addr, ok := adresser[0].(string); ok {
				addrParts = append(addrParts, addr)
			}
		}
		postnr, _ := beliggenhetsadr["postnummer"].(string)
		poststed, _ := beliggenhetsadr["poststed"].(string)
		if postnr != "" && poststed != "" {
			addrParts = append(addrParts, fmt.Sprintf("%s %s", postnr, poststed))
		}
		if len(addrParts) > 0 {
			sb.WriteString(fmt.Sprintf("- **Adresse:** %s\n", strings.Join(addrParts, ", ")))
		}
	}

	if naring1, ok := obj["naeringskode1"].(map[string]interface{}); ok {
		if beskrivelse, ok := naring1["beskrivelse"].(string); ok {
			kode, _ := naring1["kode"].(string)
			sb.WriteString(fmt.Sprintf("- **Næring:** %s (%s)\n", beskrivelse, kode))
		}
	}

	sb.WriteString("\n")
	return sb.String()
}

// summarizeRollerCreation extracts key information from roller creation
func summarizeRollerCreation(data json.RawMessage) string {
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return "Registrerte roller\n\n"
	}

	var sb strings.Builder
	sb.WriteString("Registrerte roller\n\n")

	rollegrupper, ok := obj["rollegrupper"].([]interface{})
	if !ok || len(rollegrupper) == 0 {
		sb.WriteString("Ingen roller registrert\n\n")
		return sb.String()
	}

	for _, gruppe := range rollegrupper {
		if gruppeMap, ok := gruppe.(map[string]interface{}); ok {
			if roller, ok := gruppeMap["roller"].([]interface{}); ok {
				for _, rolle := range roller {
					if rolleMap, ok := rolle.(map[string]interface{}); ok {
						// Get person info
						var personNavn string
						if person, ok := rolleMap["person"].(map[string]interface{}); ok {
							if navn, ok := person["navn"].(map[string]interface{}); ok {
								fornavn, _ := navn["fornavn"].(string)
								etternavn, _ := navn["etternavn"].(string)
								personNavn = strings.TrimSpace(fmt.Sprintf("%s %s", fornavn, etternavn))
							}

							// Add birth year if available
							if fodselsdato, ok := person["fodselsdato"].(string); ok && len(fodselsdato) >= 4 {
								personNavn += fmt.Sprintf(" (f. %s)", fodselsdato[:4])
							}
						}

						// Get rolle type
						rolleType := ""
						if rolleTypeObj, ok := rolleMap["type"].(map[string]interface{}); ok {
							if beskrivelse, ok := rolleTypeObj["beskrivelse"].(string); ok {
								rolleType = beskrivelse
							}
						}

						if personNavn != "" && rolleType != "" {
							sb.WriteString(fmt.Sprintf("- %s: %s\n", rolleType, personNavn))
						}
					}
				}
			}
		}
	}

	sb.WriteString("\n")
	return sb.String()
}
