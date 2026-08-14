// Package securestore provides a small adapter for the operating system's
// credential store. It is intentionally limited to opaque byte values.
package securestore

import (
	"encoding/base64"
	"errors"

	keyring "github.com/zalando/go-keyring"
)

// ErrNotFound reports that no value exists for an account.
var ErrNotFound = errors.New("securestore: not found")

// Store persists secrets in the credential store of the signed-in OS user.
// Account names must be non-sensitive identifiers.
type Store interface {
	Get(account string) ([]byte, error)
	Set(account string, value []byte) error
	Delete(account string) error
}

type keychainStore struct {
	service string
}

// New returns a store backed by the native credential manager on the current
// platform. It does not access the credential manager until an operation runs.
func New(service string) Store {
	return &keychainStore{service: service}
}

func (s *keychainStore) Get(account string) ([]byte, error) {
	value, err := keyring.Get(s.service, account)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	decoded, err := base64.RawStdEncoding.DecodeString(value)
	if err != nil {
		return nil, err
	}
	return decoded, nil
}

func (s *keychainStore) Set(account string, value []byte) error {
	return keyring.Set(s.service, account, base64.RawStdEncoding.EncodeToString(value))
}

func (s *keychainStore) Delete(account string) error {
	err := keyring.Delete(s.service, account)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}
