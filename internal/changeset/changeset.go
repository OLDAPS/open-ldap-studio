package changeset

import (
	"github.com/open-ldap-studio/open-ldap-studio/internal/ldapx"
)

// Kind classifies a pending mutation. SchemaCommit and ConfigModify carry
// extra warnings, because a schema or configuration write can take a directory
// out of service in ways an entry edit cannot (FR-087).
type Kind string

const (
	KindAdd           Kind = "add"
	KindModify        Kind = "modify"
	KindModRDN        Kind = "modRDN"
	KindDelete        Kind = "delete"
	KindSubtreeDelete Kind = "subtreeDelete"
	KindCopy          Kind = "copy"
	KindMove          Kind = "move"
	KindBulkModify    Kind = "bulkModify"
	KindSchemaCommit  Kind = "schemaCommit"
	KindConfigModify  Kind = "configModify"
)

// OpType is one attribute-level or entry-level change.
type OpType string

const (
	OpAddAttr     OpType = "addAttr"
	OpDeleteAttr  OpType = "deleteAttr"
	OpReplaceAttr OpType = "replaceAttr"
	OpAddValue    OpType = "addValue"
	OpDeleteValue OpType = "deleteValue"
	OpAddEntry    OpType = "addEntry"
	OpDeleteEntry OpType = "deleteEntry"
	OpRename      OpType = "rename"
)

// Operation is one change to one entry.
type Operation struct {
	// DN is raw, exactly as the user or the server wrote it.
	DN   string `json:"dn"`
	Type OpType `json:"type"`
	// Attribute carries the description and, for entry-level operations, the
	// full attribute set.
	Attribute ldapx.Attribute `json:"attribute,omitzero"`
	// Attributes is populated for OpAddEntry, which creates an entry whole.
	Attributes []ldapx.Attribute `json:"attributes,omitempty"`
	// Before and After are the attribute-level diff the preview renders. Both
	// are bytes: a diff that stringified its values could not show a change
	// that is only visible in the bytes (FR-039).
	Before [][]byte `json:"before,omitempty"`
	After  [][]byte `json:"after,omitempty"`

	// Rename fields. KeepOldRDN is explicitly chosen, never defaulted (FR-048).
	NewRDN      string `json:"newRdn,omitempty"`
	NewSuperior string `json:"newSuperior,omitempty"`
	KeepOldRDN  bool   `json:"keepOldRdn,omitempty"`
}

// Severity separates a warning the user may override from a condition that
// stops the operation.
type Severity string

const (
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

// Warning is something the user should know before confirming. Schema
// conflicts are warnings the user may override, never silent blocks (FR-063).
type Warning struct {
	Severity Severity `json:"severity"`
	DN       string   `json:"dn,omitempty"`
	Message  string   `json:"message"`
}

// ChangeSet is the only representation of a pending mutation. Nothing else
// reaches ldapx's mutation functions.
type ChangeSet struct {
	ID        string      `json:"id"`
	ProfileID string      `json:"profileId"`
	Kind      Kind        `json:"kind"`
	Ops       []Operation `json:"ops"`
	// AffectedCount is enumerated, never estimated, before a token is issued
	// (FR-039). For a subtree delete that means counting the subtree first.
	AffectedCount int `json:"affectedCount"`
	// BeforeState is captured for reversal. Its absence means the change is
	// not reversible, and the UI states that rather than implying otherwise
	// (FR-092).
	BeforeState []ldapx.Entry `json:"beforeState,omitempty"`
	Warnings    []Warning     `json:"warnings,omitempty"`
}

// Input is what the frontend submits. It is deliberately not ChangeSet: the
// affected count, the before state, and the warnings are computed here, not
// accepted from the caller.
type Input struct {
	ProfileID string      `json:"profileId"`
	Kind      Kind        `json:"kind"`
	Ops       []Operation `json:"ops"`
}

// DNs returns every distinct DN the change set touches, in first-seen order.
func (cs ChangeSet) DNs() []string { //nolint:revive // exported name is part of the bridge contract
	seen := make(map[string]struct{}, len(cs.Ops))
	out := make([]string, 0, len(cs.Ops))
	for _, op := range cs.Ops {
		if _, ok := seen[op.DN]; ok {
			continue
		}
		seen[op.DN] = struct{}{}
		out = append(out, op.DN)
	}
	return out
}

// HasErrors reports whether any warning is severe enough to block.
func (cs ChangeSet) HasErrors() bool {
	for _, w := range cs.Warnings {
		if w.Severity == SeverityError {
			return true
		}
	}
	return false
}
