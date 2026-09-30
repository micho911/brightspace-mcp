package auth

import (
	"errors"
	"reflect"
	"testing"

	"github.com/zalando/go-keyring"
)

func TestStoreRoundTrip(t *testing.T) {
	keyring.MockInit()

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
