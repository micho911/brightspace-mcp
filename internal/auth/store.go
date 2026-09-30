package auth

import (
	"encoding/json"
	"errors"
	"fmt"
)

// ErrNoSession means no session is saved; the user has to log in.
var ErrNoSession = errors.New("not logged in")

const (
	keyringService = "brightspace-mcp"
	keyringAccount = "session"
)

// Save stores the session in the OS keychain, replacing any previous one.
func Save(s Session) error {
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	if err := secrets.Set(keyringService, keyringAccount, string(data)); err != nil {
		return fmt.Errorf("save session to keychain: %w", err)
	}
	return nil
}

// Load returns the saved session, or ErrNoSession.
func Load() (Session, error) {
	data, err := secrets.Get(keyringService, keyringAccount)
	if errors.Is(err, errSecretNotFound) {
		return Session{}, ErrNoSession
	}
	if err != nil {
		return Session{}, fmt.Errorf("read session from keychain: %w", err)
	}
	var s Session
	if err := json.Unmarshal([]byte(data), &s); err != nil {
		// Unreadable (e.g. written by an incompatible version): log in again.
		return Session{}, ErrNoSession
	}
	return s, nil
}

// Delete removes the saved session. Deleting a missing session is not an error.
func Delete() error {
	err := secrets.Delete(keyringService, keyringAccount)
	if err != nil && !errors.Is(err, errSecretNotFound) {
		return fmt.Errorf("delete session from keychain: %w", err)
	}
	return nil
}
