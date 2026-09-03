package bridge

import (
	"errors"

	"github.com/open-ldap-studio/open-ldap-studio/internal/ldapx"
)

// StartSearch kicks off a search based on the provided SearchDefinition.
// Returns a job ID.
func (b *Bridge) StartSearch(profileID string, def ldapx.SearchDefinition) (string, error) {
	return "", errors.New("not implemented")
}

// FetchNextPage retrieves the next page of results for a given job.
func (b *Bridge) FetchNextPage(jobID string) (ldapx.Page, error) {
	return ldapx.Page{}, errors.New("not implemented")
}
