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
	"slices"
	"strconv"
	"strings"
)

const defaultMetadataURL = "https://raw.githubusercontent.com/google/libphonenumber/v9.0.40/resources/PhoneNumberMetadata.xml"

var countriesWithLibphonenumberLengths = []string{
	"AO", "BF", "BI", "BJ", "BW", "CD", "CF", "CG", "CI", "CM", "CV", "DJ", "DZ", "EG", "EH",
	"ER", "ET", "GA", "GH", "GM", "GN", "GQ", "GW", "IO", "KE", "KM", "LR", "LS", "LY", "MA",
	"MG", "ML", "MR", "MU", "MW", "MZ", "NA", "NE", "NG", "RE", "RW", "SC", "SD", "SH", "SL",
	"SN", "SO", "SS", "ST", "SZ", "TD", "TG", "TN", "TZ", "UG", "YT", "ZA", "ZM", "ZW",
}

type numberType struct {
	PossibleLengths struct {
		National string `xml:"national,attr"`
	} `xml:"possibleLengths"`
	ExampleNumber string `xml:"exampleNumber"`
}

type territory struct {
	ID                  string     `xml:"id,attr"`
	CountryCode         string     `xml:"countryCode,attr"`
	MainCountryForCode  bool       `xml:"mainCountryForCode,attr"`
	InternationalPrefix string     `xml:"internationalPrefix,attr"`
	NationalPrefix      string     `xml:"nationalPrefix,attr"`
	NumberPattern       string     `xml:"generalDesc>nationalNumberPattern"`
	FixedLine           numberType `xml:"fixedLine"`
	Mobile              numberType `xml:"mobile"`
	TollFree            numberType `xml:"tollFree"`
	PremiumRate         numberType `xml:"premiumRate"`
	SharedCost          numberType `xml:"sharedCost"`
	PersonalNumber      numberType `xml:"personalNumber"`
	VoIP                numberType `xml:"voip"`
	Pager               numberType `xml:"pager"`
	UAN                 numberType `xml:"uan"`
	Voicemail           numberType `xml:"voicemail"`
}

func (t territory) numberTypes() []numberType {
	return []numberType{
		t.FixedLine, t.Mobile, t.TollFree, t.PremiumRate, t.SharedCost,
		t.PersonalNumber, t.VoIP, t.Pager, t.UAN, t.Voicemail,
	}
}

type example struct {
	A2          string `json:"a2"`
	CountryCode string `json:"countryCode"`
	Type        string `json:"type"`
	Number      string `json:"number"`
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
	LocalDigitLengths   []int  `json:"localDigitLengths,omitempty"`
}

func main() {
	metadataSource := flag.String("metadata", defaultMetadataURL, "libphonenumber PhoneNumberMetadata.xml, as a URL or a file path")
	countriesPath := flag.String("countries", "countries.json", "countries file to update in place")
	examplesPath := flag.String("examples", "testdata/libphonenumber_examples.json", "file to write example numbers to, for tests")
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

		if !slices.Contains(countriesWithLibphonenumberLengths, t.ID) {
			continue
		}

		lengths, err := possibleLengths(t)
		if err != nil {
			log.Fatalf("%s: %v", t.ID, err)
		}

		countries[i].MinLocalDigits = slices.Min(lengths)
		countries[i].MaxLocalDigits = slices.Max(lengths)
		countries[i].LocalDigitLengths = lengths
	}

	err = writeJSON(*countriesPath, countries)
	if err != nil {
		log.Fatal(err)
	}

	err = writeJSON(*examplesPath, examples(territories))
	if err != nil {
		log.Fatal(err)
	}
}

func possibleLengths(t territory) ([]int, error) {
	var lengths []int
	for _, numberType := range t.numberTypes() {
		typeLengths, err := parseLengths(numberType.PossibleLengths.National)
		if err != nil {
			return nil, err
		}

		lengths = append(lengths, typeLengths...)
	}

	if len(lengths) == 0 {
		return nil, fmt.Errorf("no possible lengths")
	}

	slices.Sort(lengths)

	return slices.Compact(lengths), nil
}

func parseLengths(spec string) ([]int, error) {
	if spec == "" {
		return nil, nil
	}

	var lengths []int
	for _, part := range strings.Split(spec, ",") {
		bounds := strings.Split(strings.Trim(part, "[]"), "-")

		low, err := strconv.Atoi(bounds[0])
		if err != nil {
			return nil, fmt.Errorf("possible lengths %q: %w", spec, err)
		}

		high, err := strconv.Atoi(bounds[len(bounds)-1])
		if err != nil {
			return nil, fmt.Errorf("possible lengths %q: %w", spec, err)
		}

		for length := low; length <= high; length++ {
			lengths = append(lengths, length)
		}
	}

	return lengths, nil
}

func examples(territories map[string]territory) []example {
	var result []example
	for _, a2 := range countriesWithLibphonenumberLengths {
		t := territories[a2]
		for _, typed := range []struct {
			name       string
			numberType numberType
		}{
			{name: "fixedLine", numberType: t.FixedLine},
			{name: "mobile", numberType: t.Mobile},
		} {
			if typed.numberType.ExampleNumber == "" {
				continue
			}

			result = append(result, example{
				A2:          a2,
				CountryCode: t.CountryCode,
				Type:        typed.name,
				Number:      typed.numberType.ExampleNumber,
			})
		}
	}

	return result
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

func writeJSON(path string, value any) error {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")

	err := encoder.Encode(value)
	if err != nil {
		return err
	}

	return os.WriteFile(path, buffer.Bytes(), 0o644)
}
