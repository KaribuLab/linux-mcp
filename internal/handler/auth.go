package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/KaribuLab/linux-mcp/internal/token"
	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// jtiKey names the token identifier carried in TokenInfo.Extra, which is how
// the issuance record and the usage records are correlated.
const jtiKey = "jti"

// tokenVerifier adapts the signer to the SDK middleware. UserID is set so the
// transport binds each MCP session to the user that opened it.
func tokenVerifier(signer *token.Signer) auth.TokenVerifier {
	return func(_ context.Context, raw string, _ *http.Request) (*auth.TokenInfo, error) {
		grant, err := signer.Verify(raw)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", auth.ErrInvalidToken, err)
		}
		var scopes []string
		if grant.Scope != "" {
			scopes = []string{grant.Scope}
		}
		return &auth.TokenInfo{
			Scopes:     scopes,
			Expiration: grant.ExpiresAt,
			UserID:     grant.Subject,
			Extra:      map[string]any{jtiKey: grant.ID},
		}, nil
	}
}

// auditMiddleware records who called what. It runs after the bearer middleware,
// so every logged call is an authenticated one.
func auditMiddleware(logger *slog.Logger) mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			attrs := []any{"method", method}
			if name := toolName(req); name != "" {
				attrs = append(attrs, "tool", name)
			}
			if info := tokenInfo(req); info != nil {
				attrs = append(attrs, "sub", info.UserID)
				if id, ok := info.Extra[jtiKey].(string); ok {
					attrs = append(attrs, jtiKey, id)
				}
			}
			logger.Info("mcp call", attrs...)
			return next(ctx, method, req)
		}
	}
}

func tokenInfo(req mcp.Request) *auth.TokenInfo {
	extra := req.GetExtra()
	if extra == nil {
		return nil
	}
	return extra.TokenInfo
}

func toolName(req mcp.Request) string {
	params, ok := req.GetParams().(*mcp.CallToolParamsRaw)
	if !ok {
		return ""
	}
	return params.Name
}
