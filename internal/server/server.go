// Package server exposes Brightspace to AI assistants over the Model Context Protocol.
package server

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Name identifies this server to MCP clients.
const Name = "brightspace-mcp"

// New returns an MCP server with all tools registered.
func New(version string) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: Name, Version: version}, nil)
	addServerInfo(s, version)
	return s
}
