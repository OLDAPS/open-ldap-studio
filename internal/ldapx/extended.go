package ldapx

import (
	"context"
	"errors"

	"github.com/go-ldap/ldap/v3"

	"github.com/open-ldap-studio/open-ldap-studio/internal/secrets"
)

// WhoAmI asks the server which identity it believes is bound (RFC 4532).
//
// It answers a question no local state can: after a SASL bind, a proxied
// authorisation, or a reconnect, the DN the client typed is not necessarily
// the identity the server is applying (FR-011).
func WhoAmI(ctx context.Context, c *Conn) (string, Result, error) {
	client := c.client()
	if client == nil {
		return "", Result{}, TransportError("whoami", errors.New("the connection is not open"))
	}

	res, err := client.WhoAmI(nil)
	result := resultFrom(err)
	if err != nil {
		return "", result, translateError("whoami", err)
	}
	return res.AuthzID, result, nil
}

// PasswordModify performs the Password Modify extended operation (RFC 3062).
//
// The old and new values are secrets and never appear in a log, an event, or
// the history file: redaction happens at write time, before any of those are
// produced (FR-093).
//
// A server that generates the new password returns it, and it is handed
// straight back to the caller for display once — it is never stored.
func PasswordModify(ctx context.Context, c *Conn, dn string, oldSecret, newSecret secrets.Secret) (string, Result, error) {
	client := c.client()
	if client == nil {
		return "", Result{}, TransportError("passwordModify", errors.New("the connection is not open"))
	}

	req := ldap.NewPasswordModifyRequest(dn, string(oldSecret.Bytes()), string(newSecret.Bytes()))
	res, err := client.PasswordModify(req)
	result := resultFrom(err)
	if err != nil {
		return "", result, translateError("passwordModify", err)
	}
	return res.GeneratedPassword, result, nil
}

// Compare performs an LDAP compare, whose two success codes — compareTrue (6)
// and compareFalse (5) — are both returned as results rather than as a boolean
// with the code discarded.
func Compare(ctx context.Context, c *Conn, dn, attribute string, value []byte) (bool, Result, error) {
	client := c.client()
	if client == nil {
		return false, Result{}, TransportError("compare", errors.New("the connection is not open"))
	}

	matched, err := client.Compare(dn, attribute, string(value))
	if err != nil {
		result := resultFrom(err)
		// compareFalse is an answer, not a failure.
		if result.Code == 5 {
			return false, result, nil
		}
		return false, result, translateError("compare", err)
	}
	return matched, NewResult(6, "", ""), nil
}
