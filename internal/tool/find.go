package tool

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/KaribuLab/linux-mcp/internal/policy"
	"github.com/KaribuLab/linux-mcp/internal/toolmeta"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// FindToolDescription is the MCP tool description (agent-facing response contract).
const FindToolDescription = `Search a directory tree for entries by metadata only: name/iname (glob against basename), type (f=file, d=dir, l=symlink), and maxDepth/minDepth. No -exec/-execdir/-ok/-delete/-fprint*-style predicate exists; nothing runs a command or writes/deletes anything. Never follows symlinks while descending, and applies the same path denylist as cat/list to every visited node (a denied node is skipped, not fatal). Choose which table columns come back with showPath/showType/showSize/showModTime (bool, default true for all four); if all four are false the response still returns the Path column alone. On success the first line is metadata: [find path=... matches=returned/total truncated=bool visited=...] followed by a markdown table with only the requested columns. truncated=true when the match cap or the walk's node budget is hit. On a denied root path returns a single line [blocked class=... path=...] with no table.`

type FindFilesArgs struct {
	Path        string `json:"path" jsonschema:"absolute or relative root path to search"`
	Name        string `json:"name,omitempty" jsonschema:"glob pattern matched against the base name (case-sensitive), e.g. *.go"`
	IName       string `json:"iname,omitempty" jsonschema:"glob pattern matched against the base name (case-insensitive); used instead of name when both are set"`
	Type        string `json:"type,omitempty" jsonschema:"filter by entry type: f (regular file), d (directory), l (symlink)"`
	MaxDepth    int    `json:"maxDepth,omitempty" jsonschema:"maximum depth to descend relative to path, root is depth 0 (0 or omitted = unlimited)"`
	MinDepth    int    `json:"minDepth,omitempty" jsonschema:"minimum depth for a match to be reported, root is depth 0 (0 or omitted = include the root itself)"`
	ShowPath    *bool  `json:"showPath,omitempty" jsonschema:"include the Path column in the result table (default true)"`
	ShowType    *bool  `json:"showType,omitempty" jsonschema:"include the Type column in the result table (default true)"`
	ShowSize    *bool  `json:"showSize,omitempty" jsonschema:"include the Size column in the result table (default true)"`
	ShowModTime *bool  `json:"showModTime,omitempty" jsonschema:"include the ModTime column in the result table (default true)"`
}

type findMatch struct {
	Path    string
	Type    string
	Size    int64
	ModTime time.Time
}

type findColumns struct {
	path    bool
	typ     bool
	size    bool
	modTime bool
}

func boolOrDefaultTrue(v *bool) bool {
	return v == nil || *v
}

func resolveFindColumns(args FindFilesArgs) findColumns {
	cols := findColumns{
		path:    boolOrDefaultTrue(args.ShowPath),
		typ:     boolOrDefaultTrue(args.ShowType),
		size:    boolOrDefaultTrue(args.ShowSize),
		modTime: boolOrDefaultTrue(args.ShowModTime),
	}
	if !cols.path && !cols.typ && !cols.size && !cols.modTime {
		// Never return an empty table: fall back to Path alone.
		cols.path = true
	}
	return cols
}

func (c findColumns) header() (string, string) {
	var names []string
	if c.path {
		names = append(names, "Path")
	}
	if c.typ {
		names = append(names, "Type")
	}
	if c.size {
		names = append(names, "Size")
	}
	if c.modTime {
		names = append(names, "ModTime")
	}
	dashes := make([]string, len(names))
	for i := range dashes {
		dashes[i] = "---"
	}
	return "|" + strings.Join(names, "|") + "|\n", "|" + strings.Join(dashes, "|") + "|\n"
}

func (c findColumns) row(m findMatch) string {
	var b strings.Builder
	b.WriteByte('|')
	if c.path {
		b.WriteString(m.Path)
		b.WriteByte('|')
	}
	if c.typ {
		b.WriteString(m.Type)
		b.WriteByte('|')
	}
	if c.size {
		b.WriteString(strconv.FormatInt(m.Size, 10))
		b.WriteByte('|')
	}
	if c.modTime {
		b.WriteString(m.ModTime.Format(time.RFC3339))
		b.WriteByte('|')
	}
	b.WriteByte('\n')
	return b.String()
}

func typeLetter(mode os.FileMode) string {
	switch {
	case mode&os.ModeSymlink != 0:
		return "l"
	case mode.IsDir():
		return "d"
	case mode.IsRegular():
		return "f"
	case mode&os.ModeNamedPipe != 0:
		return "p"
	case mode&os.ModeSocket != 0:
		return "s"
	case mode&os.ModeDevice != 0:
		if mode&os.ModeCharDevice != 0 {
			return "c"
		}
		return "b"
	default:
		return "?"
	}
}

func matchesName(args FindFilesArgs, name string) bool {
	pattern := args.IName
	caseInsensitive := pattern != ""
	if pattern == "" {
		pattern = args.Name
	}
	if pattern == "" {
		return true
	}
	candidate := name
	if caseInsensitive {
		pattern = strings.ToLower(pattern)
		candidate = strings.ToLower(candidate)
	}
	matched, err := filepath.Match(pattern, candidate)
	if err != nil {
		return false
	}
	return matched
}

func matchesType(typeFilter string, mode os.FileMode) bool {
	switch typeFilter {
	case "":
		return true
	case "f":
		return mode.IsRegular()
	case "d":
		return mode.IsDir()
	case "l":
		return mode&os.ModeSymlink != 0
	default:
		return false
	}
}

func FindFiles(ctx context.Context, req *mcp.CallToolRequest, args FindFilesArgs) (*mcp.CallToolResult, any, error) {
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

	cols := resolveFindColumns(args)

	var matches []findMatch
	visited, walkTruncated, err := policy.Walk(abs, policy.WalkLimits{}, args.MinDepth, args.MaxDepth, func(path string, info os.FileInfo, depth int) error {
		if !matchesName(args, info.Name()) {
			return nil
		}
		if !matchesType(args.Type, info.Mode()) {
			return nil
		}
		matches = append(matches, findMatch{
			Path:    path,
			Type:    typeLetter(info.Mode()),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	total := len(matches)
	limit := total
	truncated := walkTruncated
	if total > policy.MaxFindMatches {
		limit = policy.MaxFindMatches
		truncated = true
	}

	headerRow, sepRow := cols.header()
	var body strings.Builder
	body.WriteString(headerRow)
	body.WriteString(sepRow)
	for i := 0; i < limit; i++ {
		body.WriteString(cols.row(matches[i]))
	}

	header := toolmeta.FindHeader{
		Path:      abs,
		Returned:  limit,
		Total:     total,
		Truncated: truncated,
		Visited:   visited,
	}
	text := toolmeta.Render(header, &body)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}, nil, nil
}

func AddFindFilesTool(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "find",
		Description: FindToolDescription,
	}, FindFiles)
}
