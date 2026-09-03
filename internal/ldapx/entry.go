package ldapx

import "strings"

// Children is a tri-state: a container whose child count is not yet known must
// not be drawn as a leaf, and must not be drawn as expandable when it is not
// (data-model §2).
type Children string

const (
	ChildrenYes     Children = "yes"
	ChildrenNo      Children = "no"
	ChildrenUnknown Children = "unknown"
)

// Attribute is one attribute description and its values.
//
// Values are bytes, not strings. Nothing converts them outside a presentation
// layer that can round-trip back to the same bytes — SC-007 depends on it, and
// so does every export path (data-model §2, the model's most consequential
// invariant).
type Attribute struct {
	// Type is the base attribute description, as received.
	Type string `json:"type"`
	// Options are the attribute's options: binary, language tags. Values
	// differing only by option are distinct attributes (FR-045).
	Options []string `json:"options,omitempty"`
	Values  [][]byte `json:"values"`
	// IsOperational comes from the schema where one is readable. Where it is
	// not, the attribute is still shown — an unknown classification is not a
	// reason to hide data (FR-027).
	IsOperational bool `json:"isOperational"`
}

// Description renders the attribute description, options included, exactly as
// it appears in LDIF: cn, userPassword;binary, description;lang-de.
func (a Attribute) Description() string {
	if len(a.Options) == 0 {
		return a.Type
	}
	return a.Type + ";" + strings.Join(a.Options, ";")
}

// ParseDescription splits an attribute description into its type and options.
// It performs no normalisation: the type is returned exactly as written.
func ParseDescription(description string) (attrType string, options []string) {
	parts := strings.Split(description, ";")
	if len(parts) == 1 {
		return parts[0], nil
	}
	return parts[0], parts[1:]
}

// Entry is one directory entry.
type Entry struct {
	// DN is exactly as returned by the server. It is never re-encoded or
	// normalised (FR-019, Principle III).
	DN string `json:"dn"`
	// Attributes are in the order received.
	Attributes  []Attribute `json:"attributes"`
	HasChildren Children    `json:"hasChildren"`
}

// Attribute returns the named attribute. Matching is case-insensitive on the
// type, as LDAP attribute descriptions are, and exact on the options.
func (e Entry) Attribute(description string) (Attribute, bool) {
	wantType, wantOptions := ParseDescription(description)
	for _, a := range e.Attributes {
		if !strings.EqualFold(a.Type, wantType) {
			continue
		}
		if len(a.Options) != len(wantOptions) {
			continue
		}
		match := true
		for i := range wantOptions {
			if !strings.EqualFold(a.Options[i], wantOptions[i]) {
				match = false
				break
			}
		}
		if match {
			return a, true
		}
	}
	return Attribute{}, false
}

// Page is one page of search or child results.
//
// ServerLimit and TruncatedByServer exist so the UI can state a truncation
// rather than silently showing a short list (FR-018, FR-037).
type Page struct {
	Entries []Entry `json:"entries"`
	// Cookie continues a paged result. Empty means there is no next page.
	Cookie []byte `json:"cookie,omitempty"`
	// LoadedCount is how many entries have been fetched so far across pages.
	LoadedCount int `json:"loadedCount"`
	// ServerLimit is the limit the server reported enforcing, 0 if none.
	ServerLimit int `json:"serverLimit"`
	// TruncatedByServer is set when the server stopped on a size or time limit.
	TruncatedByServer bool   `json:"truncatedByServer"`
	Result            Result `json:"result"`
}

// PageRequest asks for one page of children or results.
type PageRequest struct {
	// Size of 0 uses the profile's page size, then the server's default.
	Size int `json:"size"`
	// Cookie continues a previous page.
	Cookie []byte `json:"cookie,omitempty"`
	// IncludeOperational requests operational attributes explicitly with '+'.
	// They are never stripped from a response (FR-027).
	IncludeOperational bool `json:"includeOperational"`
}
