package server

import (
	"context"
	"runtime"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ServerInfo describes the running server, for troubleshooting.
type ServerInfo struct {
	Version   string `json:"version" jsonschema:"version of brightspace-mcp"`
	GoVersion string `json:"goVersion" jsonschema:"Go runtime version"`
	Platform  string `json:"platform" jsonschema:"operating system and architecture"`
}

func addServerInfo(s *mcp.Server, version string) {
	info := ServerInfo{
		Version:   version,
		GoVersion: runtime.Version(),
		Platform:  runtime.GOOS + "/" + runtime.GOARCH,
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_server_info",
		Description: "Report the brightspace-mcp version and platform. Use this when troubleshooting the server itself; it does not contact Brightspace.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(context.Context, *mcp.CallToolRequest, struct{}) (*mcp.CallToolResult, ServerInfo, error) {
		return nil, info, nil
	})
}
