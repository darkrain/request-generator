package renderer

import (
	"fmt"
	"net/url"
	"strings"
	"unicode"
)

// Decoration describes non-interactive, non-semantic imagery. Integration owns
// URI resolution. These sources are not labels and must never be localized.
type Decoration struct {
	Source           string `json:"source,omitempty"`
	BackgroundSource string `json:"background_source,omitempty"`
}

func (d *Decoration) Validate() error {
	if d == nil {
		return nil
	}
	if d.Source == "" && d.BackgroundSource == "" {
		return fmt.Errorf("decoration requires a source")
	}
	for _, source := range []string{d.Source, d.BackgroundSource} {
		if source == "" {
			continue
		}
		if strings.ContainsAny(source, "\\\"'<>`)({}") || strings.IndexFunc(source, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) >= 0 {
			return fmt.Errorf("invalid decoration URI")
		}
		u, err := url.Parse(source)
		if err != nil || strings.HasPrefix(source, "//") {
			return fmt.Errorf("invalid decoration URI")
		}
		switch strings.ToLower(u.Scheme) {
		case "javascript", "vbscript", "data", "file":
			return fmt.Errorf("unsafe decoration URI scheme")
		}
	}
	return nil
}

func cloneDecoration(d *Decoration) *Decoration {
	if d == nil {
		return nil
	}
	result := *d
	return &result
}
