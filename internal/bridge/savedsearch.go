package bridge

import (
	"errors"

	"github.com/open-ldap-studio/open-ldap-studio/internal/history"
	"github.com/open-ldap-studio/open-ldap-studio/internal/ldapx"
)

type SavedSearch struct {
	ID   string                 `json:"id"`
	Name string                 `json:"name"`
	Def  ldapx.SearchDefinition `json:"def"`
}

// ListSavedSearches returns saved searches.
func (b *Bridge) ListSavedSearches(profileID string) ([]SavedSearch, error) {
	return []SavedSearch{}, nil
}

// SaveSearch saves a search.
func (b *Bridge) SaveSearch(profileID string, search SavedSearch) (SavedSearch, error) {
	return search, errors.New("not implemented")
}

// DeleteSavedSearch deletes a saved search.
func (b *Bridge) DeleteSavedSearch(id string) error {
	return errors.New("not implemented")
}

// SearchHistory returns the recent search history.
func (b *Bridge) SearchHistory() ([]history.SearchRecord, error) {
	return []history.SearchRecord{}, nil
}
