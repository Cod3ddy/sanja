package sanja

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// Country represents a country with its dialing codes and E.164 digit constraints.
type Country struct {
	Name                string `json:"name"`
	A2                  string `json:"a2"`
	A3                  string `json:"a3"`
	NumCode             int    `json:"numCode"`
	DialingCode         string `json:"dialingCode"`
	InternationalPrefix string `json:"internationalPrefix"`
	NationalPrefix      string `json:"nationalPrefix"`
	NumberPattern       string `json:"numberPattern"`
	MainCountryForCode  bool   `json:"mainCountryForCode"`
	MinLocalDigits      int    `json:"minLocalDigits"`
	MaxLocalDigits      int    `json:"maxLocalDigits"`
	LocalDigitLengths   []int  `json:"localDigitLengths"`

	internationalPrefixPattern *regexp.Regexp
	numberPattern              *regexp.Regexp
}

const longestDialingCodeDigits = 4

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

func (n *Normalizer) CountryForNumber(e164 string) (*Country, error) {
	cleaned, err := cleanPhone(e164)
	if err != nil {
		return nil, err
	}

	digits, hasPlus := strings.CutPrefix(cleaned, "+")
	if !hasPlus || digits == "" {
		return nil, ErrInvalidPhoneNumber
	}

	for length := min(len(digits), longestDialingCodeDigits); length > 0; length-- {
		if country := n.GetCountryByCode(digits[:length]); country != nil {
			return country, nil
		}
	}

	return nil, fmt.Errorf("%w: %s", ErrUnknownCountry, e164)
}

// ValidatePhoneNumber validates that phone is a valid E.164 number for the given country.
// The number must include the country code prefix (with or without leading +).
// It checks E.164 total length (<= 15 digits) and the country's known subscriber digit range.
func (n *Normalizer) ValidatePhoneNumber(phone, countryA2 string) error {
	country := n.GetCountryByA2(countryA2)
	if country == nil {
		return fmt.Errorf("%w: %s", ErrUnknownCountry, countryA2)
	}

	e164, err := normalize(phone, country)
	if err != nil {
		return err
	}

	return checkLocalDigits(e164, country)
}

func checkLocalDigits(e164 string, country *Country) error {
	local, ok := localDigits(e164, country)
	if !ok {
		return fmt.Errorf("%w: expected code for %s", ErrPhoneNumberAndCountryCodeMismatch, country.Name)
	}

	if len(local) < country.MinLocalDigits {
		return fmt.Errorf("%w: %s requires at least %d local digits, got %d",
			ErrInvalidPhoneNumber, country.Name, country.MinLocalDigits, len(local))
	}

	if len(local) > country.MaxLocalDigits {
		return fmt.Errorf("%w: %s allows at most %d local digits, got %d",
			ErrInvalidPhoneNumber, country.Name, country.MaxLocalDigits, len(local))
	}

	if len(country.LocalDigitLengths) > 0 && !slices.Contains(country.LocalDigitLengths, len(local)) {
		return fmt.Errorf("%w: %s allows only %v local digits, got %d",
			ErrInvalidPhoneNumber, country.Name, country.LocalDigitLengths, len(local))
	}

	return nil
}

func localDigits(e164 string, country *Country) (string, bool) {
	digits := strings.TrimPrefix(e164, "+")
	for _, code := range splitDialingCodes(country.DialingCode) {
		if local, found := strings.CutPrefix(digits, code); found {
			return local, true
		}
	}

	return "", false
}
