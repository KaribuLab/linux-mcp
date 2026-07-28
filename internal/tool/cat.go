package tool

import (
	"context"
	"errors"

	"github.com/KaribuLab/linux-mcp/internal/policy"
	"github.com/KaribuLab/linux-mcp/internal/toolmeta"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// CatToolDescription is the MCP tool description (agent-facing response contract).
const CatToolDescription = `Read a text file with bounded output (max 100 lines and 64KiB). On success the first line is metadata: [cat path=... lines=... truncated=bool next=<byte-or-empty>] followed by the raw text body (not markdown). Pass next as offset to resume via Seek when truncated. On policy/content block returns a single line [blocked class=... path=...] with no file body (classes include path_denied, private_key, binary, type_denied). Does not dump full large or binary files; no server-side file cache.`

type CatFileArgs struct {
	Path   string `json:"path" jsonschema:"absolute or relative path of the file to read"`
	Offset int64  `json:"offset,omitempty" jsonschema:"optional byte cursor to resume from (use next from a prior truncated response)"`
}

func CatFile(ctx context.Context, req *mcp.CallToolRequest, args CatFileArgs) (*mcp.CallToolResult, any, error) {
	page, err := policy.ReadPage(args.Path, args.Offset, policy.Limits{})
	if err != nil {
		var d *policy.Denied
		if errors.As(err, &d) {
			text := toolmeta.Blocked{Class: string(d.Class), Path: d.Path}.String()
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: text}},
				IsError: true,
			}, nil, nil
		}
		return nil, nil, err
	}

	header := toolmeta.CatHeader{
		Path:      page.Path,
		Lines:     page.Lines,
		Truncated: page.Truncated,
		NextByte:  page.NextByte,
	}
	text := toolmeta.Render(header, &page.Body)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}, nil, nil
}

func AddCatFileTool(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "cat",
		Description: CatToolDescription,
	}, CatFile)
}
