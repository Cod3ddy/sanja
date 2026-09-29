package sanja

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustNormalizer(t *testing.T, a2 string) *Normalizer {
	t.Helper()
	n, err := NewNormalizer(a2)
	require.NoError(t, err)
	return n
}

func TestNewNormalizer_InvalidCountry(t *testing.T) {
	_, err := NewNormalizer("XX")
	assert.ErrorIs(t, err, ErrUnknownCountry)
}

func TestNewNormalizer_ValidCountry(t *testing.T) {
	n, err := NewNormalizer("MW")
	assert.NoError(t, err)
	assert.NotNil(t, n)
}

func TestNewNormalizer_LowerCaseCountry(t *testing.T) {
	n, err := NewNormalizer("mw")
	require.NoError(t, err)
	assert.Equal(t, "MW", n.defaultCountry.A2)
}

func TestNormalize(t *testing.T) {
	norm := mustNormalizer(t, "MW")

	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
		errorIs     error
	}{
		{name: "local number with leading zero", input: "0886392814", expected: "+265886392814"},
		{name: "local number without leading zero", input: "886392814", expected: "+265886392814"},
		{name: "already normalized with plus", input: "+265886392814", expected: "+265886392814"},
		{name: "number with country code without plus", input: "265886392814", expected: "+265886392814"},
		{name: "number with multiple leading zeros", input: "000886392814", expected: "+265886392814"},
		{name: "number with spaces", input: "088 639 2814", expected: "+265886392814"},
		{name: "number with dashes", input: "088-639-2814", expected: "+265886392814"},
		{name: "number with parentheses", input: "(088)6392814", expected: "+265886392814"},
		{name: "number with mixed formatting", input: "+265 (88) 639-2814", expected: "+265886392814"},

		{name: "number with letters stripped", input: "088-639A2814", expected: "+265886392814"},

		{name: "US number with country code", input: "+12025550123", expected: "+12025550123"},
		{name: "UK number with country code", input: "+442079460000", expected: "+442079460000"},
		{name: "South Africa number", input: "+27821234567", expected: "+27821234567"},

		{name: "empty string", input: "", expectError: true, errorIs: ErrInvalidPhoneNumber},
		{name: "only special characters", input: "+-() ", expectError: true, errorIs: ErrInvalidPhoneNumber},
		{name: "too short number", input: "123", expectError: true, errorIs: ErrInvalidPhoneNumber},
		{name: "exceeds E.164 max", input: "08863928149999999", expectError: true, errorIs: ErrPhoneNumberTooLong},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := norm.Normalize(tc.input)
			if tc.expectError {
				assert.Error(t, err)
				if tc.errorIs != nil {
					assert.ErrorIs(t, err, tc.errorIs)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

func TestNormalize_PlusSign(t *testing.T) {
	norm := mustNormalizer(t, "MW")

	testCases := []struct {
		name        string
		input       string
		expected    string
		expectedErr error
	}{
		{name: "plus after spaces and brackets", input: " (+265) 88 639 2814", expected: "+265886392814"},
		{name: "double plus", input: "++265886392814", expectedErr: ErrInvalidPhoneNumber},
		{name: "plus in the middle", input: "+265+886392814", expectedErr: ErrInvalidPhoneNumber},
		{name: "plus at the end", input: "0886392814+", expectedErr: ErrInvalidPhoneNumber},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := norm.Normalize(tc.input)
			if tc.expectedErr != nil {
				assert.ErrorIs(t, err, tc.expectedErr)
				assert.Empty(t, result)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestNormalize_InternationalPrefix(t *testing.T) {
	testCases := []struct {
		name           string
		defaultCountry string
		input          string
		expected       string
	}{
		{name: "00 to own country", defaultCountry: "MW", input: "00265886392814", expected: "+265886392814"},
		{name: "00 to another country", defaultCountry: "MW", input: "00447911123456", expected: "+447911123456"},
		{name: "00 with spaces", defaultCountry: "MW", input: "00 44 7911 123456", expected: "+447911123456"},
		{name: "011 from the US", defaultCountry: "US", input: "011265886392814", expected: "+265886392814"},
		{name: "00 is not international in the US", defaultCountry: "US", input: "002025551234", expected: "+12025551234"},
		{name: "0011 from Australia", defaultCountry: "AU", input: "0011265886392814", expected: "+265886392814"},
		{name: "010 from Japan", defaultCountry: "JP", input: "010265886392814", expected: "+265886392814"},
		{name: "810 from Russia", defaultCountry: "RU", input: "810265886392814", expected: "+265886392814"},
		{name: "00 with carrier code from Brazil", defaultCountry: "BR", input: "0021265886392814", expected: "+265886392814"},
		{name: "prefix followed by zero stays local", defaultCountry: "MW", input: "000886392814", expected: "+265886392814"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			norm := mustNormalizer(t, tc.defaultCountry)

			result, err := norm.Normalize(tc.input)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestInternationalPrefixesCompile(t *testing.T) {
	for _, country := range getCountries() {
		_, err := compileInternationalPrefix(country.InternationalPrefix)
		assert.NoError(t, err, country.A2)
	}
}

func TestCountryForNumber_PlusSign(t *testing.T) {
	norm := mustNormalizer(t, "MW")

	_, err := norm.CountryForNumber("+265+886392814")
	assert.ErrorIs(t, err, ErrInvalidPhoneNumber)
}

func TestValidatePhoneNumber_PlusSign(t *testing.T) {
	norm := mustNormalizer(t, "MW")

	err := norm.ValidatePhoneNumber("+265+99123456", "MW")
	assert.ErrorIs(t, err, ErrInvalidPhoneNumber)
}

func TestNormalizeBulk(t *testing.T) {
	norm := mustNormalizer(t, "MW")

	tests := []struct {
		name           string
		input          []string
		errorCount     int
		expectedOutput []string
	}{
		{
			name: "mixed valid and invalid numbers",
			input: []string{
				"0886392814",
				"265886392814",
				"+265886392814",
				"00265886392814",
				"",
				"123",
				"+447911123456",
			},
			errorCount: 2,
			expectedOutput: []string{
				"+265886392814",
				"+265886392814",
				"+265886392814",
				"+265886392814",
				"",
				"",
				"+447911123456",
			},
		},
		{
			name: "all valid numbers",
			input: []string{
				"0886392814",
				"886392814",
				"+265886392814",
				"265886392814",
			},
			errorCount: 0,
			expectedOutput: []string{
				"+265886392814",
				"+265886392814",
				"+265886392814",
				"+265886392814",
			},
		},
		{
			name:           "all invalid numbers",
			input:          []string{"", "abc", "123"},
			errorCount:     3,
			expectedOutput: []string{"", "", ""},
		},
		{
			name:           "empty slice",
			input:          []string{},
			errorCount:     0,
			expectedOutput: []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			results, errs := norm.NormalizeBulk(tc.input)

			assert.Equal(t, len(tc.input), len(results))
			assert.Equal(t, len(tc.input), len(errs))

			actualErrors := 0
			for _, err := range errs {
				if err != nil {
					actualErrors++
				}
			}
			assert.Equal(t, tc.errorCount, actualErrors)

			for i, expected := range tc.expectedOutput {
				if i < len(results) {
					assert.Equal(t, expected, results[i])
				}
			}
		})
	}
}

func TestNormalizeWithDifferentDefaultCountries(t *testing.T) {
	tests := []struct {
		defaultCountry string
		input          string
		expected       string
	}{
		{defaultCountry: "US", input: "5550123456", expected: "+15550123456"},
		{defaultCountry: "GB", input: "2079460000", expected: "+442079460000"},
		{defaultCountry: "ZA", input: "821234567", expected: "+27821234567"},
		{defaultCountry: "KE", input: "712345678", expected: "+254712345678"},
	}

	for _, tc := range tests {
		t.Run(tc.defaultCountry, func(t *testing.T) {
			norm := mustNormalizer(t, tc.defaultCountry)
			result, err := norm.Normalize(tc.input)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGetCountryByA2(t *testing.T) {
	norm := mustNormalizer(t, "MW")

	t.Run("found", func(t *testing.T) {
		c := norm.GetCountryByA2("US")
		require.NotNil(t, c)
		assert.Equal(t, "United States", c.Name)
		assert.Equal(t, "1", c.DialingCode)
	})

	t.Run("lower case", func(t *testing.T) {
		c := norm.GetCountryByA2("us")
		require.NotNil(t, c)
		assert.Equal(t, "US", c.A2)
	})

	t.Run("not found returns nil", func(t *testing.T) {
		c := norm.GetCountryByA2("XX")
		assert.Nil(t, c)
	})
}

func TestGetCountryByCode(t *testing.T) {
	norm := mustNormalizer(t, "MW")

	t.Run("found by single code", func(t *testing.T) {
		c := norm.GetCountryByCode("265")
		require.NotNil(t, c)
		assert.Equal(t, "MW", c.A2)
	})

	t.Run("found by multi-code country", func(t *testing.T) {
		c := norm.GetCountryByCode("1829")
		require.NotNil(t, c)
		assert.Equal(t, "DO", c.A2)
	})

	t.Run("not found returns nil", func(t *testing.T) {
		c := norm.GetCountryByCode("9999")
		assert.Nil(t, c)
	})
}

func TestGetCountryByCode_SharedCodes(t *testing.T) {
	norm := mustNormalizer(t, "MW")

	testCases := []struct {
		code     string
		expected string
	}{
		{code: "1", expected: "US"},
		{code: "7", expected: "RU"},
		{code: "44", expected: "GB"},
		{code: "47", expected: "NO"},
		{code: "61", expected: "AU"},
		{code: "212", expected: "MA"},
		{code: "262", expected: "RE"},
		{code: "500", expected: "FK"},
		{code: "590", expected: "GP"},
		{code: "599", expected: "CW"},
		{code: "672", expected: "NF"},
	}

	for _, tc := range testCases {
		t.Run(tc.code, func(t *testing.T) {
			c := norm.GetCountryByCode(tc.code)
			require.NotNil(t, c)
			assert.Equal(t, tc.expected, c.A2)
		})
	}
}

func TestDialingCodesFitLongestDialingCodeDigits(t *testing.T) {
	for _, country := range getCountries() {
		for _, code := range splitDialingCodes(country.DialingCode) {
			assert.LessOrEqual(t, len(code), longestDialingCodeDigits, country.A2)
		}
	}
}

func TestCountryForNumber(t *testing.T) {
	norm := mustNormalizer(t, "MW")

	testCases := []struct {
		name        string
		phone       string
		expected    string
		expectedErr error
	}{
		{name: "three-digit code", phone: "+265991234567", expected: "MW"},
		{name: "formatted number", phone: "+265 99 123 4567", expected: "MW"},
		{name: "longer code wins over shared code", phone: "+16845551234", expected: "AS"},
		{name: "second code of a multi-code country", phone: "+18295551234", expected: "DO"},
		{name: "shared code gives main country", phone: "+12025551234", expected: "US"},
		{name: "shared code 44", phone: "+447911123456", expected: "GB"},
		{name: "no plus", phone: "265991234567", expectedErr: ErrInvalidPhoneNumber},
		{name: "empty", phone: "", expectedErr: ErrInvalidPhoneNumber},
		{name: "plus only", phone: "+", expectedErr: ErrInvalidPhoneNumber},
		{name: "unassigned code", phone: "+8081234567", expectedErr: ErrUnknownCountry},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c, err := norm.CountryForNumber(tc.phone)
			if tc.expectedErr != nil {
				assert.ErrorIs(t, err, tc.expectedErr)
				assert.Nil(t, c)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expected, c.A2)
		})
	}
}

func TestValidatePhoneNumber(t *testing.T) {
	norm := mustNormalizer(t, "MW")

	tests := []struct {
		name    string
		phone   string
		country string
		wantErr error
	}{
		{
			name:    "valid Malawi number",
			phone:   "+265886392814",
			country: "MW",
		},
		{
			name:    "valid US number",
			phone:   "+12025550123",
			country: "US",
		},
		{
			name:    "valid UK number",
			phone:   "+447911123456",
			country: "GB",
		},
		{
			name:    "wrong country code",
			phone:   "+27821234567",
			country: "MW",
			wantErr: ErrPhoneNumberAndCountryCodeMismatch,
		},
		{
			name:    "unknown country",
			phone:   "+265886392814",
			country: "XX",
			wantErr: ErrUnknownCountry,
		},
		{
			name:    "too few local digits for Malawi",
			phone:   "+26512345",
			country: "MW",
			wantErr: ErrInvalidPhoneNumber,
		},
		{
			name:    "exceeds E.164 max",
			phone:   "+2658863928149999",
			country: "MW",
			wantErr: ErrPhoneNumberTooLong,
		},
		{
			name:    "valid Dominican Republic secondary code",
			phone:   "+18291234567",
			country: "DO",
		},
		{
			name:    "empty phone",
			phone:   "",
			country: "MW",
			wantErr: ErrInvalidPhoneNumber,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := norm.ValidatePhoneNumber(tc.phone, tc.country)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func BenchmarkNormalize(b *testing.B) {
	norm, _ := NewNormalizer("MW")
	phoneNumbers := []string{
		"0886392814",
		"265886392814",
		"+265886392814",
		"886392814",
	}

	for b.Loop() {
		for _, phone := range phoneNumbers {
			norm.Normalize(phone)
		}
	}
}

func BenchmarkNormalizeBulk(b *testing.B) {
	norm, _ := NewNormalizer("MW")
	phoneNumbers := []string{
		"0886392814", "265886392814", "+265886392814", "886392814",
		"0999123456", "265999123456", "+265999123456", "999123456",
	}

	b.ResetTimer()
	for b.Loop() {
		norm.NormalizeBulk(phoneNumbers)
	}
}
