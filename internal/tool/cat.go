package tool

import (
	"context"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type CatFileArgs struct {
	Path string `json:"path"`
}

func CatFile(ctx context.Context, req *mcp.CallToolRequest, args CatFileArgs) (*mcp.CallToolResult, any, error) {
	content, err := os.ReadFile(args.Path)
	if err != nil {
		return nil, nil, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(content)},
		},
	}, nil, nil
}

func AddCatFileTool(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "cat",
		Description: "Read the contents of a file",
	}, CatFile)
}
