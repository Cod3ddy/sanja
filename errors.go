package sanja

import "errors"

var ErrInvalidPhoneNumber = errors.New("invalid phone number")
var ErrDefaultCountryNotFound = errors.New("no default country set")
var ErrPhoneNumberAndCountryCodeMismatch = errors.New("phone number does not match country code")
var ErrUnknownCountry = errors.New("invalid or unknown country")
var ErrPhoneNumberTooLong = errors.New("phone number exceeds E.164 maximum length")
