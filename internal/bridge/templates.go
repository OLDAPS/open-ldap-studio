package bridge

import "errors"

// Template represents an entry template.
type Template struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	DN   string `json:"dn"`
}

// ListTemplates returns entry templates.
func (b *Bridge) ListTemplates(profileID string) ([]Template, error) {
	return []Template{}, nil
}

// SaveTemplate saves an entry template.
func (b *Bridge) SaveTemplate(profileID string, tmpl Template) (Template, error) {
	return tmpl, errors.New("not implemented")
}
