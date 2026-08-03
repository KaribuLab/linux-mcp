package tool

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/KaribuLab/linux-mcp/internal/policy"
	"github.com/KaribuLab/linux-mcp/internal/toolmeta"
	diag "github.com/florianl/go-diag"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/sys/unix"
)

// SsToolDescription is the MCP tool description (agent-facing response contract).
const SsToolDescription = `List sockets via in-process netlink sock_diag (never runs the ss binary or any host shell). Wide model: default state=LISTEN and family=inet (IPv4+IPv6) for a cheap attack-surface view; widen with state=ESTAB|all and family=inet4|inet6|unix|all. On success: [ss entries=returned/total truncated=bool columns=...] + markdown table. Identity columns Proto and Local are always present. Optional show* flags (default true): showState, showPeer, showPid, showProcess, showUser, showFamily. Cap 1000 rows. Pid/Process are resolved from socket inode via /proc/*/fd; under the reference systemd unit (CAP_SYS_PTRACE, no ProtectProc=invisible) this includes processes of other users. Empty Pid/Process when the kernel denies the walk — never fabricated.`

// Linux TCP states (include/net/tcp_states.h) — not exported by x/sys/unix.
const (
	tcpEstablished = 1
	tcpSynSent     = 2
	tcpSynRecv     = 3
	tcpFinWait1    = 4
	tcpFinWait2    = 5
	tcpTimeWait    = 6
	tcpClose       = 7
	tcpCloseWait   = 8
	tcpLastAck     = 9
	tcpListen      = 10
	tcpClosing     = 11
)

var ssIdentity = []string{"Proto", "Local"}
var ssOptionalOrder = []string{"State", "Peer", "Pid", "Process", "User", "Family"}

type SsArgs struct {
	State       string `json:"state,omitempty" jsonschema:"LISTEN (default), ESTAB, or all"`
	Family      string `json:"family,omitempty" jsonschema:"inet (default IPv4+IPv6), inet4, inet6, unix, or all"`
	ShowState   *bool  `json:"showState,omitempty" jsonschema:"include State column (default true)"`
	ShowPeer    *bool  `json:"showPeer,omitempty" jsonschema:"include Peer column (default true)"`
	ShowPid     *bool  `json:"showPid,omitempty" jsonschema:"include Pid column (default true)"`
	ShowProcess *bool  `json:"showProcess,omitempty" jsonschema:"include Process column (default true)"`
	ShowUser    *bool  `json:"showUser,omitempty" jsonschema:"include User column (default true)"`
	ShowFamily  *bool  `json:"showFamily,omitempty" jsonschema:"include Family column (default true)"`
}

func (a SsArgs) visibleColumns() []string {
	return visibleColumnsOrdered(ssIdentity, ssOptionalOrder, map[string]bool{
		"State":   showFlag(a.ShowState),
		"Peer":    showFlag(a.ShowPeer),
		"Pid":     showFlag(a.ShowPid),
		"Process": showFlag(a.ShowProcess),
		"User":    showFlag(a.ShowUser),
		"Family":  showFlag(a.ShowFamily),
	})
}

type socketInfo struct {
	Proto   string
	Local   string
	Peer    string
	State   string
	Pid     int
	Process string
	User    string
	Family  string
}

func (s socketInfo) value(column string) string {
	switch column {
	case "Proto":
		return s.Proto
	case "Local":
		return s.Local
	case "Peer":
		return s.Peer
	case "State":
		return s.State
	case "Pid":
		if s.Pid <= 0 {
			return "-"
		}
		return strconv.Itoa(s.Pid)
	case "Process":
		return s.Process
	case "User":
		return s.User
	case "Family":
		return s.Family
	default:
		return ""
	}
}

func (s socketInfo) row(columns []string) string {
	var b strings.Builder
	b.WriteByte('|')
	for _, c := range columns {
		b.WriteString(escapeCell(s.value(c)))
		b.WriteByte('|')
	}
	b.WriteByte('\n')
	return b.String()
}

type ssTable struct {
	columns   []string
	dataRows  []string
	total     int
	truncated bool
}

// socketLister dumps sockets; overridden in tests.
var socketLister = listSocketsNetlink

func normalizeSsState(s string) (string, error) {
	if s == "" {
		return "LISTEN", nil
	}
	switch strings.ToUpper(s) {
	case "LISTEN", "ESTAB", "ALL":
		return strings.ToUpper(s), nil
	default:
		return "", fmt.Errorf("invalid state %q (want LISTEN, ESTAB, or all)", s)
	}
}

func normalizeSsFamily(f string) (string, error) {
	if f == "" {
		return "inet", nil
	}
	switch strings.ToLower(f) {
	case "inet", "inet4", "inet6", "unix", "all":
		return strings.ToLower(f), nil
	default:
		return "", fmt.Errorf("invalid family %q (want inet, inet4, inet6, unix, or all)", f)
	}
}

func buildSsTable(args SsArgs) (*ssTable, error) {
	state, err := normalizeSsState(args.State)
	if err != nil {
		return nil, err
	}
	family, err := normalizeSsFamily(args.Family)
	if err != nil {
		return nil, err
	}
	columns := args.visibleColumns()
	needPid := showFlag(args.ShowPid) || showFlag(args.ShowProcess)

	raw, err := socketLister(state, family)
	if err != nil {
		return nil, err
	}

	socks := attachProcOwners(raw, needPid, showFlag(args.ShowUser))

	total := len(socks)
	truncated := total > policy.MaxListEntries
	limit := total
	if truncated {
		limit = policy.MaxListEntries
	}
	rows := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		rows = append(rows, socks[i].row(columns))
	}
	return &ssTable{columns: columns, dataRows: rows, total: total, truncated: truncated}, nil
}

type socketRow struct {
	socketInfo
	Inode uint32
	UID   uint32
}

func listSocketsNetlink(state, family string) ([]socketRow, error) {
	nl, err := diag.Open(&diag.Config{})
	if err != nil {
		return nil, fmt.Errorf("netlink sock_diag open: %w", err)
	}
	defer nl.Close()

	var out []socketRow
	wantUnix := family == "unix" || family == "all"
	wantInet := family != "unix"

	if wantInet {
		families := inetFamilies(family)
		for _, af := range families {
			for _, proto := range []uint8{unix.IPPROTO_TCP, unix.IPPROTO_UDP} {
				opt := &diag.NetOption{
					Family:   af,
					Protocol: proto,
					State:    tcpStateMask(state, proto),
				}
				objs, err := nl.NetDump(opt)
				if err != nil {
					return nil, fmt.Errorf("netlink dump family=%d proto=%d: %w", af, proto, err)
				}
				for _, o := range objs {
					if !inetStateMatch(state, proto, o.State) {
						continue
					}
					row, err := netObjectToRow(o, proto)
					if err != nil {
						continue
					}
					out = append(out, row)
				}
			}
		}
	}

	if wantUnix {
		if state == "ESTAB" {
			// Unix has no TCP ESTAB semantics; omit for ESTAB-only scans.
		} else {
			uobjs, err := nl.UnixDump(&diag.UnixOption{State: 0xffffffff, Show: 1<<0 | 1<<5}) // name + uid
			if err != nil {
				return nil, fmt.Errorf("netlink unix dump: %w", err)
			}
			for _, u := range uobjs {
				out = append(out, unixObjectToRow(u))
			}
		}
	}

	return out, nil
}

func inetFamilies(family string) []uint8 {
	switch family {
	case "inet4":
		return []uint8{unix.AF_INET}
	case "inet6":
		return []uint8{unix.AF_INET6}
	case "inet", "all":
		return []uint8{unix.AF_INET, unix.AF_INET6}
	default:
		return nil
	}
}

func tcpStateMask(state string, proto uint8) uint32 {
	if proto != unix.IPPROTO_TCP {
		return 0xffffffff
	}
	switch state {
	case "LISTEN":
		return 1 << tcpListen
	case "ESTAB":
		return 1 << tcpEstablished
	default:
		return 0xffffffff
	}
}

func inetStateMatch(state string, proto uint8, st uint8) bool {
	if proto != unix.IPPROTO_TCP {
		// UDP: include when LISTEN/all (bound sockets); skip for ESTAB-only
		return state != "ESTAB"
	}
	switch state {
	case "LISTEN":
		return st == tcpListen
	case "ESTAB":
		return st == tcpEstablished
	default:
		return true
	}
}

func tcpStateName(st uint8) string {
	switch st {
	case tcpEstablished:
		return "ESTAB"
	case tcpSynSent:
		return "SYN-SENT"
	case tcpSynRecv:
		return "SYN-RECV"
	case tcpFinWait1:
		return "FIN-WAIT-1"
	case tcpFinWait2:
		return "FIN-WAIT-2"
	case tcpTimeWait:
		return "TIME-WAIT"
	case tcpClose:
		return "CLOSE"
	case tcpCloseWait:
		return "CLOSE-WAIT"
	case tcpLastAck:
		return "LAST-ACK"
	case tcpListen:
		return "LISTEN"
	case tcpClosing:
		return "CLOSING"
	default:
		return strconv.Itoa(int(st))
	}
}

func netObjectToRow(o diag.NetObject, proto uint8) (socketRow, error) {
	localIP, err := diag.ToNetipAddrWithFamily(o.Family, o.ID.Src)
	if err != nil {
		return socketRow{}, err
	}
	remoteIP, err := diag.ToNetipAddrWithFamily(o.Family, o.ID.Dst)
	if err != nil {
		return socketRow{}, err
	}
	protoName := "tcp"
	if proto == unix.IPPROTO_UDP {
		protoName = "udp"
	}
	fam := "inet"
	if o.Family == unix.AF_INET6 {
		fam = "inet6"
	} else if o.Family == unix.AF_INET {
		fam = "inet4"
	}
	local := net.JoinHostPort(localIP.String(), strconv.Itoa(int(diag.Ntohs(o.ID.SPort))))
	peer := net.JoinHostPort(remoteIP.String(), strconv.Itoa(int(diag.Ntohs(o.ID.DPort))))
	st := "-"
	if proto == unix.IPPROTO_TCP {
		st = tcpStateName(o.State)
	}
	return socketRow{
		socketInfo: socketInfo{
			Proto:  protoName,
			Local:  local,
			Peer:   peer,
			State:  st,
			Family: fam,
			User:   lookupUID(o.UID),
		},
		Inode: o.INode,
		UID:   o.UID,
	}, nil
}

func unixObjectToRow(u diag.UnixObject) socketRow {
	name := "-"
	if u.Name != nil && *u.Name != "" {
		name = *u.Name
	}
	userName := "-"
	var uid uint32
	if u.UID != nil {
		uid = *u.UID
		userName = lookupUID(uid)
	}
	return socketRow{
		socketInfo: socketInfo{
			Proto:  "unix",
			Local:  name,
			Peer:   "-",
			State:  "-",
			Family: "unix",
			User:   userName,
		},
		Inode: u.Ino,
		UID:   uid,
	}
}

func buildSocketInodeMap() map[uint32]struct{ Pid int; Comm string } {
	m := make(map[uint32]struct{ Pid int; Comm string })
	entries, err := os.ReadDir(procRoot)
	if err != nil {
		return m
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		fdDir := filepath.Join(procRoot, e.Name(), "fd")
		fds, err := os.ReadDir(fdDir)
		if err != nil {
			continue
		}
		var comm string
		if b, err := os.ReadFile(filepath.Join(procRoot, e.Name(), "comm")); err == nil {
			comm = strings.TrimSpace(string(b))
		}
		for _, fd := range fds {
			target, err := os.Readlink(filepath.Join(fdDir, fd.Name()))
			if err != nil {
				continue
			}
			const prefix = "socket:["
			if !strings.HasPrefix(target, prefix) || !strings.HasSuffix(target, "]") {
				continue
			}
			inoStr := target[len(prefix) : len(target)-1]
			ino, err := strconv.ParseUint(inoStr, 10, 32)
			if err != nil {
				continue
			}
			m[uint32(ino)] = struct {
				Pid  int
				Comm string
			}{Pid: pid, Comm: comm}
		}
	}
	return m
}

func attachProcOwners(rows []socketRow, needPid, needUser bool) []socketInfo {
	var inodeMap map[uint32]struct{ Pid int; Comm string }
	if needPid {
		inodeMap = buildSocketInodeMap()
	}
	out := make([]socketInfo, len(rows))
	for i, r := range rows {
		info := r.socketInfo
		if needUser && info.User == "" {
			info.User = lookupUID(r.UID)
		}
		if needPid {
			if owner, ok := inodeMap[r.Inode]; ok {
				info.Pid = owner.Pid
				info.Process = owner.Comm
			}
		}
		out[i] = info
	}
	return out
}

func renderSsTable(meta fmt.Stringer, tbl *ssTable) *mcp.CallToolResult {
	var body strings.Builder
	body.WriteString(markdownHeader(tbl.columns))
	for _, row := range tbl.dataRows {
		body.WriteString(row)
	}
	text := toolmeta.Render(meta, &body)
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}
}

func Ss(ctx context.Context, req *mcp.CallToolRequest, args SsArgs) (*mcp.CallToolResult, any, error) {
	tbl, err := buildSsTable(args)
	if err != nil {
		return nil, nil, err
	}
	meta := toolmeta.SsHeader{
		Returned:  len(tbl.dataRows),
		Total:     tbl.total,
		Truncated: tbl.truncated,
		Columns:   joinColumns(tbl.columns),
	}
	return renderSsTable(meta, tbl), nil, nil
}

func AddSsTool(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "ss",
		Description: SsToolDescription,
	}, Ss)
}
