package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/micho911/brightspace-mcp/internal/auth"
	"github.com/micho911/brightspace-mcp/internal/brightspace"
)

// fakeBrightspace serves handler and returns a Connect that talks to it.
func fakeBrightspace(t *testing.T, handler http.HandlerFunc) Connect {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return func() (*brightspace.Client, error) {
		return brightspace.NewClient(srv.URL, nil), nil
	}
}

func callWhoAmI(t *testing.T, bs Connect) *mcp.CallToolResult {
	t.Helper()
	res, err := connect(t, bs).CallTool(context.Background(), &mcp.CallToolParams{Name: "whoami"})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	return res
}

func errorText(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	if !res.IsError || len(res.Content) == 0 {
		t.Fatalf("want a tool error, got %+v", res)
	}
	text, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("error content is %T, want text", res.Content[0])
	}
	return text.Text
}

func TestWhoAmI(t *testing.T) {
	bs := fakeBrightspace(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Identifier":"42","FirstName":"Ada","LastName":"Lovelace","UniqueName":"au123456"}`))
	})

	res := callWhoAmI(t, bs)
	if res.IsError {
		t.Fatalf("tool returned error: %+v", res.Content)
	}
	raw, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatalf("marshal structured content: %v", err)
	}
	var got CurrentUser
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal structured content: %v", err)
	}
	if got.UserID != "42" || got.FirstName != "Ada" || got.LastName != "Lovelace" ||
		got.Username != "au123456" || !strings.HasPrefix(got.Brightspace, "http://127.0.0.1") {
		t.Errorf("whoami = %+v", got)
	}
}

func TestWhoAmINotLoggedIn(t *testing.T) {
	bs := func() (*brightspace.Client, error) { return nil, auth.ErrNoSession }

	if msg := errorText(t, callWhoAmI(t, bs)); !strings.Contains(msg, "brightspace-mcp login") {
		t.Errorf("error %q does not tell the user to log in", msg)
	}
}

func TestWhoAmISessionExpired(t *testing.T) {
	bs := fakeBrightspace(t, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/d2l/login", http.StatusFound)
	})

	msg := errorText(t, callWhoAmI(t, bs))
	if !strings.Contains(msg, "expired") || !strings.Contains(msg, "brightspace-mcp login http://127.0.0.1") {
		t.Errorf("error %q does not tell the user to log in again to the same instance", msg)
	}
}
