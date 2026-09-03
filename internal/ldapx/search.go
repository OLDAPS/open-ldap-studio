package ldapx

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-ldap/ldap/v3"

	"github.com/open-ldap-studio/open-ldap-studio/internal/ldapx/controls"
	"github.com/open-ldap-studio/open-ldap-studio/internal/profiles"
)

// Scope is the search scope.
type Scope int

const (
	ScopeBase Scope = iota
	ScopeOneLevel
	ScopeSubtree
)

func (s Scope) ldapScope() int {
	switch s {
	case ScopeBase:
		return ldap.ScopeBaseObject
	case ScopeOneLevel:
		return ldap.ScopeSingleLevel
	default:
		return ldap.ScopeWholeSubtree
	}
}

// Degradation records a capability the server did not provide, with the reason
// it gave. Every one of these becomes a capability:unavailable event: a
// degradation the user is not told about is the failure SC-016 tests for.
type Degradation struct {
	Capability Capability `json:"capability"`
	Reason     string     `json:"reason"`
}

// SearchRequest is one search, expressed the way the user expressed it.
type SearchRequest struct {
	BaseDN string
	Scope  Scope
	// Filter is a string and is transmitted as written. It is validated by
	// parsing a copy; the original is what is sent (FR-028, Principle III).
	Filter     string
	Attributes []string
	// IncludeOperational requests '+' alongside the named attributes.
	IncludeOperational bool
	Aliases            profiles.AliasPolicy
	SizeLimit          int
	TimeLimit          int
	Page               PageRequest
	Sort               []controls.SortKey
	VLV                *controls.VLVRequest
	// ManageDsaIT makes referral and alias entries visible as entries, which
	// is how they are edited rather than followed.
	ManageDsaIT bool
}

func derefAliases(p profiles.AliasPolicy) int {
	switch p {
	case profiles.AliasSearch:
		return ldap.DerefInSearching
	case profiles.AliasFind:
		return ldap.DerefFindingBaseObj
	case profiles.AliasAlways:
		return ldap.DerefAlways
	default:
		return ldap.NeverDerefAliases
	}
}

// Search runs one page of a search and returns it, with the server's verbatim
// result attached whether it succeeded or not.
//
// A size or time limit hit is not an error here: it is a truncated page,
// labelled as such. Swallowing sizeLimitExceeded as an empty result set is one
// of the prohibited transformations (FR-037, error-model.md).
func Search(ctx context.Context, c *Conn, req SearchRequest) (Page, []Degradation, error) {
	client := c.client()
	if client == nil {
		return Page{}, nil, TransportError("search", errors.New("the connection is not open"))
	}

	attributes := req.Attributes
	if req.IncludeOperational {
		attributes = append(append([]string{}, attributes...), "+")
	}
	if len(attributes) == 0 {
		attributes = []string{"*"}
	}

	var requestControls []ldap.Control
	pageSize := req.Page.Size
	if pageSize > 0 {
		requestControls = append(requestControls, controls.Paging(pageSize, req.Page.Cookie))
	}
	if len(req.Sort) > 0 {
		requestControls = append(requestControls, controls.Sort(req.Sort))
	}
	if req.VLV != nil {
		requestControls = append(requestControls, req.VLV)
	}
	if req.ManageDsaIT {
		requestControls = append(requestControls, controls.ManageDsaIT(true))
	}

	search := ldap.NewSearchRequest(
		req.BaseDN,
		req.Scope.ldapScope(),
		derefAliases(req.Aliases),
		req.SizeLimit,
		req.TimeLimit,
		false,
		req.Filter,
		attributes,
		requestControls,
	)

	type outcome struct {
		res *ldap.SearchResult
		err error
	}
	done := make(chan outcome, 1)
	go func() {
		res, err := client.Search(search)
		done <- outcome{res, err}
	}()

	var result *ldap.SearchResult
	var err error
	select {
	case <-ctx.Done():
		// Abandoning is the server's business; closing the connection is ours.
		// Either way the caller is not left waiting past its cancellation.
		return Page{}, nil, Cancelled("search")
	case o := <-done:
		result, err = o.res, o.err
	}

	page := Page{Result: resultFrom(err)}
	var degradations []Degradation

	if err != nil {
		serverResult := ResultOf(translateError("search", err))
		if serverResult == nil || !serverResult.Truncated() {
			return page, degradations, translateError("search", err)
		}
		// A truncated page carries entries and a result code. Both are shown.
		page.Result = *serverResult
		page.TruncatedByServer = true
		page.ServerLimit = req.SizeLimit
	}

	if result != nil {
		page.Entries = make([]Entry, 0, len(result.Entries))
		for _, e := range result.Entries {
			page.Entries = append(page.Entries, convertEntry(e))
		}
		page.LoadedCount = len(page.Entries)
		page.Result.Referrals = append(page.Result.Referrals, result.Referrals...)

		if pageSize > 0 {
			cookie, present := controls.PagingCookie(result.Controls)
			switch {
			case present:
				page.Cookie = cookie
			default:
				// The server ignored the paging control. Saying nothing here
				// would leave the UI offering a "next page" that can never
				// arrive.
				degradations = append(degradations, Degradation{
					Capability: CapPaging,
					Reason:     "the server returned no paged results control, so results are not paged",
				})
			}
		}
		if len(req.Sort) > 0 {
			if code, present := controls.SortResult(result.Controls); !present || code != 0 {
				reason := "the server returned no sort response control"
				if present {
					reason = fmt.Sprintf("the server refused the sort with result code %d", code)
				}
				degradations = append(degradations, Degradation{Capability: CapSorting, Reason: reason})
			}
		}
		if req.VLV != nil {
			if _, present := controls.FindVLVResponse(result.Controls); !present {
				degradations = append(degradations, Degradation{
					Capability: CapVLV,
					Reason:     "the server returned no virtual list view response control",
				})
			}
		}
	}

	return page, degradations, nil
}

// AttrHasSubordinates and AttrNumSubordinates are the operational attributes a
// directory uses to say whether an entry has children.
//
// hasSubordinates is X.511 and widely implemented; numSubordinates is an
// OpenLDAP extension and is treated as a bonus. Neither is required: an entry
// that carries neither stays ChildrenUnknown, which the tree draws as
// expandable-but-unproven rather than guessing either way.
const (
	AttrHasSubordinates = "hasSubordinates"
	AttrNumSubordinates = "numSubordinates"
)

// convertEntry maps a go-ldap entry onto ours, keeping byte values and the
// attribute description exactly as received.
func convertEntry(e *ldap.Entry) Entry {
	entry := Entry{DN: e.DN, HasChildren: childrenOf(e), Attributes: make([]Attribute, 0, len(e.Attributes))}
	for _, a := range e.Attributes {
		attrType, options := ParseDescription(a.Name)
		values := a.ByteValues
		if values == nil {
			values = make([][]byte, 0, len(a.Values))
			for _, v := range a.Values {
				values = append(values, []byte(v))
			}
		}
		entry.Attributes = append(entry.Attributes, Attribute{
			Type:    attrType,
			Options: options,
			Values:  values,
		})
	}
	return entry
}

// childrenOf reads the server's answer to "does this entry have children".
//
// It is a report, not an inference: nothing here counts, searches, or assumes.
// An entry the server said nothing about is unknown, and the UI says so rather
// than drawing a leaf that might not be one (data-model §2).
func childrenOf(e *ldap.Entry) Children {
	for _, a := range e.Attributes {
		attrType, _ := ParseDescription(a.Name)
		switch {
		case strings.EqualFold(attrType, AttrHasSubordinates):
			if len(a.Values) == 0 {
				continue
			}
			if strings.EqualFold(a.Values[0], "TRUE") {
				return ChildrenYes
			}
			if strings.EqualFold(a.Values[0], "FALSE") {
				return ChildrenNo
			}
		case strings.EqualFold(attrType, AttrNumSubordinates):
			if len(a.Values) == 0 {
				continue
			}
			n, err := strconv.Atoi(strings.TrimSpace(a.Values[0]))
			if err != nil {
				continue
			}
			if n > 0 {
				return ChildrenYes
			}
			return ChildrenNo
		}
	}
	return ChildrenUnknown
}

// ReadOptions selects what a single-entry read returns.
type ReadOptions struct {
	// IncludeOperational adds '+' to the requested attributes (FR-027).
	IncludeOperational bool
	Attributes         []string
	ManageDsaIT        bool
}

// ReadEntry reads one entry by DN, base-scoped.
func ReadEntry(ctx context.Context, c *Conn, dn string, opts ReadOptions) (Entry, Result, error) {
	page, _, err := Search(ctx, c, SearchRequest{
		BaseDN:             dn,
		Scope:              ScopeBase,
		Filter:             "(objectClass=*)",
		Attributes:         opts.Attributes,
		IncludeOperational: opts.IncludeOperational,
		ManageDsaIT:        opts.ManageDsaIT,
	})
	if err != nil {
		return Entry{}, page.Result, err
	}
	if len(page.Entries) == 0 {
		return Entry{}, page.Result, ServerError("read",
			NewResult(NoSuchObject, "", "the server returned no entry for that DN"))
	}
	return page.Entries[0], page.Result, nil
}

// ListChildren fetches one page of a container's immediate children.
//
// Child count is deliberately not requested here: counting 100,000 children to
// draw an expander is how a browser becomes unusable on a large container
// (FR-018).
func ListChildren(ctx context.Context, c *Conn, dn string, page PageRequest) (Page, []Degradation, error) {
	size := page.Size
	if size == 0 {
		size = c.Profile().Limits.PageSize
	}
	return Search(ctx, c, SearchRequest{
		BaseDN: dn,
		Scope:  ScopeOneLevel,
		Filter: "(objectClass=*)",
		// The subordinate hints are requested by name so the tree can tell a
		// leaf from an unopened container without a speculative search per
		// row. A server that does not have them simply returns neither, and
		// the rows stay ChildrenUnknown.
		Attributes:         []string{"*", AttrHasSubordinates, AttrNumSubordinates},
		IncludeOperational: page.IncludeOperational,
		Aliases:            c.Profile().Aliases,
		SizeLimit:          c.Profile().Limits.SizeLimit,
		TimeLimit:          c.Profile().Limits.TimeLimit,
		Page:               PageRequest{Size: size, Cookie: page.Cookie},
	})
}

// FilterDiagnostic is the answer to "is this filter valid", and nothing more.
//
// There is no method that returns a rewritten filter, because there is no
// circumstance in which the client should decide what the user meant to type
// (FR-019, FR-028).
type FilterDiagnostic struct {
	OK bool `json:"ok"`
	// Position is a byte offset into the filter, -1 when unknown.
	Position int    `json:"position"`
	Message  string `json:"message"`
}

// ValidateFilter parses a copy of the filter and reports what it found. It
// never contacts a server, and the string it was given is unchanged.
func ValidateFilter(filter string) FilterDiagnostic {
	if strings.TrimSpace(filter) == "" {
		return FilterDiagnostic{OK: false, Position: 0, Message: "the filter is empty"}
	}

	copied := strings.Clone(filter)
	if _, err := ldap.CompileFilter(copied); err != nil {
		return FilterDiagnostic{OK: false, Position: filterErrorPosition(err), Message: err.Error()}
	}
	return FilterDiagnostic{OK: true, Position: -1}
}

// filterErrorPosition digs a byte offset out of go-ldap's parse error where it
// carries one, so the editor can put the caret on the problem.
func filterErrorPosition(err error) int {
	var ldapErr *ldap.Error
	if errors.As(err, &ldapErr) {
		var position int
		if _, scanErr := fmt.Sscanf(ldapErr.Err.Error(), "ldap: unexpected end of filter at position %d", &position); scanErr == nil {
			return position
		}
	}
	return -1
}
