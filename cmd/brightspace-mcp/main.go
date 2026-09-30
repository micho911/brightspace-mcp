// Command brightspace-mcp is an MCP server for D2L Brightspace.
//
// Run without arguments, it serves MCP over stdin/stdout. Stdout belongs to
// the protocol in that mode, so diagnostics go to stderr.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/micho911/brightspace-mcp/internal/server"
	"github.com/micho911/brightspace-mcp/internal/version"
)

const usage = `Usage: brightspace-mcp [command]

Commands:
  serve         Serve MCP over stdin/stdout (default)
  login <url>   Take the Brightspace session from your browser and save it
  logout        Remove the saved session
  version       Print the version
`

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "brightspace-mcp: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdout io.Writer) error {
	cmd := "serve"
	if len(args) > 0 {
		cmd = args[0]
	}

	switch cmd {
	case "serve":
		return server.New(version.String()).Run(ctx, &mcp.StdioTransport{})
	case "login":
		return login(ctx, args[1:], stdout)
	case "logout":
		return logout(stdout)
	case "version", "--version":
		_, err := fmt.Fprintln(stdout, version.String())
		return err
	case "help", "--help", "-h":
		_, err := fmt.Fprint(stdout, usage)
		return err
	default:
		return fmt.Errorf("unknown command %q\n\n%s", cmd, usage)
	}
}
