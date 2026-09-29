package sanja

import (
	_ "embed"
	"encoding/json"
	"sync"
)

//go:embed countries.json
var countriesData []byte

var getCountries = sync.OnceValue(parseCountries)

func parseCountries() []Country {
	var result []Country
	if err := json.Unmarshal(countriesData, &result); err != nil {
		panic("sanja: failed to parse countries data: " + err.Error())
	}

	for i := range result {
		pattern, err := compileInternationalPrefix(result[i].InternationalPrefix)
		if err != nil {
			panic("sanja: international prefix for " + result[i].A2 + ": " + err.Error())
		}

		result[i].internationalPrefixPattern = pattern
	}

	return result
}
