package tool

import (
	"context"
	"fmt"

	"github.com/KaribuLab/linux-mcp/internal/policy"
	"github.com/KaribuLab/linux-mcp/internal/toolmeta"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// SsGrepToolDescription is the MCP tool description (agent-facing response contract).
const SsGrepToolDescription = `List sockets via netlink sock_diag and filter markdown ROWS by a pattern (never runs the ss binary). Same state/family/show* args as ss (including Pid/Process via /proc/*/fd; under the reference systemd unit this resolves foreign process owners). Pattern modes match list_grep: glob against Local when extended=false and pattern contains * ? [; otherwise literal/RE2 on the full data row — prefer specific patterns like 0.0.0.0:3306 (Peer may also contain 0.0.0.0). On success: [ss_grep entries=returned/total truncated=bool columns=...] + matching rows only.`

type SsGrepArgs struct {
	Pattern     string `json:"pattern" jsonschema:"filter for socket rows: glob against Local when it contains * ? [; otherwise literal substring on the full row; RE2 when extended=true"`
	Extended    bool   `json:"extended,omitempty" jsonschema:"if true, compile pattern as RE2 against the full data row"`
	IgnoreCase  bool   `json:"ignoreCase,omitempty" jsonschema:"case-insensitive match"`
	State       string `json:"state,omitempty" jsonschema:"LISTEN (default), ESTAB, or all"`
	Family      string `json:"family,omitempty" jsonschema:"inet (default), inet4, inet6, unix, or all"`
	ShowState   *bool  `json:"showState,omitempty" jsonschema:"include State column (default true)"`
	ShowPeer    *bool  `json:"showPeer,omitempty" jsonschema:"include Peer column (default true)"`
	ShowPid     *bool  `json:"showPid,omitempty" jsonschema:"include Pid column (default true)"`
	ShowProcess *bool  `json:"showProcess,omitempty" jsonschema:"include Process column (default true)"`
	ShowUser    *bool  `json:"showUser,omitempty" jsonschema:"include User column (default true)"`
	ShowFamily  *bool  `json:"showFamily,omitempty" jsonschema:"include Family column (default true)"`
}

func (a SsGrepArgs) ssArgs() SsArgs {
	return SsArgs{
		State:       a.State,
		Family:      a.Family,
		ShowState:   a.ShowState,
		ShowPeer:    a.ShowPeer,
		ShowPid:     a.ShowPid,
		ShowProcess: a.ShowProcess,
		ShowUser:    a.ShowUser,
		ShowFamily:  a.ShowFamily,
	}
}

func SsGrep(ctx context.Context, req *mcp.CallToolRequest, args SsGrepArgs) (*mcp.CallToolResult, any, error) {
	re, err := compileRowFilter(args.Pattern, args.Extended, args.IgnoreCase)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid pattern: %w", err)
	}

	tbl, err := buildSsTable(args.ssArgs())
	if err != nil {
		return nil, nil, err
	}

	matched := make([]string, 0, len(tbl.dataRows))
	for _, row := range tbl.dataRows {
		ok, matchErr := matchTableRow(row, args.Pattern, args.Extended, args.IgnoreCase, re, 2) // Local
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
	meta := toolmeta.SsGrepHeader{
		Returned:  len(tbl.dataRows),
		Total:     totalMatched,
		Truncated: tbl.truncated || outTrunc,
		Columns:   joinColumns(tbl.columns),
	}
	return renderSsTable(meta, tbl), nil, nil
}

func AddSsGrepTool(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "ss_grep",
		Description: SsGrepToolDescription,
	}, SsGrep)
}
