package server

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// CurrentUser is the Brightspace user the server acts as.
type CurrentUser struct {
	UserID      string `json:"userId" jsonschema:"Brightspace user ID"`
	FirstName   string `json:"firstName"`
	LastName    string `json:"lastName"`
	Username    string `json:"username" jsonschema:"login name, e.g. a student number"`
	Brightspace string `json:"brightspace" jsonschema:"address of the Brightspace instance"`
}

func addWhoAmI(s *mcp.Server, connect Connect) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "whoami",
		Description: "Report which Brightspace user and instance this server is logged in as. Use this to check the connection or when the user asks who they are logged in as.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, CurrentUser, error) {
		c, err := connect()
		if err != nil {
			return nil, CurrentUser{}, sessionError(err, "")
		}
		id, err := c.WhoAmI(ctx)
		if err != nil {
			return nil, CurrentUser{}, sessionError(err, c.BaseURL())
		}
		return nil, CurrentUser{
			UserID:      id.Identifier,
			FirstName:   id.FirstName,
			LastName:    id.LastName,
			Username:    id.UniqueName,
			Brightspace: c.BaseURL(),
		}, nil
	})
}
