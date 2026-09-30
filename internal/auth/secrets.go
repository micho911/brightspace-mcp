package auth

import "errors"

// errSecretNotFound means the OS credential store has no such item.
var errSecretNotFound = errors.New("secret not found")

// secretStore is the OS credential store (the Keychain on macOS).
type secretStore interface {
	Get(service, account string) (string, error)
	// Set replaces any existing item.
	Set(service, account, secret string) error
	// Delete removes the item; a missing item is errSecretNotFound.
	Delete(service, account string) error
}

// secrets is the store in use; tests replace it.
var secrets secretStore = osSecrets{}
