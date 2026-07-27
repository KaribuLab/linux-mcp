package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/KaribuLab/linux-mcp/internal/handler"
)

func main() {
	handler := handler.NewHandler()

	// MCP Inspector runs in the browser and needs CORS + OPTIONS preflight.
	if err := http.ListenAndServe("localhost:5000", handler); err != nil {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}
}
