//go:build !darwin && !linux && !windows

package auth

import "errors"

func defaultBrowser() (browser, error) {
	return nil, errors.New("login is only supported on macOS for now")
}
