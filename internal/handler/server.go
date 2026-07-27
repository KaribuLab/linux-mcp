package handler

import (
	"net/http"
	"slices"

	"github.com/KaribuLab/linux-mcp/internal/tool"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func NewHandler() http.Handler {
	server := mcp.NewServer(&mcp.Implementation{Name: "linux-mcp"}, nil)

	tool.AddCatFileTool(server)
	tool.AddListFilesTool(server)

	return withCORS(mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
		return server
	}, nil))
}

func withCORS(next http.Handler, allowedOrigins ...string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if len(allowedOrigins) > 0 {
			if !slices.Contains(allowedOrigins, origin) {
				w.WriteHeader(http.StatusForbidden)
				return
			}
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept, Mcp-Session-Id, Mcp-Protocol-Version, Last-Event-ID")
		w.Header().Set("Access-Control-Expose-Headers", "Mcp-Session-Id")
		w.Header().Set("Vary", "Origin")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
