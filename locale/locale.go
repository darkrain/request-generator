package locale

import "strings"

type Lang string

const (
	EN Lang = "en"
	RU Lang = "ru"
	AR Lang = "ar"
)

// LangTitle returns the native display name for a language.
var LangTitles = map[Lang]string{
	EN: "English",
	RU: "Русский",
	AR: "العربية",
}

// FieldI18n holds per-field translations for the i18n response block.
type FieldI18n struct {
	Title   string            `json:"title"`
	Options map[string]string `json:"options,omitempty"`
}

// ParseAcceptLanguage parses the Accept-Language header and returns
// the best matching supported locale, or defaultLang if none match.
func ParseAcceptLanguage(header string, supported []Lang, defaultLang Lang) Lang {
	if header == "" {
		return defaultLang
	}
	parts := strings.Split(header, ",")
	for _, part := range parts {
		tag := strings.TrimSpace(strings.SplitN(part, ";", 2)[0])
		tag = strings.ToLower(tag)
		for _, lang := range supported {
			if strings.HasPrefix(tag, string(lang)) {
				return lang
			}
		}
	}
	return defaultLang
}

// Resolve returns the localized string from a translations map, falling back to the default.
func Resolve(translations map[string]string, lang Lang, fallback string) string {
	if translations != nil {
		if v, ok := translations[string(lang)]; ok {
			return v
		}
	}
	return fallback
}

var messages = map[Lang]map[string]string{
	RU: {
		"required": "%s - не может быть пустым",
		"in":       "%s - должен быть одним из %v",
		"email":    "%s неправильный Email адрес",
		"url":      "%s неправильный URL адрес",
		"length":   "%s должен быть в пределах %v - %v",
	},
	EN: {
		"required": "%s is required",
		"in":       "%s must be one of %v",
		"email":    "%s is not a valid email address",
		"url":      "%s is not a valid URL",
		"length":   "%s must be between %v and %v",
	},
}

// Message returns a localized message template by key.
func Message(lang Lang, key string) string {
	if msgs, ok := messages[lang]; ok {
		if msg, ok := msgs[key]; ok {
			return msg
		}
	}
	return messages[EN][key]
}
