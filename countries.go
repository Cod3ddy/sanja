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
	return result
}
