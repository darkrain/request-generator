package renderer

import (
	"fmt"
	"strings"
)

// PageHint is optional explanatory content. Key scopes dismissal in the
// consumer's storage adapter; all display strings belong to the producer.
type PageHint struct {
	Key         string `json:"key,omitempty"`
	Title       string `json:"title,omitempty"`
	Text        string `json:"text,omitempty"`
	Acknowledge string `json:"acknowledge,omitempty"`
	Close       string `json:"close,omitempty"`
	Icon        string `json:"icon,omitempty"`
}

func (h *PageHint) Validate() error {
	if h != nil && (strings.TrimSpace(h.Key) != h.Key || strings.ContainsAny(h.Key, "\r\n\x00")) {
		return fmt.Errorf("page hint key must not contain surrounding whitespace or control characters")
	}
	return nil
}
