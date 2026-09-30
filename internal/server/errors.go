package server

import (
	"errors"
	"fmt"

	"github.com/micho911/brightspace-mcp/internal/auth"
	"github.com/micho911/brightspace-mcp/internal/brightspace"
)

// sessionError turns a missing or expired session into an instruction the
// assistant can pass on to the user. Other errors are returned unchanged.
func sessionError(err error, baseURL string) error {
	switch {
	case errors.Is(err, auth.ErrNoSession):
		return errors.New("not logged in to Brightspace: ask the user to run `brightspace-mcp login <their Brightspace URL>` in a terminal, then try again")
	case errors.Is(err, brightspace.ErrSessionExpired):
		return fmt.Errorf("the Brightspace session has expired: ask the user to run `brightspace-mcp login %s` in a terminal, then try again", baseURL)
	}
	return err
}
