package ldapx

import (
	"context"
	"slices"
)

// RootDSE is what a server says about itself: naming contexts, the controls
// and extensions it supports, and the SASL mechanisms it offers.
//
// It is read once per connection and is what every capability decision is made
// against, so that a degradation can be announced before an operation fails
// rather than after (FR-012, SC-016).
type RootDSE struct {
	NamingContexts       []string `json:"namingContexts"`
	SupportedControl     []string `json:"supportedControl"`
	SupportedExtension   []string `json:"supportedExtension"`
	SupportedSASLMechs   []string `json:"supportedSaslMechanisms"`
	SupportedLDAPVersion []string `json:"supportedLdapVersion"`
	SubschemaSubentry    string   `json:"subschemaSubentry"`
	VendorName           string   `json:"vendorName,omitempty"`
	VendorVersion        string   `json:"vendorVersion,omitempty"`
	// ConfigContext is where OpenLDAP keeps cn=config; its presence is what
	// makes configuration-as-entries editing possible.
	ConfigContext string `json:"configContext,omitempty"`
	// Raw keeps every attribute the server returned, including the ones this
	// struct does not name. Nothing is dropped.
	Raw map[string][]string `json:"raw"`
}

// Control OIDs whose presence in the Root DSE decides a capability.
const (
	oidPagedResults  = "1.2.840.113556.1.4.319"
	oidSortRequest   = "1.2.840.113556.1.4.473"
	oidVLVRequest    = "2.16.840.1.113730.3.4.9"
	oidSubtreeDelete = "1.2.840.113556.1.4.805"
	oidManageDsaIT   = "2.16.840.1.113730.3.4.2"

	oidStartTLS       = "1.3.6.1.4.1.1466.20037"
	oidWhoAmI         = "1.3.6.1.4.1.4203.1.11.3"
	oidPasswordModify = "1.3.6.1.4.1.4203.1.11.1"
)

// ReadRootDSE fetches the Root DSE. It is requested with both '*' and '+'
// because several servers publish their operational attributes only on the
// explicit request.
func ReadRootDSE(ctx context.Context, c *Conn) (RootDSE, Result, error) {
	page, _, err := Search(ctx, c, SearchRequest{
		BaseDN:             "",
		Scope:              ScopeBase,
		Filter:             "(objectClass=*)",
		Attributes:         []string{"*"},
		IncludeOperational: true,
	})
	if err != nil {
		return RootDSE{}, page.Result, err
	}
	if len(page.Entries) == 0 {
		return RootDSE{}, page.Result, ServerError("rootDSE",
			NewResult(Other, "", "the server returned no Root DSE"))
	}

	entry := page.Entries[0]
	dse := RootDSE{Raw: make(map[string][]string, len(entry.Attributes))}
	for _, a := range entry.Attributes {
		values := make([]string, 0, len(a.Values))
		for _, v := range a.Values {
			values = append(values, string(v))
		}
		dse.Raw[a.Description()] = values

		switch lower(a.Type) {
		case "namingcontexts":
			dse.NamingContexts = values
		case "supportedcontrol":
			dse.SupportedControl = values
		case "supportedextension":
			dse.SupportedExtension = values
		case "supportedsaslmechanisms":
			dse.SupportedSASLMechs = values
		case "supportedldapversion":
			dse.SupportedLDAPVersion = values
		case "subschemasubentry":
			if len(values) > 0 {
				dse.SubschemaSubentry = values[0]
			}
		case "vendorname":
			if len(values) > 0 {
				dse.VendorName = values[0]
			}
		case "vendorversion":
			if len(values) > 0 {
				dse.VendorVersion = values[0]
			}
		case "configcontext":
			if len(values) > 0 {
				dse.ConfigContext = values[0]
			}
		}
	}
	return dse, page.Result, nil
}

// SupportsControl reports whether the server advertises a control OID.
func (d RootDSE) SupportsControl(oid string) bool { return slices.Contains(d.SupportedControl, oid) }

// SupportsExtension reports whether the server advertises an extended
// operation OID.
func (d RootDSE) SupportsExtension(oid string) bool {
	return slices.Contains(d.SupportedExtension, oid)
}

// SupportsSASL reports whether the server offers a SASL mechanism.
func (d RootDSE) SupportsSASL(mechanism string) bool {
	return slices.Contains(d.SupportedSASLMechs, mechanism)
}

// MissingCapabilities lists what this server cannot do, with the reason to
// show the user. A server that answers no Root DSE at all is reported as such
// rather than assumed capable.
func (d RootDSE) MissingCapabilities() []Degradation {
	var missing []Degradation
	check := func(capability Capability, oid, what string) {
		if !d.SupportsControl(oid) {
			missing = append(missing, Degradation{
				Capability: capability,
				Reason:     "the server does not advertise " + what + " (" + oid + ")",
			})
		}
	}
	check(CapPaging, oidPagedResults, "the simple paged results control")
	check(CapSorting, oidSortRequest, "the server-side sorting control")
	check(CapVLV, oidVLVRequest, "the virtual list view control")

	if !d.SupportsExtension(oidWhoAmI) && !d.SupportsExtension(oidPasswordModify) {
		missing = append(missing, Degradation{
			Capability: CapExtendedOp,
			Reason:     "the server advertises neither Who Am I nor Password Modify",
		})
	}
	if d.SubschemaSubentry == "" {
		missing = append(missing, Degradation{
			Capability: CapSchema,
			Reason:     "the server publishes no subschemaSubentry, so its schema cannot be read",
		})
	}
	if d.ConfigContext == "" {
		missing = append(missing, Degradation{
			Capability: CapConfigEntries,
			Reason:     "the server publishes no configContext, so its configuration is not editable as entries",
		})
	}
	return missing
}

func lower(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] += 'a' - 'A'
		}
	}
	return string(b)
}
