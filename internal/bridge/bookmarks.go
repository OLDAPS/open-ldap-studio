package bridge

import "errors"

type Bookmark struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	DN   string `json:"dn"`
}

// ListBookmarks returns bookmarks.
func (b *Bridge) ListBookmarks(profileID string) ([]Bookmark, error) {
	return []Bookmark{}, nil
}

// SaveBookmark saves a bookmark.
func (b *Bridge) SaveBookmark(profileID string, bm Bookmark) (Bookmark, error) {
	return bm, errors.New("not implemented")
}

// DeleteBookmark deletes a bookmark.
func (b *Bridge) DeleteBookmark(id string) error {
	return errors.New("not implemented")
}
