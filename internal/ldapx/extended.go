package ldapx

import (
	"context"

	"github.com/go-ldap/ldap/v3"

	"github.com/open-ldap-studio/open-ldap-studio/internal/secrets"
)

// WhoAmI asks the server which identity it believes is bound (RFC 4532).
//
// It answers a question no local state can: after a SASL bind, a proxied
// authorisation, or a reconnect, the DN the client typed is not necessarily
// the identity the server is applying (FR-011).
func WhoAmI(ctx context.Context, c *Conn) (string, Result, error) {
	var identity string
	err := c.readOperation(ctx, "whoami", func(client *ldap.Conn) error {
		res, err := client.WhoAmI(nil)
		if err == nil {
			identity = res.AuthzID
		}
		return err
	})
	result := resultFrom(err)
	if err != nil {
		return "", result, translateError("whoami", err)
	}
	return identity, result, nil
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
	req := ldap.NewPasswordModifyRequest(dn, string(oldSecret.Bytes()), string(newSecret.Bytes()))
	var generated string
	result, err := c.write(ctx, "passwordModify", func(client *ldap.Conn) error {
		res, err := client.PasswordModify(req)
		if err == nil {
			generated = res.GeneratedPassword
		}
		return err
	})
	return generated, result, err
}

// Compare performs an LDAP compare, whose two success codes — compareTrue (6)
// and compareFalse (5) — are both returned as results rather than as a boolean
// with the code discarded.
func Compare(ctx context.Context, c *Conn, dn, attribute string, value []byte) (bool, Result, error) {
	var matched bool
	err := c.readOperation(ctx, "compare", func(client *ldap.Conn) error {
		var err error
		matched, err = client.Compare(dn, attribute, string(value))
		return err
	})
	if err != nil {
		result := resultFrom(err)
		// compareFalse is an answer, not a failure.
		if result.Code == 5 {
			return false, result, nil
		}
		return false, result, translateError("compare", err)
	}
	code := 5 // compareFalse
	if matched {
		code = 6 // compareTrue
	}
	return matched, NewResult(code, "", ""), nil
}
