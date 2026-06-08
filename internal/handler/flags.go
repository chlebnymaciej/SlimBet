package handler

import (
	"fmt"
	"html/template"
)

// teamFlags maps football-data.org team names to ISO 3166-1 alpha-2 codes (lowercase).
// Used to build flagcdn.com image URLs. Special subdivisions: gb-eng, gb-sct, gb-wls.
var teamFlags = map[string]string{
	// CONCACAF
	"United States":       "us",
	"USA":                 "us",
	"Mexico":              "mx",
	"Canada":              "ca",
	"Panama":              "pa",
	"Costa Rica":          "cr",
	"Honduras":            "hn",
	"Jamaica":             "jm",
	"El Salvador":         "sv",
	"Trinidad and Tobago": "tt",
	"Cuba":                "cu",
	"Haiti":               "ht",
	"Guatemala":           "gt",

	// CONMEBOL
	"Brazil":    "br",
	"Argentina": "ar",
	"Colombia":  "co",
	"Uruguay":   "uy",
	"Ecuador":   "ec",
	"Venezuela": "ve",
	"Chile":     "cl",
	"Paraguay":  "py",
	"Peru":      "pe",
	"Bolivia":   "bo",

	// UEFA
	"France":                 "fr",
	"Germany":                "de",
	"Spain":                  "es",
	"England":                "gb-eng",
	"Portugal":               "pt",
	"Netherlands":            "nl",
	"Belgium":                "be",
	"Italy":                  "it",
	"Poland":                 "pl",
	"Switzerland":            "ch",
	"Austria":                "at",
	"Croatia":                "hr",
	"Denmark":                "dk",
	"Serbia":                 "rs",
	"Scotland":               "gb-sct",
	"Turkey":                 "tr",
	"Ukraine":                "ua",
	"Hungary":                "hu",
	"Slovakia":               "sk",
	"Romania":                "ro",
	"Czech Republic":         "cz",
	"Czechia":                "cz",
	"Albania":                "al",
	"Slovenia":               "si",
	"Georgia":                "ge",
	"Wales":                  "gb-wls",
	"Greece":                 "gr",
	"Norway":                 "no",
	"Sweden":                 "se",
	"Finland":                "fi",
	"Iceland":                "is",
	"North Macedonia":        "mk",
	"Bosnia and Herzegovina": "ba",
	"Bosnia-Herzegovina":     "ba",
	"Kosovo":                 "xk",
	"Ireland":                "ie",

	// CAF
	"Morocco":            "ma",
	"Senegal":            "sn",
	"Cameroon":           "cm",
	"Egypt":              "eg",
	"Nigeria":            "ng",
	"Côte d'Ivoire":      "ci",
	"Ivory Coast":        "ci",
	"Mali":               "ml",
	"Algeria":            "dz",
	"Tunisia":            "tn",
	"Ghana":              "gh",
	"Congo DR":           "cd",
	"DR Congo":           "cd",
	"South Africa":       "za",
	"Tanzania":           "tz",
	"Zambia":             "zm",
	"Uganda":             "ug",
	"Cape Verde":         "cv",
	"Cape Verde Islands": "cv",
	"Mozambique":         "mz",
	"Burkina Faso":       "bf",

	// AFC
	"Japan":                "jp",
	"South Korea":          "kr",
	"Korea Republic":       "kr",
	"Saudi Arabia":         "sa",
	"Australia":            "au",
	"Iran":                 "ir",
	"Qatar":                "qa",
	"Iraq":                 "iq",
	"Jordan":               "jo",
	"Oman":                 "om",
	"Uzbekistan":           "uz",
	"Indonesia":            "id",
	"China PR":             "cn",
	"China":                "cn",
	"Palestine":            "ps",
	"Bahrain":              "bh",
	"United Arab Emirates": "ae",
	"UAE":                  "ae",
	"Kuwait":               "kw",

	// OFC
	"New Zealand": "nz",
	"Curaçao":     "cw",
	"Curacao":     "cw",
}

func teamFlag(name string) template.HTML {
	code := teamFlags[name]
	if code == "" {
		return ""
	}
	return template.HTML(fmt.Sprintf(
		`<img src="https://flagcdn.com/20x15/%s.png" alt="%s" style="vertical-align:middle;margin-right:.2rem;border-radius:2px">`,
		code, name,
	))
}
