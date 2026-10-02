// Package server exposes Brightspace to AI assistants over the Model Context Protocol.
package server

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/micho911/brightspace-mcp/internal/brightspace"
)

// Name identifies this server to MCP clients.
const Name = "brightspace-mcp"

// Connect returns a Brightspace client for the logged-in user, or
// auth.ErrNoSession. It is called for every tool call, so logging in again
// takes effect without restarting the server.
type Connect func() (*brightspace.Client, error)

// New returns an MCP server with all tools registered.
func New(version string, connect Connect) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: Name, Version: version}, nil)
	addServerInfo(s, version)
	addWhoAmI(s, connect)
	addListCourses(s, connect)
	addListAnnouncements(s, connect)
	addListActivityFeed(s, connect)
	addListUpcoming(s, connect)
	addListAssignments(s, connect)
	addGetAssignment(s, connect)
	addGetCourseContent(s, connect)
	addGetContentTopic(s, connect)
	addReadCourseFile(s, connect)
	return s
}
