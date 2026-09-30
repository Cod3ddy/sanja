package sanja

import "strings"

// cleanPhone removes all non-digit characters except +
func cleanPhone(phone string) (string, error) {
	var result strings.Builder
	for _, r := range phone {
		switch {
		case r >= '0' && r <= '9':
			result.WriteRune(r)
		case r == '+' && result.Len() == 0:
			result.WriteRune(r)
		case r == '+':
			return "", ErrInvalidPhoneNumber
		}
	}

	return result.String(), nil
}

// splitDialingCodes splits dialing code string into individual codes
func splitDialingCodes(dialingCode string) []string {
	var codes = strings.Split(dialingCode, ",")
	var result = make([]string, 0, len(codes))

	for _, code := range codes {
		cleanCode := strings.TrimSpace(code)

		// remove any non-digiti chars from the code (like  '1-684' to '1684')
		cleanCode = strings.ReplaceAll(cleanCode, "-", "")
		if cleanCode != "" {
			result = append(result, cleanCode)
		}
	}

	return result
}
