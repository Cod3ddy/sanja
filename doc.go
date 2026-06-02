// Package sanja provides phone number normalization to E.164 format.
//
// Overview:
// Sanja normalizes phone numbers to international E.164 format (+countryCodeSubscriber)
// with support for 250+ countries and bulk processing.
//
// Example:
//
//	norm, err := sanja.NewNormalizer("US")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	normalized, err := norm.Normalize("555-123-4567")
//	// Result: "+15551234567"
//
// Features:
//   - Local to international (E.164) conversion
//   - Bulk phone number processing
//   - Country code detection for 250+ countries
//   - E.164 length validation (max 15 digits)
//   - Per-country subscriber digit validation via ValidatePhoneNumber
package sanja
