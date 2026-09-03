package bridge

import "errors"

// GetPreferences returns application preferences.
func (b *Bridge) GetPreferences() (map[string]any, error) {
	return map[string]any{}, nil
}

// SetPreferences saves application preferences.
func (b *Bridge) SetPreferences(prefs map[string]any) error {
	return errors.New("not implemented")
}

// PurgeCache purges application cache.
func (b *Bridge) PurgeCache() error {
	return errors.New("not implemented")
}
