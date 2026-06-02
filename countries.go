package sanja

import (
	_ "embed"
	"encoding/json"
)

//go:embed countries.json
var countriesData []byte

func getCountries() []Country {
	var result []Country
	if err := json.Unmarshal(countriesData, &result); err != nil {
		panic("sanja: failed to parse countries data: " + err.Error())
	}
	return result
}
