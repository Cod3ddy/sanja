# Sanja 

> **Sanja** (pronounced *sahn-jah*) means "organize", "sort", or "order" in Chichewa - perfectly describing what this package does for phone numbers!

A lightweight Go package for normalizing and organizing international phone numbers. Sanja helps you clean, validate, and standardize phone numbers with country code handling.

## Features

- [x] **Phone Number Normalization**: Convert local numbers to international format
- [x] **Country Code Handling**: Automatic detection and addition of country codes
- [x] **Bulk Processing**: Normalize multiple numbers at once
- [x] **Flexible Configuration**: Set default country for local number handling
- [x] **Comprehensive Country Data**: 250+ countries with ISO codes and dialing codes

## Installation

```bash
go get github.com/cod3ddy/sanja
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/cod3ddy/sanja"
)

func main() {
    // Create a normalizer with Malawi as default country
    norm := sanja.NewNormalizer("MW")
    
    // Normalize a local Malawian number
    normalized, err := norm.Normalize("0886392814")
    if err != nil {
        panic(err)
    }
    
    fmt.Println(normalized) // Output: +265886392814
}
```

## Usage Examples

### Basic Normalization

```go
norm := sanja.NewNormalizer("US")

// Local number gets US country code
normalized, _ := norm.Normalize("555-123-4567")
// Result: +15551234567

// Already international - unchanged
normalized, _ := norm.Normalize("+442079460000")
// Result: +442079460000

// Number with country code but no + prefix
normalized, _ := norm.Normalize("265886392814")
// Result: +265886392814
```

### Bulk Processing

```go
norm := sanja.NewNormalizer("MW")

phones := []string{
    "0886392814",
    "265886392814", 
    "+265886392814",
    "00886392814",
}

results, errors := norm.NormalizeBulk(phones)

for i, phone := range results {
    if errors[i] != nil {
        fmt.Printf("Error with %s: %v\n", phones[i], errors[i])
    } else {
        fmt.Printf("Normalized: %s → %s\n", phones[i], phone)
    }
}
```

### Country Information

```go
norm := sanja.NewNormalizer("US")

// Get country by ISO A2 code
country := norm.GetCountryByA2("MW")
fmt.Printf("Malawi dialing code: %s\n", country.DialingCode)
// Output: Malawi dialing code: 265

// Get country by dialing code
country = norm.GetCountryByCode("44")
fmt.Printf("Country with code 44: %s\n", country.Name)
// Output: Country with code 44: United Kingdom

// Get the country an international number belongs to
country, err := norm.CountryForNumber("+16845551234")
fmt.Printf("Country for +1 684 555 1234: %s\n", country.Name)
// Output: Country for +1 684 555 1234: American Samoa
```

Some dialling codes are shared by several countries, such as `1` (the United States, Canada and much of the Caribbean) and `7` (Russia and Kazakhstan). For those codes, `GetCountryByCode` and `CountryForNumber` return the main country, so `+1 416 555 1234` (Toronto) comes back as the United States.

## Supported Countries

Sanja includes comprehensive country data for **250+ countries and territories** with:

- **ISO 3166-1 Alpha-2 codes** (e.g., `US`, `GB`, `MW`)
- **ISO 3166-1 Alpha-3 codes** (e.g., `USA`, `GBR`, `MWI`) 
- **ISO 3166-1 Numeric codes** (e.g., `840`, `826`, `454`)
- **International dialing codes** (e.g., `1`, `44`, `265`)

### Data Source
The country data used in this package was sourced from [Kaggle - Country 2ISO3UN Digit Code and Dialing Code](https://www.kaggle.com/datasets/migeruj/country-2iso3un-digit-code-and-dialing-code). The main country for each shared dialling code (`mainCountryForCode`) and the prefix each country dials to call abroad (`internationalPrefix`) come from Google’s [libphonenumber metadata](https://github.com/google/libphonenumber/blob/master/resources/PhoneNumberMetadata.xml).

## API Reference

### Types

```go
type Country struct {
    Name                string
    A2                  string // ISO Alpha-2 code (e.g., "US")
    A3                  string // ISO Alpha-3 code (e.g., "USA")
    NumCode             int    // ISO Numeric code (e.g., 840)
    DialingCode         string // International dialing code (e.g., "1"), or several separated by commas
    InternationalPrefix string // Pattern for what callers dial before a foreign number (e.g., "00", "011")
    MainCountryForCode  bool   // True for the country returned when several share a dialing code
    MinLocalDigits      int    // Fewest digits after the dialing code
    MaxLocalDigits      int    // Most digits after the dialing code
}

type Normalizer struct {
    countries      []Country 
	defaultCountry *Country
	codeMap        map[string]*Country
}
```

### Testing

## Handling Local Numbers

When a number doesn't have an international prefix, Sanja uses the default country:

```go
// With US as default
norm := sanja.NewNormalizer("US")
norm.Normalize("4151234567") // → "+14151234567"

// With Malawi as default  
norm := sanja.NewNormalizer("MW")
norm.Normalize("886392814") // → "+265886392814"

// The default country's prefix for calling abroad works like "+"
norm.Normalize("00447911123456") // → "+447911123456" (Malawi dials 00)
usNorm := sanja.NewNormalizer("US")
usNorm.Normalize("011265886392814") // → "+265886392814" (the US dials 011)
```

## Error Handling

```go
norm := sanja.NewNormalizer("US")

// Empty string
_, err := norm.Normalize("")
// err: "invalid phone number"

// Invalid default country
norm := sanja.NewNormalizer("INVALID")
_, err := norm.Normalize("123456789")
// err: "no default country set"
```
## Testing
for testing i used this package by stretchr:  [Assert](https://www.github.com/stretchr/testify/assert)

## Contributing

Contributions are welcome! Please feel free to submit pull requests, report bugs, or suggest new features.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Country data sourced from [Kaggle](https://www.kaggle.com/datasets/migeruj/country-2iso3un-digit-code-and-dialing-code)
- Name inspired by the Chichewa word "sanja" meaning to organize or put in order