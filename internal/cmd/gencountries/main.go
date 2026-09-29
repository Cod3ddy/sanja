package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

const defaultMetadataURL = "https://raw.githubusercontent.com/google/libphonenumber/v9.0.40/resources/PhoneNumberMetadata.xml"

type territory struct {
	ID                  string `xml:"id,attr"`
	CountryCode         string `xml:"countryCode,attr"`
	MainCountryForCode  bool   `xml:"mainCountryForCode,attr"`
	InternationalPrefix string `xml:"internationalPrefix,attr"`
	NationalPrefix      string `xml:"nationalPrefix,attr"`
	NumberPattern       string `xml:"generalDesc>nationalNumberPattern"`
}

type metadata struct {
	Territories []territory `xml:"territories>territory"`
}

type country struct {
	Name                string `json:"name"`
	A2                  string `json:"a2"`
	A3                  string `json:"a3"`
	NumCode             int    `json:"numCode"`
	DialingCode         string `json:"dialingCode"`
	InternationalPrefix string `json:"internationalPrefix"`
	NationalPrefix      string `json:"nationalPrefix"`
	NumberPattern       string `json:"numberPattern"`
	MainCountryForCode  bool   `json:"mainCountryForCode,omitempty"`
	MinLocalDigits      int    `json:"minLocalDigits"`
	MaxLocalDigits      int    `json:"maxLocalDigits"`
}

func main() {
	metadataSource := flag.String("metadata", defaultMetadataURL, "libphonenumber PhoneNumberMetadata.xml, as a URL or a file path")
	countriesPath := flag.String("countries", "countries.json", "countries file to update in place")
	flag.Parse()

	territories, err := loadTerritories(*metadataSource)
	if err != nil {
		log.Fatal(err)
	}

	countries, err := readCountries(*countriesPath)
	if err != nil {
		log.Fatal(err)
	}

	territoriesPerCode := make(map[string]int)
	for _, t := range territories {
		territoriesPerCode[t.CountryCode]++
	}

	countriesPerCode := make(map[string]int)
	for _, c := range countries {
		for _, code := range strings.Split(c.DialingCode, ",") {
			countriesPerCode[strings.ReplaceAll(strings.TrimSpace(code), "-", "")]++
		}
	}

	for i := range countries {
		t, found := territories[countries[i].A2]
		if !found {
			log.Printf("%s is not in libphonenumber, keeping its current values", countries[i].A2)
			continue
		}

		countries[i].InternationalPrefix = t.InternationalPrefix
		countries[i].NationalPrefix = t.NationalPrefix
		countries[i].NumberPattern = strings.Join(strings.Fields(t.NumberPattern), "")
		onlyOneInLibphonenumber := territoriesPerCode[t.CountryCode] == 1
		sharedInOurData := countriesPerCode[t.CountryCode] > 1
		countries[i].MainCountryForCode = t.MainCountryForCode || (onlyOneInLibphonenumber && sharedInOurData)
	}

	err = writeCountries(*countriesPath, countries)
	if err != nil {
		log.Fatal(err)
	}
}

func loadTerritories(source string) (map[string]territory, error) {
	data, err := readSource(source)
	if err != nil {
		return nil, err
	}

	var parsed metadata
	err = xml.Unmarshal(data, &parsed)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", source, err)
	}

	territories := make(map[string]territory, len(parsed.Territories))
	for _, t := range parsed.Territories {
		territories[t.ID] = t
	}

	return territories, nil
}

func readSource(source string) ([]byte, error) {
	if !strings.HasPrefix(source, "https://") {
		return os.ReadFile(source)
	}

	response, err := http.Get(source)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download %s: %s", source, response.Status)
	}

	return io.ReadAll(response.Body)
}

func readCountries(path string) ([]country, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var countries []country
	err = json.Unmarshal(data, &countries)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	return countries, nil
}

func writeCountries(path string, countries []country) error {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")

	err := encoder.Encode(countries)
	if err != nil {
		return err
	}

	return os.WriteFile(path, buffer.Bytes(), 0o644)
}
