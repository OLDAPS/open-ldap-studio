package connections

import (
	"context"

	"github.com/open-ldap-studio/open-ldap-studio/internal/ldapx"
	"github.com/open-ldap-studio/open-ldap-studio/internal/profiles"
)

// Connect supplies the application lifecycle port without exposing a socket.
func (c *Manager) Connect(ctx context.Context, profile profiles.Profile) (ldapx.Result, error) {
	_, err := c.Dial(ctx, profile, nil)
	if result := ldapx.ResultOf(err); result != nil {
		return *result, err
	}
	if err != nil {
		return ldapx.Result{}, err
	}
	return ldapx.NewResult(ldapx.Success, "", ""), nil
}

func (c *Manager) WhoAmI(ctx context.Context, id string) (string, ldapx.Result, error) {
	conn, err := c.Conn(id)
	if err != nil {
		return "", ldapx.Result{}, err
	}
	return ldapx.WhoAmI(ctx, conn)
}

func (c *Manager) RootDSE(ctx context.Context, id string) (ldapx.RootDSE, ldapx.Result, error) {
	conn, err := c.Conn(id)
	if err != nil {
		return ldapx.RootDSE{}, ldapx.Result{}, err
	}
	return ldapx.ReadRootDSE(ctx, conn)
}

func (c *Manager) ReadEntry(ctx context.Context, id, dn string, options ldapx.ReadOptions) (ldapx.Entry, ldapx.Result, error) {
	conn, err := c.Conn(id)
	if err != nil {
		return ldapx.Entry{}, ldapx.Result{}, err
	}
	return ldapx.ReadEntry(ctx, conn, dn, options)
}

func (c *Manager) ListChildren(ctx context.Context, id, dn string, page ldapx.PageRequest) (ldapx.Page, []ldapx.Degradation, error) {
	conn, err := c.Conn(id)
	if err != nil {
		return ldapx.Page{}, nil, err
	}
	return ldapx.ListChildren(ctx, conn, dn, page)
}
