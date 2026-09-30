package server

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// connect starts the server in memory and returns a connected client session.
func connect(t *testing.T, bs Connect) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()
	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	ss, err := New("v0.0.0-test", bs).Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	t.Cleanup(func() { _ = ss.Close() })

	cs, err := mcp.NewClient(&mcp.Implementation{Name: "test-client"}, nil).Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

func TestGetServerInfo(t *testing.T) {
	cs := connect(t, nil)

	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "get_server_info"})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool returned error: %+v", res.Content)
	}

	raw, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatalf("marshal structured content: %v", err)
	}
	var got ServerInfo
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal structured content: %v", err)
	}
	if got.Version != "v0.0.0-test" {
		t.Errorf("Version = %q, want %q", got.Version, "v0.0.0-test")
	}
	if got.GoVersion == "" || got.Platform == "" {
		t.Errorf("missing runtime details: %+v", got)
	}
}
