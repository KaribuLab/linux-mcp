package tool

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/KaribuLab/linux-mcp/internal/policy"
	"github.com/KaribuLab/linux-mcp/internal/toolmeta"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// PsToolDescription is the MCP tool description (agent-facing response contract).
const PsToolDescription = `List processes via /proc only (never runs the ps binary or any host shell). On success the first line is metadata: [ps entries=returned/total truncated=bool columns=...] followed by a markdown table. Identity columns Pid and Comm are always present. Optional boolean show* flags (default true) hide columns to save tokens: showPpid, showUser, showStat, showCpu, showMem, showCmdline, showExe. Column order is fixed. includeKernel=false (default) omits kernel threads (empty cmdline); includeKernel=true includes them. Cap 1000 rows; truncated=true when more match. Mem is RSS in KiB. Cmdline/Exe are truncated with "…" when longer than the documented limit.`

const (
	psCmdlineMax = 256
	psExeMax     = 256
)

var psIdentity = []string{"Pid", "Comm"}
var psOptionalOrder = []string{"Ppid", "User", "Stat", "Cpu", "Mem", "Cmdline", "Exe"}

type PsArgs struct {
	IncludeKernel bool  `json:"includeKernel,omitempty" jsonschema:"if true, include kernel threads (default false)"`
	ShowPpid      *bool `json:"showPpid,omitempty" jsonschema:"include Ppid column (default true)"`
	ShowUser      *bool `json:"showUser,omitempty" jsonschema:"include User column (default true)"`
	ShowStat      *bool `json:"showStat,omitempty" jsonschema:"include Stat column (default true)"`
	ShowCpu       *bool `json:"showCpu,omitempty" jsonschema:"include Cpu column (default true)"`
	ShowMem       *bool `json:"showMem,omitempty" jsonschema:"include Mem column as RSS KiB (default true)"`
	ShowCmdline   *bool `json:"showCmdline,omitempty" jsonschema:"include Cmdline column (default true)"`
	ShowExe       *bool `json:"showExe,omitempty" jsonschema:"include Exe column (default true)"`
}

func (a PsArgs) visibleColumns() []string {
	return visibleColumnsOrdered(psIdentity, psOptionalOrder, map[string]bool{
		"Ppid":    showFlag(a.ShowPpid),
		"User":    showFlag(a.ShowUser),
		"Stat":    showFlag(a.ShowStat),
		"Cpu":     showFlag(a.ShowCpu),
		"Mem":     showFlag(a.ShowMem),
		"Cmdline": showFlag(a.ShowCmdline),
		"Exe":     showFlag(a.ShowExe),
	})
}

type procInfo struct {
	Pid     int
	Ppid    int
	Comm    string
	User    string
	Stat    string
	Cpu     string
	MemKiB  uint64
	Cmdline string
	Exe     string
	Kernel  bool
}

func (p procInfo) value(column string) string {
	switch column {
	case "Pid":
		return strconv.Itoa(p.Pid)
	case "Ppid":
		return strconv.Itoa(p.Ppid)
	case "Comm":
		return p.Comm
	case "User":
		return p.User
	case "Stat":
		return p.Stat
	case "Cpu":
		return p.Cpu
	case "Mem":
		return strconv.FormatUint(p.MemKiB, 10)
	case "Cmdline":
		return p.Cmdline
	case "Exe":
		return p.Exe
	default:
		return ""
	}
}

func (p procInfo) row(columns []string) string {
	var b strings.Builder
	b.WriteByte('|')
	for _, c := range columns {
		b.WriteString(escapeCell(p.value(c)))
		b.WriteByte('|')
	}
	b.WriteByte('\n')
	return b.String()
}

func escapeCell(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return s
}

func truncateEllipsis(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "…"
}

type psTable struct {
	columns   []string
	dataRows  []string
	total     int
	truncated bool
}

// procRoot is overridable in tests (default /proc).
var procRoot = "/proc"

func buildPsTable(args PsArgs) (*psTable, error) {
	columns := args.visibleColumns()
	needUser := showFlag(args.ShowUser)
	needCmdline := true // needed to classify kernel threads
	needExe := showFlag(args.ShowExe)
	needMem := showFlag(args.ShowMem)

	entries, err := os.ReadDir(procRoot)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", procRoot, err)
	}

	procs := make([]procInfo, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		pi, ok := readProc(pid, needUser, needCmdline, needExe, needMem)
		if !ok {
			continue
		}
		if !args.IncludeKernel && pi.Kernel {
			continue
		}
		procs = append(procs, pi)
	}

	total := len(procs)
	truncated := total > policy.MaxListEntries
	limit := total
	if truncated {
		limit = policy.MaxListEntries
	}

	rows := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		rows = append(rows, procs[i].row(columns))
	}
	return &psTable{columns: columns, dataRows: rows, total: total, truncated: truncated}, nil
}

func readProc(pid int, needUser, needCmdline, needExe, needMem bool) (procInfo, bool) {
	dir := filepath.Join(procRoot, strconv.Itoa(pid))
	statBytes, err := os.ReadFile(filepath.Join(dir, "stat"))
	if err != nil {
		return procInfo{}, false
	}
	pidOut, comm, state, ppid, ok := parseProcStat(statBytes)
	if !ok || pidOut != pid {
		return procInfo{}, false
	}

	pi := procInfo{Pid: pid, Ppid: ppid, Comm: comm, Stat: state, Cpu: "-"}

	var cmdline string
	if needCmdline {
		if b, err := os.ReadFile(filepath.Join(dir, "cmdline")); err == nil {
			cmdline = strings.ReplaceAll(string(bytes.TrimRight(b, "\x00")), "\x00", " ")
			cmdline = strings.TrimSpace(cmdline)
		}
	}
	pi.Kernel = cmdline == ""
	pi.Cmdline = truncateEllipsis(cmdline, psCmdlineMax)

	if needExe {
		if target, err := os.Readlink(filepath.Join(dir, "exe")); err == nil {
			pi.Exe = truncateEllipsis(target, psExeMax)
		}
	}

	statusPath := filepath.Join(dir, "status")
	var status []byte
	if needMem || needUser {
		status, _ = os.ReadFile(statusPath)
	}
	if needMem && status != nil {
		pi.MemKiB = parseStatusVmRSS(status)
	}
	if needUser {
		if st, err := os.Stat(dir); err == nil {
			if sys, ok := st.Sys().(*syscall.Stat_t); ok {
				pi.User = lookupUID(sys.Uid)
			} else if uid := parseStatusUID(status); uid >= 0 {
				pi.User = lookupUID(uint32(uid))
			}
		}
	}

	return pi, true
}

func parseStatusUID(b []byte) int64 {
	if b == nil {
		return -1
	}
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "Uid:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				u, err := strconv.ParseInt(fields[1], 10, 64)
				if err == nil {
					return u
				}
			}
		}
	}
	return -1
}

func parseStatusVmRSS(b []byte) uint64 {
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "VmRSS:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				v, err := strconv.ParseUint(fields[1], 10, 64)
				if err == nil {
					return v
				}
			}
		}
	}
	return 0
}

// parseProcStat parses /proc/pid/stat. Comm may contain spaces and is wrapped in ().
func parseProcStat(b []byte) (pid int, comm, state string, ppid int, ok bool) {
	s := string(b)
	lparen := strings.IndexByte(s, '(')
	rparen := strings.LastIndexByte(s, ')')
	if lparen < 1 || rparen <= lparen {
		return 0, "", "", 0, false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(s[:lparen]))
	if err != nil {
		return 0, "", "", 0, false
	}
	comm = s[lparen+1 : rparen]
	rest := strings.Fields(s[rparen+1:])
	if len(rest) < 2 {
		return 0, "", "", 0, false
	}
	state = rest[0]
	ppid, err = strconv.Atoi(rest[1])
	if err != nil {
		return 0, "", "", 0, false
	}
	return pid, comm, state, ppid, true
}

func lookupUID(uid uint32) string {
	u, err := user.LookupId(strconv.FormatUint(uint64(uid), 10))
	if err != nil {
		return strconv.FormatUint(uint64(uid), 10)
	}
	return u.Username
}

func renderPsTable(meta fmt.Stringer, tbl *psTable) *mcp.CallToolResult {
	var body strings.Builder
	body.WriteString(markdownHeader(tbl.columns))
	for _, row := range tbl.dataRows {
		body.WriteString(row)
	}
	text := toolmeta.Render(meta, &body)
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}
}

func Ps(ctx context.Context, req *mcp.CallToolRequest, args PsArgs) (*mcp.CallToolResult, any, error) {
	tbl, err := buildPsTable(args)
	if err != nil {
		return nil, nil, err
	}
	meta := toolmeta.PsHeader{
		Returned:  len(tbl.dataRows),
		Total:     tbl.total,
		Truncated: tbl.truncated,
		Columns:   joinColumns(tbl.columns),
	}
	return renderPsTable(meta, tbl), nil, nil
}

func AddPsTool(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "ps",
		Description: PsToolDescription,
	}, Ps)
}
