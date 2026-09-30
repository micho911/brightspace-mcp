package auth

import (
	"errors"
	"reflect"
	"testing"
)

// memSecrets is an in-memory secretStore.
type memSecrets map[string]string

func (m memSecrets) Get(service, account string) (string, error) {
	s, ok := m[service+"/"+account]
	if !ok {
		return "", errSecretNotFound
	}
	return s, nil
}

func (m memSecrets) Set(service, account, secret string) error {
	m[service+"/"+account] = secret
	return nil
}

func (m memSecrets) Delete(service, account string) error {
	if _, ok := m[service+"/"+account]; !ok {
		return errSecretNotFound
	}
	delete(m, service+"/"+account)
	return nil
}

// useMemSecrets replaces the OS credential store for the test.
func useMemSecrets(t *testing.T) memSecrets {
	t.Helper()
	m := memSecrets{}
	old := secrets
	secrets = m
	t.Cleanup(func() { secrets = old })
	return m
}

func TestStoreRoundTrip(t *testing.T) {
	useMemSecrets(t)

	if _, err := Load(); !errors.Is(err, ErrNoSession) {
		t.Fatalf("Load before Save: err = %v, want ErrNoSession", err)
	}

	want := Session{
		BaseURL: "https://brightspace.example.edu",
		Cookies: []Cookie{{Name: "d2lSessionVal", Value: "secret"}},
	}
	if err := Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Load = %+v, want %+v", got, want)
	}

	if err := Delete(); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := Load(); !errors.Is(err, ErrNoSession) {
		t.Errorf("Load after Delete: err = %v, want ErrNoSession", err)
	}
	if err := Delete(); err != nil {
		t.Errorf("Delete twice: %v", err)
	}
}

func TestLoadUnreadableSession(t *testing.T) {
	m := useMemSecrets(t)
	m[keyringService+"/"+keyringAccount] = "not json"

	if _, err := Load(); !errors.Is(err, ErrNoSession) {
		t.Errorf("Load: err = %v, want ErrNoSession", err)
	}
}
