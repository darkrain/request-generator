package module

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/darkrain/request-generator/locale"
)

// LoadTranslationsFile reads a nested JSON file, flattens it to dot-separated keys,
// and stores translations for the given locale.
//
// JSON structure:
//
//	{
//	  "users": {
//	    "label": "Users",
//	    "fields": { "email": "Email", "phone": "Phone" },
//	    "options": { "role": { "admin": "Admin" } },
//	    "actions": { "list": "User list" }
//	  }
//	}
//
// Produces keys: "users.label", "users.fields.email", "users.options.role.admin", etc.
func (g *Generator) LoadTranslationsFile(lang locale.Lang, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read translations file %s: %w", path, err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parse translations file %s: %w", path, err)
	}

	flat := make(map[string]string)
	flattenJSON(raw, "", flat)

	if g.translations == nil {
		g.translations = make(map[locale.Lang]map[string]string)
	}
	g.translations[lang] = flat
	return nil
}

func flattenJSON(data map[string]interface{}, prefix string, result map[string]string) {
	for key, value := range data {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}
		switch v := value.(type) {
		case string:
			result[fullKey] = v
		case map[string]interface{}:
			flattenJSON(v, fullKey, result)
		}
	}
}

// Translate resolves a translation key for the given locale.
// Returns the translated text, or the key itself if not found.
func (g *Generator) Translate(lang locale.Lang, key string) string {
	if g.translations != nil {
		if langMap, ok := g.translations[lang]; ok {
			if text, ok := langMap[key]; ok {
				return text
			}
		}
	}
	return key
}
