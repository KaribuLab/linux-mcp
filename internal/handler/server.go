package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/KaribuLab/linux-mcp/internal/token"
	"github.com/KaribuLab/linux-mcp/internal/tool"
	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Config assembles the MCP endpoint. Verifier and Addr are required: without a
// verifier there is no way to authenticate callers, and without the listen
// address the Host header cannot be validated.
type Config struct {
	Verifier *token.Signer
	// Addr is the address serve listens on, used to validate the Host header.
	Addr string
	// Origins is the browser origin allowlist. The zero value allows none.
	Origins Origins
	Logger  *slog.Logger
}

func (c Config) logger() *slog.Logger {
	if c.Logger != nil {
		return c.Logger
	}
	return slog.Default()
}

func NewHandler(cfg Config) (http.Handler, error) {
	if cfg.Verifier == nil {
		return nil, errors.New("handler requires a token verifier")
	}
	host, err := newHostCheck(cfg.Addr)
	if err != nil {
		return nil, err
	}

	server := mcp.NewServer(&mcp.Implementation{Name: "linux-mcp"}, nil)

	tool.AddCatFileTool(server)
	tool.AddListFilesTool(server)
	tool.AddFindFilesTool(server)
	tool.AddGrepTool(server)
	tool.AddListGrepTool(server)
	tool.AddFindGrepTool(server)
	tool.AddPsTool(server)
	tool.AddPsGrepTool(server)
	tool.AddSsTool(server)
	tool.AddSsGrepTool(server)
	server.AddReceivingMiddleware(auditMiddleware(cfg.logger()))

	mcpHandler := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
		return server
	}, nil)

	requireToken := auth.RequireBearerToken(tokenVerifier(cfg.Verifier), bearerOptions())

	// Host first: a rebinding attempt must be stopped even on a preflight.
	return withHostCheck(
		withCORS(requireToken(mcpHandler), cfg.Origins, cfg.logger()),
		host, cfg.logger(),
	), nil
}

// bearerOptions are the requirements every request must satisfy.
//
// ResourceMetadataURL is left empty on purpose: this server has no OAuth
// authorization server to discover, and advertising one would send clients on
// a pointless round trip. Tokens come from the linux-mcp auth command.
func bearerOptions() *auth.RequireBearerTokenOptions {
	return &auth.RequireBearerTokenOptions{Scopes: []string{token.ScopeRead}}
}
