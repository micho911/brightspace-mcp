//go:build !darwin

package auth

import (
	"errors"

	"github.com/zalando/go-keyring"
)

// osSecrets uses the platform credential store through go-keyring (Secret
// Service on Linux, Credential Manager on Windows).
type osSecrets struct{}

func (osSecrets) Get(service, account string) (string, error) {
	secret, err := keyring.Get(service, account)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", errSecretNotFound
	}
	return secret, err
}

func (osSecrets) Set(service, account, secret string) error {
	return keyring.Set(service, account, secret)
}

func (osSecrets) Delete(service, account string) error {
	err := keyring.Delete(service, account)
	if errors.Is(err, keyring.ErrNotFound) {
		return errSecretNotFound
	}
	return err
}
