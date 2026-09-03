package history

// SearchRecord represents a historical search entry.
type SearchRecord struct {
	ID        string `json:"id"`
	ProfileID string `json:"profileId"`
	Filter    string `json:"filter"`
	BaseDN    string `json:"baseDN"`
	Timestamp int64  `json:"timestamp"`
}

// AddSearch adds a search to history.
func AddSearch(record SearchRecord) error {
	return nil
}
