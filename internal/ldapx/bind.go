package ldapx

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/go-ldap/ldap/v3/gssapi"

	"github.com/open-ldap-studio/open-ldap-studio/internal/profiles"
	"github.com/open-ldap-studio/open-ldap-studio/internal/secrets"
)

// BindRequest is everything a bind needs. The secret arrives as a
// secrets.Secret, which cannot be serialised or printed, and is zeroed by the
// caller once the bind returns.
type BindRequest struct {
	Method profiles.BindMethod
	// DN is raw text, sent exactly as the user typed it (FR-020).
	DN     string
	Secret secrets.Secret
	// Realm and ServicePrincipal apply to DIGEST-MD5 and GSSAPI.
	Realm            string
	ServicePrincipal string
	// KeytabPath, CCachePath and Krb5ConfPath select a GSSAPI credential
	// source. An empty CCachePath uses the platform default.
	KeytabPath   string
	CCachePath   string
	Krb5ConfPath string
}

// Bind authenticates the connection and returns the server's verbatim result.
//
// Two rules are structural here rather than conventional:
//
//   - A failed bind is never retried, and never falls back to an anonymous
//     bind. A silent privilege downgrade makes a failed bind look like a
//     successful one (deviation D7, contract X8).
//   - CRAM-MD5 runs on the raw socket before go-ldap is started, because
//     go-ldap can neither perform it nor be extended to.
func Bind(ctx context.Context, c *Conn, req BindRequest) (Result, error) {
	if c == nil {
		return Result{}, TransportError("bind", errors.New("no connection"))
	}

	timeout := time.Duration(c.profile.Timeouts.ReadMS) * time.Millisecond
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	// CRAM-MD5 is the one mechanism that must run before the client starts.
	if req.Method == profiles.BindCramMD5 {
		result, err := bindCramMD5(c.rawConn, req.DN, req.Secret, timeout)
		if err != nil {
			return result, err
		}
		if err := c.activate(); err != nil {
			return result, err
		}
		c.setBound(req.DN)
		return result, nil
	}

	if err := c.activate(); err != nil {
		return Result{}, err
	}

	client := c.client()
	var err error

	switch req.Method {
	case profiles.BindAnonymous, "":
		// An anonymous bind is a bind, not the absence of one. It is chosen
		// explicitly and shown in the status bar as such.
		err = client.UnauthenticatedBind("")
	case profiles.BindSimple:
		_, err = client.SimpleBind(&ldap.SimpleBindRequest{
			Username: req.DN,
			Password: string(req.Secret.Bytes()),
		})
	case profiles.BindExternal:
		// SASL EXTERNAL takes the identity from the TLS client certificate,
		// so it is refused on an unencrypted transport rather than silently
		// binding as nobody.
		if !c.Encrypted() {
			return Result{}, LocalValidationError("bind",
				"SASL EXTERNAL requires TLS: the identity comes from the client certificate")
		}
		err = client.ExternalBind()
	case profiles.BindDigestMD5:
		_, err = client.DigestMD5Bind(&ldap.DigestMD5BindRequest{
			Host:     c.profile.Host,
			Username: req.DN,
			Password: string(req.Secret.Bytes()),
		})
	case profiles.BindGSSAPI:
		err = bindGSSAPI(client, c.profile, req)
	default:
		return Result{}, LocalValidationError("bind",
			fmt.Sprintf("unknown bind method %q", req.Method))
	}

	result := resultFrom(err)
	if err != nil {
		return result, translateError("bind", err)
	}
	c.setBound(req.DN)
	return result, nil
}

// bindGSSAPI builds a Kerberos client from whichever credential source the
// profile names, then performs the SASL exchange.
func bindGSSAPI(client *ldap.Conn, p profiles.Profile, req BindRequest) error {
	servicePrincipal := req.ServicePrincipal
	if servicePrincipal == "" {
		servicePrincipal = "ldap/" + p.Host
	}

	var (
		gssClient ldap.GSSAPIClient
		err       error
	)
	switch {
	case req.KeytabPath != "":
		gssClient, err = gssapi.NewClientWithKeytab(req.DN, req.Realm, req.KeytabPath, req.Krb5ConfPath)
	case req.CCachePath != "":
		gssClient, err = gssapi.NewClientFromCCache(req.CCachePath, req.Krb5ConfPath)
	case !req.Secret.IsZero():
		gssClient, err = gssapi.NewClientWithPassword(req.DN, req.Realm, string(req.Secret.Bytes()), req.Krb5ConfPath)
	default:
		return LocalValidationError("bind",
			"GSSAPI needs a credential cache, a keytab, or a password; none was supplied")
	}
	if err != nil {
		return &Error{
			Category: CategoryAuth,
			Op:       "bind",
			Detail:   "the Kerberos credential source could not be opened: " + err.Error(),
			Err:      err,
		}
	}
	defer func() {
		if closer, ok := gssClient.(interface{ Close() error }); ok {
			_ = closer.Close()
		}
	}()

	return client.GSSAPIBind(gssClient, servicePrincipal, "")
}

// setBound records the authenticated identity for the status bar.
func (c *Conn) setBound(dn string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.boundDN = dn
}

// Rebind re-authenticates after a reconnect.
//
// The caller must treat the session as one that requires confirmation before
// its next write: the server may have applied a different policy to the new
// session, so a write queued before the drop is not automatically safe
// (FR-015, conn:reconnected).
func Rebind(ctx context.Context, c *Conn, req BindRequest) (Result, error) {
	return Bind(ctx, c, req)
}
