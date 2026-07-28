// Package command wires the linux-mcp CLI.
package command

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// version is injected at link time with -ldflags "-X <pkg>.version=<tag>".
var version = "dev"

// DefaultSocketPath is the rendezvous point between serve and auth.
const DefaultSocketPath = "/run/linux-mcp/issue.sock"

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "linux-mcp",
		Short: "Linux filesystem MCP server",
		Long: "linux-mcp exposes read-oriented Linux filesystem tools to MCP clients.\n" +
			"Run the server with 'serve' and obtain an access token with 'auth'.",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	// Shared by serve (which creates it) and auth (which connects to it).
	root.PersistentFlags().String("socket", DefaultSocketPath, "path of the token issuance unix socket")
	root.AddCommand(newServeCmd(), newAuthCmd())
	return root
}

// Execute runs the CLI and returns the process exit code.
func Execute() int {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	return 0
}
