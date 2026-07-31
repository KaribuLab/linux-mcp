package tool

import (
	"context"
	"fmt"
	"strings"

	"github.com/KaribuLab/linux-mcp/internal/policy"
	"github.com/KaribuLab/linux-mcp/internal/toolmeta"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ListGrepToolDescription is the MCP tool description (agent-facing response contract).
const ListGrepToolDescription = `List a directory and filter listing ROWS by a pattern (like ls | grep). NEVER reads file contents — this is not content search (use grep/find_grep for that). Filters markdown data rows only (table headers are never matches). Pattern modes when extended=false: if pattern contains glob metacharacters * ? or [, it is matched with Go filepath.Match against the entry Name/File column only (so "*.txt" keeps foo.txt); otherwise the pattern is literal substring text against the full data row (so ".txt" or "readme" work). extended=true always compiles RE2 against the full data row (globs are NOT auto-detected — use regex like \.txt$). ignoreCase applies to all modes. Listing flags match list: all, list, show*. On success: [list_grep path=... entries=returned/total truncated=bool] + same markdown table shape as list with only matching rows. truncated=true when the 1000-entry listing window was hit or filtered matches exceed the output cap. Denied path: [blocked class=... path=...] with no table.`

type ListGrepArgs struct {
	Path            string `json:"path" jsonschema:"the path to list the files from"`
	Pattern         string `json:"pattern" jsonschema:"filter for listing rows: glob against Name when it contains * ? [ (e.g. *.txt); otherwise literal substring on the full row; RE2 on the full row when extended=true. Does not search file contents"`
	Extended        bool   `json:"extended,omitempty" jsonschema:"if true, compile pattern as an RE2 regular expression against the full data row (disables glob auto-detection)"`
	IgnoreCase      bool   `json:"ignoreCase,omitempty" jsonschema:"case-insensitive match"`
	All             bool   `json:"all" jsonschema:"whether to list all files, including hidden files"`
	List            bool   `json:"list" jsonschema:"whether to list detailed columns (true) or names only (false)"`
	ShowSize        *bool  `json:"showSize,omitempty" jsonschema:"only used when list=true; whether to include the Size column (default true)"`
	ShowMode        *bool  `json:"showMode,omitempty" jsonschema:"only used when list=true; whether to include the Mode column (default true)"`
	ShowOwner       *bool  `json:"showOwner,omitempty" jsonschema:"only used when list=true; whether to include the Owner column (default true)"`
	ShowGroup       *bool  `json:"showGroup,omitempty" jsonschema:"only used when list=true; whether to include the Group column (default true)"`
	ShowModTime     *bool  `json:"showModTime,omitempty" jsonschema:"only used when list=true; whether to include the ModTime column (default true)"`
	ShowIsDir       *bool  `json:"showIsDir,omitempty" jsonschema:"only used when list=true; whether to include the IsDir column (default true)"`
	ShowIsSymlink   *bool  `json:"showIsSymlink,omitempty" jsonschema:"only used when list=true; whether to include the IsSymlink column (default true)"`
	ShowSymlinkPath *bool  `json:"showSymlinkPath,omitempty" jsonschema:"only used when list=true; whether to include the SymlinkPath column (default true)"`
}

func (a ListGrepArgs) listArgs() ListFilesArgs {
	return ListFilesArgs{
		Path:            a.Path,
		All:             a.All,
		List:            a.List,
		ShowSize:        a.ShowSize,
		ShowMode:        a.ShowMode,
		ShowOwner:       a.ShowOwner,
		ShowGroup:       a.ShowGroup,
		ShowModTime:     a.ShowModTime,
		ShowIsDir:       a.ShowIsDir,
		ShowIsSymlink:   a.ShowIsSymlink,
		ShowSymlinkPath: a.ShowSymlinkPath,
	}
}

// patternLooksLikeGlob reports shell-style glob metacharacters. Used only when
// extended=false so agents can pass "*.txt" and match names, not file contents.
func patternLooksLikeGlob(pattern string) bool {
	return strings.ContainsAny(pattern, "*?[")
}

func ListGrep(ctx context.Context, req *mcp.CallToolRequest, args ListGrepArgs) (*mcp.CallToolResult, any, error) {
	re, err := compileRowFilter(args.Pattern, args.Extended, args.IgnoreCase)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid pattern: %w", err)
	}

	tbl, blocked, err := buildListTable(args.listArgs())
	if err != nil {
		return nil, nil, err
	}
	if blocked != nil {
		return blocked, nil, nil
	}

	matched := make([]string, 0, len(tbl.dataRows))
	for _, row := range tbl.dataRows {
		ok, matchErr := matchTableRow(row, args.Pattern, args.Extended, args.IgnoreCase, re, 1)
		if matchErr != nil {
			return nil, nil, fmt.Errorf("invalid glob pattern: %w", matchErr)
		}
		if ok {
			matched = append(matched, row)
		}
	}

	totalMatched := len(matched)
	limit := totalMatched
	outTruncated := false
	if totalMatched > policy.MaxListEntries {
		limit = policy.MaxListEntries
		outTruncated = true
	}
	truncated := tbl.listTruncated || outTruncated

	var body strings.Builder
	tbl.writeHeader(&body)
	for i := 0; i < limit; i++ {
		body.WriteString(matched[i])
	}

	header := toolmeta.ListGrepHeader{
		Path:      tbl.abs,
		Returned:  limit,
		Total:     totalMatched,
		Truncated: truncated,
	}
	if tbl.listTruncated {
		header.Next = tbl.next
	}
	if tbl.detailed {
		header.Columns = strings.Join(tbl.columns, ",")
	}
	text := toolmeta.Render(header, &body)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}, nil, nil
}

func AddListGrepTool(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_grep",
		Description: ListGrepToolDescription,
	}, ListGrep)
}
