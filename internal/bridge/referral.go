package bridge

import (
	"errors"

	"github.com/open-ldap-studio/open-ldap-studio/internal/ldapx"
)

// ResolveReferral looks up a referral target.
func (b *Bridge) ResolveReferral(target string) (ldapx.ReferralTarget, error) {
	return ldapx.ReferralTarget{}, errors.New("not implemented")
}

// FollowReferral follows a referral using a target profile.
func (b *Bridge) FollowReferral(profileID, target string) error {
	return errors.New("not implemented")
}

// ParseLDAPURL parses an LDAP URL.
func (b *Bridge) ParseLDAPURL(url string) (ldapx.ReferralTarget, error) {
	return ldapx.ReferralTarget{}, errors.New("not implemented")
}
