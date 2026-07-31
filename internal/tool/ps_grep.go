package tool

import (
	"context"
	"fmt"

	"github.com/KaribuLab/linux-mcp/internal/policy"
	"github.com/KaribuLab/linux-mcp/internal/toolmeta"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// PsGrepToolDescription is the MCP tool description (agent-facing response contract).
const PsGrepToolDescription = `List processes via /proc and filter markdown ROWS by a pattern (like ps | grep), without running host binaries. Same show* / includeKernel args as ps. Pattern modes match list_grep: glob against Comm when extended=false and pattern contains * ? [; otherwise literal substring or RE2 (extended=true) on the full data row. On success: [ps_grep entries=returned/total truncated=bool columns=...] + markdown table of matches only. truncated=true if the underlying inventory hit the 1000-row cap or filtered matches exceed the output cap.`

type PsGrepArgs struct {
	Pattern       string `json:"pattern" jsonschema:"filter for process rows: glob against Comm when it contains * ? [; otherwise literal substring on the full row; RE2 when extended=true"`
	Extended      bool   `json:"extended,omitempty" jsonschema:"if true, compile pattern as RE2 against the full data row (disables glob auto-detection)"`
	IgnoreCase    bool   `json:"ignoreCase,omitempty" jsonschema:"case-insensitive match"`
	IncludeKernel bool   `json:"includeKernel,omitempty" jsonschema:"if true, include kernel threads (default false)"`
	ShowPpid      *bool  `json:"showPpid,omitempty" jsonschema:"include Ppid column (default true)"`
	ShowUser      *bool  `json:"showUser,omitempty" jsonschema:"include User column (default true)"`
	ShowStat      *bool  `json:"showStat,omitempty" jsonschema:"include Stat column (default true)"`
	ShowCpu       *bool  `json:"showCpu,omitempty" jsonschema:"include Cpu column (default true)"`
	ShowMem       *bool  `json:"showMem,omitempty" jsonschema:"include Mem column as RSS KiB (default true)"`
	ShowCmdline   *bool  `json:"showCmdline,omitempty" jsonschema:"include Cmdline column (default true)"`
	ShowExe       *bool  `json:"showExe,omitempty" jsonschema:"include Exe column (default true)"`
}

func (a PsGrepArgs) psArgs() PsArgs {
	return PsArgs{
		IncludeKernel: a.IncludeKernel,
		ShowPpid:      a.ShowPpid,
		ShowUser:      a.ShowUser,
		ShowStat:      a.ShowStat,
		ShowCpu:       a.ShowCpu,
		ShowMem:       a.ShowMem,
		ShowCmdline:   a.ShowCmdline,
		ShowExe:       a.ShowExe,
	}
}

func PsGrep(ctx context.Context, req *mcp.CallToolRequest, args PsGrepArgs) (*mcp.CallToolResult, any, error) {
	re, err := compileRowFilter(args.Pattern, args.Extended, args.IgnoreCase)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid pattern: %w", err)
	}

	tbl, err := buildPsTable(args.psArgs())
	if err != nil {
		return nil, nil, err
	}

	matched := make([]string, 0, len(tbl.dataRows))
	for _, row := range tbl.dataRows {
		ok, matchErr := matchTableRow(row, args.Pattern, args.Extended, args.IgnoreCase, re, 2) // Comm
		if matchErr != nil {
			return nil, nil, fmt.Errorf("invalid glob pattern: %w", matchErr)
		}
		if ok {
			matched = append(matched, row)
		}
	}

	totalMatched := len(matched)
	limit := totalMatched
	outTrunc := false
	if totalMatched > policy.MaxListEntries {
		limit = policy.MaxListEntries
		outTrunc = true
	}
	tbl.dataRows = matched[:limit]
	meta := toolmeta.PsGrepHeader{
		Returned:  len(tbl.dataRows),
		Total:     totalMatched,
		Truncated: tbl.truncated || outTrunc,
		Columns:   joinColumns(tbl.columns),
	}
	return renderPsTable(meta, tbl), nil, nil
}

func AddPsGrepTool(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "ps_grep",
		Description: PsGrepToolDescription,
	}, PsGrep)
}
