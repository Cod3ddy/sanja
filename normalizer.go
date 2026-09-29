package sanja

import (
	"fmt"
	"regexp"
	"strings"
)

const (
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
	return normalize(phone, n.defaultCountry)
}

func (n *Normalizer) NormalizeAndValidate(phone string) (string, error) {
	e164, err := n.Normalize(phone)
	if err != nil {
		return "", err
	}

	country := n.defaultCountry
	if _, ok := localDigits(e164, country); !ok {
		country, err = n.CountryForNumber(e164)
		if err != nil {
			return "", err
		}
	}

	if err := checkLocalDigits(e164, country); err != nil {
		return "", err
	}

	return e164, nil
}

func normalize(phone string, defaultCountry *Country) (string, error) {
	cleaned, err := cleanPhone(phone)
	if err != nil {
		return "", err
	}

	cleaned = replaceInternationalPrefix(cleaned, defaultCountry)

	var result string
	switch {
	case strings.HasPrefix(cleaned, "+"):
		result = cleaned
	case hasCountryCode(cleaned, defaultCountry):
		result = "+" + cleaned
	default:
		local := strings.TrimLeft(cleaned, "0")
		if local == "" {
			return "", ErrInvalidPhoneNumber
		}

		result = "+" + splitDialingCodes(defaultCountry.DialingCode)[0] + local
	}

	if result == "+" {
		return "", ErrInvalidPhoneNumber
	}

	if len(result)-len("+") > e164MaxDigits {
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
func hasCountryCode(phone string, country *Country) bool {
	if strings.HasPrefix(phone, "0") || len(phone) <= country.MaxLocalDigits {
		return false
	}

	for _, code := range splitDialingCodes(country.DialingCode) {
		local, found := strings.CutPrefix(phone, code)
		if found && len(local) >= country.MinLocalDigits && len(local) <= country.MaxLocalDigits {
			return true
		}
	}

	return false
}

func compileInternationalPrefix(pattern string) (*regexp.Regexp, error) {
	if pattern == "" {
		return nil, nil
	}

	return regexp.Compile("^(?:" + pattern + ")")
}

func replaceInternationalPrefix(phone string, country *Country) string {
	if country.internationalPrefixPattern == nil {
		return phone
	}

	prefix := country.internationalPrefixPattern.FindString(phone)
	rest := phone[len(prefix):]
	if prefix == "" || rest == "" || rest[0] == '0' {
		return phone
	}

	return "+" + rest
}
