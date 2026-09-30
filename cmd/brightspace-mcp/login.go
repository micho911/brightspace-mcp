package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/micho911/brightspace-mcp/internal/auth"
	"github.com/micho911/brightspace-mcp/internal/brightspace"
)

func login(ctx context.Context, args []string, stdout io.Writer) error {
	if len(args) != 1 {
		return errors.New("usage: brightspace-mcp login <brightspace-url>\n\nexample: brightspace-mcp login https://brightspace.au.dk")
	}
	baseURL, err := brightspace.ParseBaseURL(args[0])
	if err != nil {
		return err
	}

	var who brightspace.Identity
	session, err := auth.Login(ctx, baseURL, func(ctx context.Context, cookies []*http.Cookie) error {
		var err error
		who, err = brightspace.NewClient(baseURL, cookies).WhoAmI(ctx)
		return err
	}, stdout)
	if err != nil {
		return err
	}
	if err := auth.Save(session); err != nil {
		return err
	}

	_, err = fmt.Fprintf(stdout, "Logged in to %s as %s %s (%s).\n", baseURL, who.FirstName, who.LastName, who.UniqueName)
	return err
}

func logout(stdout io.Writer) error {
	if err := auth.Delete(); err != nil {
		return err
	}
	_, err := fmt.Fprintln(stdout, "Logged out: the saved Brightspace session was removed from this computer.")
	return err
}
