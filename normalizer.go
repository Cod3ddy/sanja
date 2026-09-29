package sanja

import (
	"fmt"
	"strings"
)

const (
	// MinimumLocalDigitsLength is the minimum number of subscriber digits accepted.
	MinimumLocalDigitsLength int = 7
	// MaximumLocalDigitsLength is the E.164 maximum subscriber digits (15 total minus a 1-digit country code minimum).
	MaximumLocalDigitsLength int = 12
	// e164MaxDigits is the E.164 hard cap: country code + subscriber <= 15 digits.
	e164MaxDigits int = 15
)

// Normalizer handles phone number normalization to E.164 format.
type Normalizer struct {
	countries      []Country
	defaultCountry *Country
	codeMap        map[string]*Country
}

// NewNormalizer creates a Normalizer with the given ISO 3166-1 alpha-2 code as the default country.
// Returns an error if the country code is not recognised.
func NewNormalizer(defaultCountryA2 string) (*Normalizer, error) {
	n := &Normalizer{
		countries: getCountries(),
		codeMap:   make(map[string]*Country),
	}

	for i := range n.countries {
		country := &n.countries[i]
		for _, code := range splitDialingCodes(country.DialingCode) {
			if _, taken := n.codeMap[code]; !taken || country.MainCountryForCode {
				n.codeMap[code] = country
			}
		}
	}

	n.defaultCountry = n.GetCountryByA2(defaultCountryA2)
	if n.defaultCountry == nil {
		return nil, fmt.Errorf("%w: %s", ErrUnknownCountry, defaultCountryA2)
	}

	return n, nil
}

// Normalize returns phone in E.164 format (+countryCodeSubscriber).
//
// Resolution order:
//  1. Numbers already prefixed with "+" are returned as-is (with length validation).
//  2. Numbers that begin with the default country's dialing code (no "+" or leading zeros)
//     receive a "+" prefix.
//  3. Everything else is treated as a local number: leading zeros are stripped and the
//     default country code is prepended.
//
// Returns ErrInvalidPhoneNumber for numbers that are too short,
// and ErrPhoneNumberTooLong for numbers that exceed the E.164 15-digit cap.
func (n *Normalizer) Normalize(phone string) (string, error) {
	cleaned := cleanPhone(phone)

	if strings.HasPrefix(cleaned, "+") {
		digits := cleaned[1:]
		if len(digits) < MinimumLocalDigitsLength {
			return "", ErrInvalidPhoneNumber
		}
		if len(digits) > e164MaxDigits {
			return "", ErrPhoneNumberTooLong
		}
		return cleaned, nil
	}

	if cleaned == "" || len(cleaned) < MinimumLocalDigitsLength {
		return "", ErrInvalidPhoneNumber
	}

	if n.hasCountryCode(cleaned) {
		if len(cleaned) > e164MaxDigits {
			return "", ErrPhoneNumberTooLong
		}
		return "+" + cleaned, nil
	}

	local := strings.TrimLeft(cleaned, "0")
	if len(local) < MinimumLocalDigitsLength {
		return "", ErrInvalidPhoneNumber
	}
	if len(local) > MaximumLocalDigitsLength {
		return "", ErrPhoneNumberTooLong
	}

	defaultCode := splitDialingCodes(n.defaultCountry.DialingCode)[0]
	result := "+" + defaultCode + local
	if len(result)-1 > e164MaxDigits {
		return "", ErrPhoneNumberTooLong
	}

	return result, nil
}

// NormalizeBulk normalizes a slice of phone numbers, returning a result and error per entry.
func (n *Normalizer) NormalizeBulk(phones []string) ([]string, []error) {
	normalized := make([]string, len(phones))
	errs := make([]error, len(phones))

	for i, phone := range phones {
		normalized[i], errs[i] = n.Normalize(phone)
	}

	return normalized, errs
}

// hasCountryCode reports whether phone (no "+" prefix) begins with the default country's
// dialing code. the leading "0" is treated as a local dial prefix and returns false immediately,
// preventing false matches against country codes that share digits with local number prefixes.
func (n *Normalizer) hasCountryCode(phone string) bool {
	if strings.HasPrefix(phone, "0") {
		return false
	}
	for _, code := range splitDialingCodes(n.defaultCountry.DialingCode) {
		if strings.HasPrefix(phone, code) {
			return true
		}
	}
	return false
}
