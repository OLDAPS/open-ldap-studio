package ldapx

import (
	"context"
	"errors"

	"github.com/go-ldap/ldap/v3"

	"github.com/open-ldap-studio/open-ldap-studio/internal/ldapx/controls"
)

// The mutation dispatch functions.
//
// These have exactly one calling package, internal/changeset, and contract
// test C1 asserts it. Go has no way to say "package-private to one friend", so
// the boundary is enforced by a test rather than by the compiler — which is
// why the test is written before any of this and blocks merge.
//
// Every function here returns the server's verbatim Result on success and on
// failure, and reports Indeterminate when the connection dies after the
// request went out (FR-089, contract X4).

// AddEntry creates an entry.
func AddEntry(ctx context.Context, c *Conn, dn string, attributes []Attribute) (Result, error) {
	req := ldap.NewAddRequest(dn, nil)
	for _, a := range attributes {
		values := make([]string, 0, len(a.Values))
		for _, v := range a.Values {
			values = append(values, string(v))
		}
		req.Attribute(a.Description(), values)
	}

	return c.write(ctx, "add", func(client *ldap.Conn) error { return client.Add(req) })
}

// ModifyEntry applies attribute-level changes to an entry.
type Change struct {
	// Operation is add, delete or replace, matching RFC 4511's modify types.
	Operation string
	Attribute Attribute
}

const (
	ChangeAdd     = "add"
	ChangeDelete  = "delete"
	ChangeReplace = "replace"
)

// ModifyEntry applies changes to an existing entry.
func ModifyEntry(ctx context.Context, c *Conn, dn string, changes []Change) (Result, error) {
	req := ldap.NewModifyRequest(dn, nil)
	for _, ch := range changes {
		values := make([]string, 0, len(ch.Attribute.Values))
		for _, v := range ch.Attribute.Values {
			values = append(values, string(v))
		}
		switch ch.Operation {
		case ChangeAdd:
			req.Add(ch.Attribute.Description(), values)
		case ChangeDelete:
			req.Delete(ch.Attribute.Description(), values)
		case ChangeReplace:
			req.Replace(ch.Attribute.Description(), values)
		default:
			return Result{}, LocalValidationError("modify",
				"unknown modification type "+ch.Operation)
		}
	}

	return c.write(ctx, "modify", func(client *ldap.Conn) error { return client.Modify(req) })
}

// RenameEntry performs a modify DN.
//
// KeepOldRDN is passed through exactly as chosen: whether the old RDN value
// stays on the entry is a decision with data consequences, and it is never
// defaulted silently (FR-048).
func RenameEntry(ctx context.Context, c *Conn, dn, newRDN, newSuperior string, keepOldRDN bool) (Result, error) {
	req := ldap.NewModifyDNRequest(dn, newRDN, !keepOldRDN, newSuperior)
	return c.write(ctx, "modifyDN", func(client *ldap.Conn) error { return client.ModifyDN(req) })
}

// DeleteEntry deletes a leaf entry.
//
// A server answering notAllowedOnNonLeaf (66) is reported verbatim; the
// subtree delete is offered as a distinct, separately confirmed action rather
// than retried automatically (FR-046).
func DeleteEntry(ctx context.Context, c *Conn, dn string) (Result, error) {
	return c.write(ctx, "delete", func(client *ldap.Conn) error { return client.Del(ldap.NewDelRequest(dn, nil)) })
}

// DeleteSubtree deletes an entry and everything beneath it using the tree
// delete control, where the server supports it.
//
// The caller must have counted the subtree and obtained a distinct
// confirmation naming that count before reaching here (FR-046).
func DeleteSubtree(ctx context.Context, c *Conn, dn string) (Result, error) {
	req := ldap.NewDelRequest(dn, []ldap.Control{controls.SubtreeDelete()})
	return c.write(ctx, "subtreeDelete", func(client *ldap.Conn) error { return client.Del(req) })
}

// finishWrite turns a write's outcome into a Result, distinguishing the case
// that matters most: a request that was sent and never answered.
func finishWrite(ctx context.Context, op string, err error) (Result, error) {
	if err == nil {
		return NewResult(Success, "", ""), nil
	}

	result := resultFrom(err)
	if ctx.Err() != nil && (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)) {
		// send was already entered. Cancellation cannot establish whether the
		// server applied the request before the transport closed.
		return result, Indeterminate(op, err)
	}

	var ldapErr *ldap.Error
	if errors.As(err, &ldapErr) && ldapErr.ResultCode == ldap.ErrorNetwork {
		// The request went out and the socket died. Whether the server applied
		// it is genuinely unknowable from here, and reporting either success
		// or failure is how a directory gets double-modified.
		return result, Indeterminate(op, err)
	}
	return result, translateError(op, err)
}
