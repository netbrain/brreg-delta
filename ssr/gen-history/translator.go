package main

import (
	"fmt"
	"strings"
)

// TranslateChange converts a field change into human-readable Norwegian text
func TranslateChange(field string, oldValue, newValue interface{}) string {
	// Handle roller.json changes specially
	if field == "rollegrupper" {
		return translateRollegrupperChange(oldValue, newValue)
	}

	// Handle boolean fields with semantic meaning
	switch field {
	case "konkurs":
		return translateBoolean(oldValue, newValue, "Har meldt seg konkurs", "Konkurs avsluttet")
	case "underAvvikling":
		return translateBoolean(oldValue, newValue, "Under avvikling", "Avvikling avsluttet")
	case "underTvangsavviklingEllerTvangsopplosning":
		return translateBoolean(oldValue, newValue, "Under tvangsavvikling eller tvangsoppløsning", "Tvangsavvikling avsluttet")
	case "registrertIMvaregisteret":
		return translateBoolean(oldValue, newValue, "Registrert i Merverdiavgiftsregisteret", "Slettet fra Merverdiavgiftsregisteret")
	case "registrertIForetaksregisteret":
		return translateBoolean(oldValue, newValue, "Registrert i Foretaksregisteret", "Slettet fra Foretaksregisteret")
	case "registrertIStiftelsesregisteret":
		return translateBoolean(oldValue, newValue, "Registrert i Stiftelsesregisteret", "Slettet fra Stiftelsesregisteret")
	case "registrertIFrivillighetsregisteret":
		return translateBoolean(oldValue, newValue, "Registrert i Frivillighetsregisteret", "Slettet fra Frivillighetsregisteret")
	}

	// Handle specific named fields
	switch field {
	case "navn":
		return fmt.Sprintf("Navneendring fra «%s» til «%s»", formatValue(oldValue), formatValue(newValue))
	case "antallAnsatte":
		return translateEmployeeChange(oldValue, newValue)
	case "organisasjonsform.kode":
		return fmt.Sprintf("Endret organisasjonsform fra %s til %s", formatValue(oldValue), formatValue(newValue))
	case "sisteInnsendteAarsregnskap":
		if oldValue == nil {
			return fmt.Sprintf("Årsregnskap for %s innsendt", formatValue(newValue))
		}
		return fmt.Sprintf("Årsregnskap oppdatert fra %s til %s", formatValue(oldValue), formatValue(newValue))
	}

	// Handle address changes
	if strings.HasPrefix(field, "forretningsadresse.") || strings.HasPrefix(field, "postadresse.") || strings.HasPrefix(field, "beliggenhetsadresse.") {
		return translateAddressField(field, oldValue, newValue)
	}

	// Handle nested objects being added/removed
	if isComplexValue(newValue) && oldValue == nil {
		return fmt.Sprintf("La til %s", translateFieldName(field))
	}
	if isComplexValue(oldValue) && newValue == nil {
		return fmt.Sprintf("Fjernet %s", translateFieldName(field))
	}

	// Default: translate field name and show values
	if oldValue == nil {
		return fmt.Sprintf("%s satt til %s", translateFieldName(field), formatValue(newValue))
	}
	if newValue == nil {
		return fmt.Sprintf("%s fjernet (var %s)", translateFieldName(field), formatValue(oldValue))
	}

	return fmt.Sprintf("%s endret fra %s til %s", translateFieldName(field), formatValue(oldValue), formatValue(newValue))
}

// translateBoolean handles boolean field changes with custom messages
func translateBoolean(oldValue, newValue interface{}, trueMessage, falseMessage string) string {
	oldBool := toBool(oldValue)
	newBool := toBool(newValue)

	if !oldBool && newBool {
		return trueMessage
	}
	if oldBool && !newBool {
		return falseMessage
	}
	return ""
}

// translateEmployeeChange creates readable text for employee count changes
func translateEmployeeChange(oldValue, newValue interface{}) string {
	oldCount := toInt(oldValue)
	newCount := toInt(newValue)

	if oldCount == 0 && newCount > 0 {
		return fmt.Sprintf("Registrert med %d ansatte", newCount)
	}

	diff := newCount - oldCount
	if diff > 0 {
		return fmt.Sprintf("Antall ansatte økt fra %d til %d (+%d)", oldCount, newCount, diff)
	} else if diff < 0 {
		return fmt.Sprintf("Antall ansatte redusert fra %d til %d (%d)", oldCount, newCount, diff)
	}
	return ""
}

// translateAddressField translates address field changes
func translateAddressField(field string, oldValue, newValue interface{}) string {
	parts := strings.Split(field, ".")
	if len(parts) < 2 {
		return ""
	}

	addressType := parts[0]
	fieldName := parts[1]

	addressTypeNorwegian := map[string]string{
		"forretningsadresse":   "Forretningsadresse",
		"postadresse":          "Postadresse",
		"beliggenhetsadresse":  "Beliggenhetsadresse",
	}

	fieldNameNorwegian := map[string]string{
		"adresse":      "adresse",
		"postnummer":   "postnummer",
		"poststed":     "poststed",
		"kommune":      "kommune",
		"land":         "land",
	}

	addrType := addressTypeNorwegian[addressType]
	fName := fieldNameNorwegian[fieldName]

	if oldValue == nil {
		return fmt.Sprintf("%s %s satt til %s", addrType, fName, formatValue(newValue))
	}
	if newValue == nil {
		return fmt.Sprintf("%s %s fjernet", addrType, fName)
	}
	return fmt.Sprintf("%s %s endret fra %s til %s", addrType, fName, formatValue(oldValue), formatValue(newValue))
}

// translateFieldName translates field names to Norwegian
func translateFieldName(field string) string {
	translations := map[string]string{
		"organisasjonsnummer":                         "Organisasjonsnummer",
		"navn":                                        "Navn",
		"organisasjonsform":                           "Organisasjonsform",
		"organisasjonsform.kode":                      "Organisasjonsform",
		"organisasjonsform.beskrivelse":               "Organisasjonsform beskrivelse",
		"registreringsdatoEnhetsregisteret":           "Registreringsdato",
		"registrertIMvaregisteret":                    "MVA-registrering",
		"naeringskode1":                               "Næringskode",
		"naeringskode1.kode":                          "Næringskode",
		"naeringskode1.beskrivelse":                   "Næringsbeskrivelse",
		"antallAnsatte":                               "Antall ansatte",
		"forretningsadresse":                          "Forretningsadresse",
		"postadresse":                                 "Postadresse",
		"beliggenhetsadresse":                         "Beliggenhetsadresse",
		"institusjonellSektorkode":                    "Institusjonell sektorkode",
		"institusjonellSektorkode.kode":               "Institusjonell sektorkode",
		"institusjonellSektorkode.beskrivelse":        "Institusjonell sektor",
		"registrertIForetaksregisteret":               "Foretaksregister",
		"registrertIStiftelsesregisteret":             "Stiftelsesregister",
		"registrertIFrivillighetsregisteret":          "Frivillighetsregister",
		"sisteInnsendteAarsregnskap":                  "Siste årsregnskap",
		"konkurs":                                     "Konkurs",
		"underAvvikling":                              "Under avvikling",
		"underTvangsavviklingEllerTvangsopplosning":   "Tvangsavvikling",
		"maalform":                                    "Målform",
		"overordnetEnhet":                             "Overordnet enhet",
	}

	if norwegian, exists := translations[field]; exists {
		return norwegian
	}

	// Return field as-is if no translation exists
	return field
}

// formatValue formats a value for display
func formatValue(val interface{}) string {
	if val == nil {
		return "null"
	}

	switch v := val.(type) {
	case string:
		return v
	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%.2f", v)
	case bool:
		if v {
			return "ja"
		}
		return "nei"
	case []interface{}:
		if len(v) == 0 {
			return "[]"
		}
		// Format arrays as comma-separated list
		var parts []string
		for _, item := range v {
			parts = append(parts, formatValue(item))
		}
		return strings.Join(parts, ", ")
	default:
		return fmt.Sprintf("%v", v)
	}
}

// toBool converts a value to boolean
func toBool(val interface{}) bool {
	if val == nil {
		return false
	}
	if b, ok := val.(bool); ok {
		return b
	}
	return false
}

// toInt converts a value to integer
func toInt(val interface{}) int {
	if val == nil {
		return 0
	}
	if f, ok := val.(float64); ok {
		return int(f)
	}
	if i, ok := val.(int); ok {
		return i
	}
	return 0
}

// translateRollegrupperChange translates changes in roller.json
func translateRollegrupperChange(oldValue, newValue interface{}) string {
	oldRoller := extractRoller(oldValue)
	newRoller := extractRoller(newValue)

	oldGrupper := extractRollegruppeTypes(oldValue)
	newGrupper := extractRollegruppeTypes(newValue)

	// Find added and removed people
	var added []string
	var removed []string
	var removedGrupper []string

	// Build map of old roller by name for easy lookup
	oldMap := make(map[string]string) // name -> role
	for _, r := range oldRoller {
		oldMap[r.name] = r.role
	}

	// Build map of new roller
	newMap := make(map[string]string)
	for _, r := range newRoller {
		newMap[r.name] = r.role
	}

	// Find added
	for _, r := range newRoller {
		if _, exists := oldMap[r.name]; !exists {
			added = append(added, fmt.Sprintf("%s (%s)", r.name, r.role))
		}
	}

	// Find removed
	for _, r := range oldRoller {
		if _, exists := newMap[r.name]; !exists {
			removed = append(removed, fmt.Sprintf("%s (%s)", r.name, r.role))
		}
	}

	// Find removed rollegruppetyper (entire categories removed, like SIGN)
	for _, oldType := range oldGrupper {
		found := false
		for _, newType := range newGrupper {
			if oldType == newType {
				found = true
				break
			}
		}
		if !found {
			removedGrupper = append(removedGrupper, oldType)
		}
	}

	var parts []string
	if len(added) > 0 {
		if len(added) == 1 {
			parts = append(parts, fmt.Sprintf("La til %s", added[0]))
		} else {
			parts = append(parts, fmt.Sprintf("La til %s", strings.Join(added, ", ")))
		}
	}

	if len(removed) > 0 {
		if len(removed) == 1 {
			parts = append(parts, fmt.Sprintf("Fjernet %s", removed[0]))
		} else {
			parts = append(parts, fmt.Sprintf("Fjernet %s", strings.Join(removed, ", ")))
		}
	}

	if len(removedGrupper) > 0 {
		for _, gruppe := range removedGrupper {
			parts = append(parts, fmt.Sprintf("Fjernet rollegruppe: %s", gruppe))
		}
	}

	if len(parts) == 0 {
		return "Rolleendringer"
	}

	return strings.Join(parts, "; ")
}

type rolle struct {
	name string
	role string
}

// extractRoller extracts person names and roles from rollegrupper
func extractRoller(val interface{}) []rolle {
	var result []rolle

	if val == nil {
		return result
	}

	// val should be []interface{} (array of rollegrupper)
	grupper, ok := val.([]interface{})
	if !ok {
		return result
	}

	for _, gruppe := range grupper {
		gruppeMap, ok := gruppe.(map[string]interface{})
		if !ok {
			continue
		}

		// Get rolle type (Styre, Daglig ledelse, etc)
		var gruppeType string
		if typeObj, ok := gruppeMap["type"].(map[string]interface{}); ok {
			if beskrivelse, ok := typeObj["beskrivelse"].(string); ok {
				gruppeType = beskrivelse
			}
		}

		// Get roller array
		roller, ok := gruppeMap["roller"].([]interface{})
		if !ok {
			continue
		}

		for _, r := range roller {
			rolleMap, ok := r.(map[string]interface{})
			if !ok {
				continue
			}

			// Get role type
			var rolleType string
			if typeObj, ok := rolleMap["type"].(map[string]interface{}); ok {
				if beskrivelse, ok := typeObj["beskrivelse"].(string); ok {
					rolleType = beskrivelse
				}
			}

			// Get person name
			var personName string
			if personObj, ok := rolleMap["person"].(map[string]interface{}); ok {
				if navnObj, ok := personObj["navn"].(map[string]interface{}); ok {
					fornavn, _ := navnObj["fornavn"].(string)
					etternavn, _ := navnObj["etternavn"].(string)
					personName = strings.TrimSpace(fornavn + " " + etternavn)
				}
			}

			if personName != "" && rolleType != "" {
				result = append(result, rolle{
					name: personName,
					role: rolleType,
				})
			}
		}

		// Suppress unused variable warning
		_ = gruppeType
	}

	return result
}

// extractRollegruppeTypes extracts rollegruppetyper from rollegrupper
func extractRollegruppeTypes(val interface{}) []string {
	var result []string

	if val == nil {
		return result
	}

	// val should be []interface{} (array of rollegrupper)
	grupper, ok := val.([]interface{})
	if !ok {
		return result
	}

	for _, gruppe := range grupper {
		gruppeMap, ok := gruppe.(map[string]interface{})
		if !ok {
			continue
		}

		// Get rolle type (Styre, Daglig ledelse, etc)
		if typeObj, ok := gruppeMap["type"].(map[string]interface{}); ok {
			if beskrivelse, ok := typeObj["beskrivelse"].(string); ok {
				result = append(result, beskrivelse)
			}
		}
	}

	return result
}
