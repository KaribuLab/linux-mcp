package tool

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/KaribuLab/linux-mcp/internal/policy"
	"github.com/KaribuLab/linux-mcp/internal/toolmeta"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// FindGrepToolDescription is the MCP tool description (agent-facing response contract).
const FindGrepToolDescription = `Find entries by metadata then search their file contents for a pattern (like find | xargs grep). Discovery predicates match find: name/iname (glob on basename), type (f/d/l), maxDepth/minDepth. No find show* columns — output is grep-style, not a find table. pattern defaults to literal text (extended=false); extended=true uses RE2; ignoreCase applies to either mode. Only readable regular files among find matches are content-scanned; directories and non-regular entries are not grepped. Binary files are skipped silently during the multi-file scan; private-key content is searched but matching row content is replaced with "[private-key content redacted]" and counted in redacted. On success: [find_grep path=... matches=returned/total truncated=bool filesScanned=... redacted=... visited=...] followed by raw path:line:content rows. truncated=true when the match cap or walk node budget is hit. Denied root path returns [blocked class=... path=...] with no rows.`

type FindGrepArgs struct {
	Path       string `json:"path" jsonschema:"absolute or relative root path to search"`
	Pattern    string `json:"pattern" jsonschema:"literal text (default) or RE2 regular expression (extended=true) to search for in file contents"`
	Extended   bool   `json:"extended,omitempty" jsonschema:"if true, compile pattern as an RE2 regular expression instead of matching it as literal text"`
	IgnoreCase bool   `json:"ignoreCase,omitempty" jsonschema:"case-insensitive match"`
	Name       string `json:"name,omitempty" jsonschema:"glob pattern matched against the base name (case-sensitive), e.g. *.go"`
	IName      string `json:"iname,omitempty" jsonschema:"glob pattern matched against the base name (case-insensitive); used instead of name when both are set"`
	Type       string `json:"type,omitempty" jsonschema:"filter by entry type: f (regular file), d (directory), l (symlink)"`
	MaxDepth   int    `json:"maxDepth,omitempty" jsonschema:"maximum depth to descend relative to path, root is depth 0 (0 or omitted = unlimited)"`
	MinDepth   int    `json:"minDepth,omitempty" jsonschema:"minimum depth for a match to be reported, root is depth 0 (0 or omitted = include the root itself)"`
}

func (a FindGrepArgs) findArgs() FindFilesArgs {
	return FindFilesArgs{
		Path:     a.Path,
		Name:     a.Name,
		IName:    a.IName,
		Type:     a.Type,
		MaxDepth: a.MaxDepth,
		MinDepth: a.MinDepth,
	}
}

func FindGrep(ctx context.Context, req *mcp.CallToolRequest, args FindGrepArgs) (*mcp.CallToolResult, any, error) {
	abs, err := policy.CheckPath(args.Path)
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

	re, err := compileGrepPattern(args.Pattern, args.Extended, args.IgnoreCase)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid pattern: %w", err)
	}

	findPred := args.findArgs()
	var (
		matches      []string
		totalMatches int
		redacted     int
		filesScanned int
	)

	visited, walkTruncated, err := policy.Walk(abs, policy.WalkLimits{}, args.MinDepth, args.MaxDepth, func(path string, info os.FileInfo, depth int) error {
		if !matchesName(findPred, info.Name()) {
			return nil
		}
		if !matchesType(args.Type, info.Mode()) {
			return nil
		}
		// Content search only for regular files (find | xargs grep over dirs is a no-op).
		if !info.Mode().IsRegular() {
			return nil
		}
		scanned, _, scanErr := grepScan(path, re, &matches, &totalMatches, &redacted)
		if scanErr != nil {
			return nil
		}
		if scanned {
			filesScanned++
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	truncated := walkTruncated || totalMatches > len(matches)

	var body strings.Builder
	for _, row := range matches {
		body.WriteString(row)
	}

	header := toolmeta.FindGrepHeader{
		Path:         abs,
		Returned:     len(matches),
		Total:        totalMatches,
		Truncated:    truncated,
		FilesScanned: filesScanned,
		Redacted:     redacted,
		Visited:      visited,
	}
	text := toolmeta.Render(header, &body)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}, nil, nil
}

func AddFindGrepTool(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "find_grep",
		Description: FindGrepToolDescription,
	}, FindGrep)
}
