//go:build darwin && cgo

package auth

import (
	"errors"
	"fmt"

	"github.com/keybase/go-keychain"
)

// osSecrets uses the Security framework directly, so the Keychain grants
// access to this binary. (Going through /usr/bin/security, as go-keyring
// does, makes the Keychain trust that tool for every program that runs it.)
type osSecrets struct{}

func (osSecrets) Get(service, account string) (string, error) {
	data, err := keychain.GetGenericPassword(service, account, "", "")
	if err != nil {
		return "", err
	}
	if data == nil {
		return "", errSecretNotFound
	}
	return string(data), nil
}

// Set updates the item in place, or adds it if it does not exist. The
// Keychain lets only an item's creator delete it or change its access list,
// so an item created by another program (or by an older unsigned build)
// keeps its owner; updating it asks the user for access instead.
func (osSecrets) Set(service, account, secret string) error {
	query := keychain.NewItem()
	query.SetSecClass(keychain.SecClassGenericPassword)
	query.SetService(service)
	query.SetAccount(account)
	update := keychain.NewItem()
	update.SetData([]byte(secret))

	err := keychain.UpdateItem(query, update)
	if !errors.Is(err, keychain.ErrorItemNotFound) {
		return err
	}
	item := keychain.NewGenericPassword(service, account, service, []byte(secret), "")
	item.SetSynchronizable(keychain.SynchronizableNo)
	return keychain.AddItem(item)
}

func (osSecrets) Delete(service, account string) error {
	err := keychain.DeleteGenericPasswordItem(service, account)
	switch {
	case errors.Is(err, keychain.ErrorItemNotFound):
		return errSecretNotFound
	case errors.Is(err, keychain.ErrorInvalidOwnerEdit):
		return fmt.Errorf("the Keychain item %q was created by another program, so only it can remove it: "+
			"delete it in the Keychain Access app: %w", service, err)
	}
	return err
}
