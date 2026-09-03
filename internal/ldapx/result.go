package ldapx

import "fmt"

// Result is the server's own answer, carried verbatim.
//
// It is returned on success as well as failure. Nothing between this package
// and the bridge may collapse it to a bool or a bare error: SC-006 measures
// "the server's own words survive to the user" at 100%, and contract X2
// asserts no such collapse exists (error-model.md § The one rule).
type Result struct {
	// Code is always populated. 0 (success) is a value, not an absence.
	Code int `json:"code"`
	// MatchedDN is verbatim as received.
	MatchedDN string `json:"matchedDn"`
	// DiagnosticMessage is verbatim, byte-for-byte as received. It is never
	// trimmed, re-cased, re-wrapped, or replaced with a friendlier message.
	DiagnosticMessage string `json:"diagnosticMessage"`
	// Interpretation is ADDITIVE plain-language text, rendered alongside
	// DiagnosticMessage and never instead of it.
	Interpretation string    `json:"interpretation"`
	Referrals      []string  `json:"referrals,omitempty"`
	Controls       []Control `json:"controls,omitempty"`
}

// Control is a response control carried back on a Result — paged results
// cookies, sort responses, VLV positions.
type Control struct {
	OID      string `json:"oid"`
	Critical bool   `json:"critical"`
	Value    []byte `json:"value,omitempty"`
}

// Result codes the specification names behaviour for. The rest are surfaced
// generically with their diagnostic (error-model.md § Result codes).
const (
	Success                      = 0
	TimeLimitExceeded            = 3
	SizeLimitExceeded            = 4
	Referral                     = 10
	UnavailableCriticalExtension = 12
	ConstraintViolation          = 19
	AttributeOrValueExists       = 20
	NoSuchObject                 = 32
	InvalidCredentials           = 49
	InsufficientAccessRights     = 50
	UnwillingToPerform           = 53
	NotAllowedOnNonLeaf          = 66
	Other                        = 80
)

// interpretations is the additive plain-language layer. Entries here never
// replace a diagnostic message; they are shown beside it.
var interpretations = map[int]string{
	Success:                      "The server accepted the operation.",
	TimeLimitExceeded:            "The server stopped on its time limit. The results shown are partial.",
	SizeLimitExceeded:            "The server stopped on its size limit. The results shown are partial.",
	Referral:                     "The entry lives on another server. Follow the referral to continue.",
	UnavailableCriticalExtension: "The server does not support a control this operation marked critical.",
	ConstraintViolation:          "A value broke a rule the server enforces, such as a password policy.",
	AttributeOrValueExists:       "That attribute value is already present on the entry.",
	NoSuchObject:                 "No entry exists at that DN on this server.",
	InvalidCredentials:           "The bind was rejected. The DN or the password is not accepted.",
	InsufficientAccessRights:     "The bound identity is not permitted to perform this operation.",
	UnwillingToPerform:           "The server refused the operation, commonly a schema or configuration write.",
	NotAllowedOnNonLeaf:          "The entry has children. Deleting it requires a subtree delete.",
	Other:                        "The server reported an unspecified failure.",
}

// codeNames are the RFC 4511 names, used for display beside the number.
var codeNames = map[int]string{
	Success: "success", 1: "operationsError", 2: "protocolError",
	TimeLimitExceeded: "timeLimitExceeded", SizeLimitExceeded: "sizeLimitExceeded",
	5: "compareFalse", 6: "compareTrue", 7: "authMethodNotSupported",
	8: "strongerAuthRequired", Referral: "referral", 11: "adminLimitExceeded",
	UnavailableCriticalExtension: "unavailableCriticalExtension", 13: "confidentialityRequired",
	14: "saslBindInProgress", 16: "noSuchAttribute", 17: "undefinedAttributeType",
	18: "inappropriateMatching", ConstraintViolation: "constraintViolation",
	AttributeOrValueExists: "attributeOrValueExists", 21: "invalidAttributeSyntax",
	NoSuchObject: "noSuchObject", 33: "aliasProblem", 34: "invalidDNSyntax",
	36: "aliasDereferencingProblem", 48: "inappropriateAuthentication",
	InvalidCredentials: "invalidCredentials", InsufficientAccessRights: "insufficientAccessRights",
	51: "busy", 52: "unavailable", UnwillingToPerform: "unwillingToPerform",
	54: "loopDetect", 64: "namingViolation", 65: "objectClassViolation",
	NotAllowedOnNonLeaf: "notAllowedOnNonLeaf", 67: "notAllowedOnRDN",
	68: "entryAlreadyExists", 69: "objectClassModsProhibited", 71: "affectsMultipleDSAs",
	Other: "other",
}

// NewResult builds a Result, attaching the additive interpretation. The
// diagnostic is stored exactly as passed — callers must hand over the server's
// bytes, not a rendering of them.
func NewResult(code int, matchedDN, diagnostic string) Result {
	return Result{
		Code:              code,
		MatchedDN:         matchedDN,
		DiagnosticMessage: diagnostic,
		Interpretation:    interpretations[code],
	}
}

// CodeName returns the RFC 4511 name for a result code, or "" if unknown. An
// unknown code is still shown as a number; it is never suppressed.
func CodeName(code int) string { return codeNames[code] }

// OK reports whether the server accepted the operation. It is a convenience
// for control flow only — it never replaces a Result on the way to the UI.
func (r Result) OK() bool { return r.Code == Success }

// Truncated reports whether the server cut the result set short. Both codes
// carry partial results that must be labelled, never silently shown as
// complete (FR-037).
func (r Result) Truncated() bool {
	return r.Code == SizeLimitExceeded || r.Code == TimeLimitExceeded
}

// String renders the code, its name, and the server's diagnostic — in that
// order, with nothing dropped.
func (r Result) String() string {
	name := codeNames[r.Code]
	if name == "" {
		name = "unknown"
	}
	if r.DiagnosticMessage == "" {
		return fmt.Sprintf("%d (%s)", r.Code, name)
	}
	return fmt.Sprintf("%d (%s): %s", r.Code, name, r.DiagnosticMessage)
}
