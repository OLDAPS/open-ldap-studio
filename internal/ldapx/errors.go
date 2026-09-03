package ldapx

import (
	"errors"
	"fmt"
)

// Category is the taxonomy every user-visible failure falls into, exactly
// once. The category determines the affordance the UI offers, which is why it
// exists beyond tidiness (error-model.md § Error categories).
type Category string

const (
	// CategoryServer — the directory returned a non-zero result code. Carries a
	// Result. The operation is definitively not applied.
	CategoryServer Category = "serverError"
	// CategoryAuth — bind rejected, credential expired, ticket missing, account
	// locked. Never retried in a loop, and never silently downgraded to
	// anonymous (deviation D7, contract X8).
	CategoryAuth Category = "authError"
	// CategoryTransport — dial failure, DNS, TLS negotiation, connection reset.
	CategoryTransport Category = "transportError"
	// CategoryTrust — certificate validation failed. The connection stays
	// refused until the user decides (FR-007).
	CategoryTrust Category = "trustError"
	// CategoryCredentialStore — agent locked, unavailable, absent, or the secret
	// is gone. Recoverable: falls through to the session-only path (FR-005).
	CategoryCredentialStore Category = "credentialStoreError"
	// CategoryLocalValidation — filter syntax, LDIF parse, schema violation
	// caught before dispatch. Nothing was sent to the server.
	CategoryLocalValidation Category = "localValidationError"
	// CategoryCapability — the server does not support a required control or
	// operation. Degrade with the reason stated (SC-016).
	CategoryCapability Category = "capabilityError"
	// CategoryIndeterminate — the connection was lost after a write was sent.
	// Neither success nor failure, because it is genuinely unknowable from the
	// client, and pretending otherwise is how a directory gets double-modified
	// (FR-089, contract X4).
	CategoryIndeterminate Category = "indeterminate"
	// CategoryCancelled — the user cancelled. Not an error state: report what
	// was and was not done.
	CategoryCancelled Category = "cancelled"
)

// Error is a categorised failure that may carry the server's verbatim Result.
type Error struct {
	Category Category
	// Result is populated for CategoryServer and CategoryAuth, and sometimes
	// for CategoryCapability. A nil Result means the failure never reached the
	// server, which is itself information the UI states.
	Result *Result
	// Op names what was attempted, in the user's terms ("bind", "search").
	Op string
	// Detail is our own text. It never overwrites Result.DiagnosticMessage.
	Detail string
	Err    error
}

func (e *Error) Error() string {
	switch {
	case e.Result != nil && e.Detail != "":
		return fmt.Sprintf("%s: %s (%s)", e.Op, e.Result.String(), e.Detail)
	case e.Result != nil:
		return fmt.Sprintf("%s: %s", e.Op, e.Result.String())
	case e.Detail != "":
		return fmt.Sprintf("%s: %s", e.Op, e.Detail)
	default:
		return fmt.Sprintf("%s: %s", e.Op, e.Category)
	}
}

func (e *Error) Unwrap() error { return e.Err }

// Is lets callers match on category alone: errors.Is(err, ldapx.ErrIndeterminate).
func (e *Error) Is(target error) bool {
	var sentinel *Error
	if errors.As(target, &sentinel) {
		return sentinel.Category == e.Category && sentinel.Op == "" && sentinel.Result == nil
	}
	return false
}

// Category sentinels for errors.Is.
var (
	ErrServer          = &Error{Category: CategoryServer}
	ErrAuth            = &Error{Category: CategoryAuth}
	ErrTransport       = &Error{Category: CategoryTransport}
	ErrTrust           = &Error{Category: CategoryTrust}
	ErrCredentialStore = &Error{Category: CategoryCredentialStore}
	ErrLocalValidation = &Error{Category: CategoryLocalValidation}
	ErrCapability      = &Error{Category: CategoryCapability}
	ErrIndeterminate   = &Error{Category: CategoryIndeterminate}
	ErrCancelled       = &Error{Category: CategoryCancelled}
)

// ServerError wraps a non-zero Result. The Result travels with the error so no
// call site has to choose between reporting a code and reporting a failure.
func ServerError(op string, r Result) *Error {
	category := CategoryServer
	if r.Code == InvalidCredentials || r.Code == 8 /* strongerAuthRequired */ || r.Code == 48 {
		category = CategoryAuth
	}
	if r.Code == UnavailableCriticalExtension {
		category = CategoryCapability
	}
	return &Error{Category: category, Result: &r, Op: op}
}

// Indeterminate marks a write whose outcome cannot be known: the request was
// sent and the connection died before an answer arrived. The caller must
// re-read the entry; the history records it as indeterminate, never as a
// failure (FR-089).
func Indeterminate(op string, cause error) *Error {
	return &Error{
		Category: CategoryIndeterminate,
		Op:       op,
		Detail:   "the connection was lost after the request was sent; the server may or may not have applied it",
		Err:      cause,
	}
}

// TransportError reports a failure that never reached the directory.
func TransportError(op string, cause error) *Error {
	return &Error{Category: CategoryTransport, Op: op, Detail: cause.Error(), Err: cause}
}

// TrustError reports a certificate the trust store does not vouch for.
func TrustError(op, reason string, cause error) *Error {
	return &Error{Category: CategoryTrust, Op: op, Detail: reason, Err: cause}
}

// CapabilityError reports a control or extended operation the server lacks.
// The reason is mandatory: every degradation is announced (SC-016).
func CapabilityError(op, capability, reason string) *Error {
	return &Error{
		Category: CategoryCapability,
		Op:       op,
		Detail:   fmt.Sprintf("%s unavailable: %s", capability, reason),
	}
}

// LocalValidationError reports a failure caught before anything was sent.
func LocalValidationError(op, detail string) *Error {
	return &Error{Category: CategoryLocalValidation, Op: op, Detail: detail}
}

// Cancelled reports user cancellation, which is not an error state.
func Cancelled(op string) *Error {
	return &Error{Category: CategoryCancelled, Op: op, Detail: "cancelled"}
}

// CategoryOf returns the category of err, or "" if it is not an *Error.
func CategoryOf(err error) Category {
	var e *Error
	if errors.As(err, &e) {
		return e.Category
	}
	return ""
}

// ResultOf returns the verbatim Result carried by err, or nil if the failure
// never reached the server.
func ResultOf(err error) *Result {
	var e *Error
	if errors.As(err, &e) {
		return e.Result
	}
	return nil
}
