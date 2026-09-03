package credentials

import (
	"errors"
)

// Store persists credential metadata (not the secrets themselves).
type Store struct {
}

func (s *Store) Save(c Credential) error {
	return errors.New("not implemented")
}

func (s *Store) Get(id string) (Credential, error) {
	return Credential{}, errors.New("not implemented")
}

func (s *Store) Delete(id string) error {
	return errors.New("not implemented")
}
