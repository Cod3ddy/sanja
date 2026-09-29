package sanja

import (
	"fmt"
	"strings"
)

// Country represents a country with its dialing codes and E.164 digit constraints.
type Country struct {
	Name               string `json:"name"`
	A2                 string `json:"a2"`
	A3                 string `json:"a3"`
	NumCode            int    `json:"numCode"`
	DialingCode        string `json:"dialingCode"`
	MainCountryForCode bool   `json:"mainCountryForCode"`
	MinLocalDigits     int    `json:"minLocalDigits"`
	MaxLocalDigits     int    `json:"maxLocalDigits"`
}

// GetCountryByA2 returns a country by its ISO 3166-1 alpha-2 code, or nil if not found.
func (n *Normalizer) GetCountryByA2(a2 string) *Country {
	for i := range n.countries {
		if strings.EqualFold(n.countries[i].A2, a2) {
			return &n.countries[i]
		}
	}

	return nil
}

// GetCountryByCode returns a country by its dialing code, or nil if not found.
func (n *Normalizer) GetCountryByCode(code string) *Country {
	return n.codeMap[code]
}

// ValidatePhoneNumber validates that phone is a valid E.164 number for the given country.
// The number must include the country code prefix (with or without leading +).
// It checks E.164 total length (<= 15 digits) and the country's known subscriber digit range.
func (n *Normalizer) ValidatePhoneNumber(phone, countryA2 string) error {
	country := n.GetCountryByA2(countryA2)
	if country == nil {
		return fmt.Errorf("%w: %s", ErrUnknownCountry, countryA2)
	}

	cleaned := cleanPhone(phone)
	if cleaned == "" {
		return ErrInvalidPhoneNumber
	}

	var matchedCode string
	for _, code := range splitDialingCodes(country.DialingCode) {
		if strings.HasPrefix(cleaned, "+"+code) || strings.HasPrefix(cleaned, code) {
			matchedCode = code
			break
		}
	}

	if matchedCode == "" {
		return fmt.Errorf("%w: expected code for %s", ErrPhoneNumberAndCountryCodeMismatch, country.Name)
	}

	withoutPlus := strings.TrimPrefix(cleaned, "+")
	subscriber := withoutPlus[len(matchedCode):]

	if len(matchedCode)+len(subscriber) > 15 {
		return fmt.Errorf("%w: exceeds E.164 maximum of 15 digits", ErrPhoneNumberTooLong)
	}

	if len(subscriber) < country.MinLocalDigits {
		return fmt.Errorf("%w: %s requires at least %d local digits, got %d",
			ErrInvalidPhoneNumber, country.Name, country.MinLocalDigits, len(subscriber))
	}

	if len(subscriber) > country.MaxLocalDigits {
		return fmt.Errorf("%w: %s allows at most %d local digits, got %d",
			ErrInvalidPhoneNumber, country.Name, country.MaxLocalDigits, len(subscriber))
	}

	return nil
}
