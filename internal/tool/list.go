package tool

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/KaribuLab/linux-mcp/internal/policy"
	"github.com/KaribuLab/linux-mcp/internal/toolmeta"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ListToolDescription is the MCP tool description (agent-facing response contract).
const ListToolDescription = `List directory entries with a bounded markdown response (max 1000 rows). On success the first line is metadata: [list path=... entries=returned/total truncated=bool] followed by a markdown table. With list=false columns are |File|; with list=true columns are |Name|Size|Mode|Owner|Group|ModTime|IsDir|IsSymlink|SymlinkPath| by default. When list=true, eight optional boolean flags let you hide columns to save tokens: showSize, showMode, showOwner, showGroup, showModTime, showIsDir, showIsSymlink, showSymlinkPath — each defaults to true (visible); set to false to omit that column. Name is always included and cannot be hidden. Column order is always fixed (never reordered by these flags). These flags are ignored when list=false. When list=true, the metadata line also includes columns=<c1,c2,...> listing the exact columns present in the table, in order. On path policy block returns a single line [blocked class=... path=...] with no rows. Does not dump unbounded directory listings.`

// listColumns is the fixed canonical order of detailed-mode columns. Name is
// always visible; every other column can be hidden via a Show* flag.
var listColumns = []string{"Name", "Size", "Mode", "Owner", "Group", "ModTime", "IsDir", "IsSymlink", "SymlinkPath"}

type ListFilesArgs struct {
	Path            string `json:"path" jsonschema:"the path to list the files from"`
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

// visibleColumns resolves the fixed, ordered list of detailed-mode columns to
// render, applying the Show* flags (nil or true => visible). Name is always
// included and cannot be hidden.
func (a ListFilesArgs) visibleColumns() []string {
	show := func(v *bool) bool { return v == nil || *v }
	flags := map[string]bool{
		"Size":        show(a.ShowSize),
		"Mode":        show(a.ShowMode),
		"Owner":       show(a.ShowOwner),
		"Group":       show(a.ShowGroup),
		"ModTime":     show(a.ShowModTime),
		"IsDir":       show(a.ShowIsDir),
		"IsSymlink":   show(a.ShowIsSymlink),
		"SymlinkPath": show(a.ShowSymlinkPath),
	}
	cols := make([]string, 0, len(listColumns))
	for _, c := range listColumns {
		if c == "Name" || flags[c] {
			cols = append(cols, c)
		}
	}
	return cols
}

type fileInfo struct {
	Name        string
	Size        int64
	Mode        os.FileMode
	Owner       string
	Group       string
	ModTime     time.Time
	IsDir       bool
	IsSymlink   bool
	SymlinkPath string
}

// value returns the rendered string for a single column of this entry.
func (f fileInfo) value(column string) string {
	switch column {
	case "Name":
		return f.Name
	case "Size":
		return strconv.FormatInt(f.Size, 10)
	case "Mode":
		return f.Mode.String()
	case "Owner":
		return f.Owner
	case "Group":
		return f.Group
	case "ModTime":
		return f.ModTime.String()
	case "IsDir":
		return strconv.FormatBool(f.IsDir)
	case "IsSymlink":
		return strconv.FormatBool(f.IsSymlink)
	case "SymlinkPath":
		return f.SymlinkPath
	default:
		return ""
	}
}

func (f fileInfo) detailedRow(columns []string) string {
	var b strings.Builder
	b.WriteByte('|')
	for _, c := range columns {
		b.WriteString(f.value(c))
		b.WriteByte('|')
	}
	b.WriteByte('\n')
	return b.String()
}

func (f fileInfo) simpleRow() string {
	return fmt.Sprintf("|%s|\n", f.Name)
}

func markdownHeader(columns []string) string {
	var b strings.Builder
	b.WriteByte('|')
	for _, c := range columns {
		b.WriteString(c)
		b.WriteByte('|')
	}
	b.WriteByte('\n')
	b.WriteByte('|')
	b.WriteString(strings.Repeat("---|", len(columns)))
	b.WriteByte('\n')
	return b.String()
}

func ListFiles(ctx context.Context, req *mcp.CallToolRequest, args ListFilesArgs) (*mcp.CallToolResult, any, error) {
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

	files, err := os.ReadDir(abs)
	if err != nil {
		return nil, nil, err
	}

	visible := make([]os.DirEntry, 0, len(files))
	for _, file := range files {
		if args.All || !strings.HasPrefix(file.Name(), ".") {
			visible = append(visible, file)
		}
	}
	total := len(visible)
	truncated := total > policy.MaxListEntries
	limit := total
	if truncated {
		limit = policy.MaxListEntries
	}

	var columns []string
	var computeOwner, computeGroup, computeSymlinkPath bool
	var body strings.Builder
	if args.List {
		columns = args.visibleColumns()
		computeOwner = args.ShowOwner == nil || *args.ShowOwner
		computeGroup = args.ShowGroup == nil || *args.ShowGroup
		computeSymlinkPath = args.ShowSymlinkPath == nil || *args.ShowSymlinkPath
		body.WriteString(markdownHeader(columns))
	} else {
		body.WriteString("|File|\n")
		body.WriteString("|---|\n")
	}

	for i := 0; i < limit; i++ {
		file := visible[i]
		if args.List {
			info, err := file.Info()
			if err != nil {
				return nil, nil, fmt.Errorf("failed to get file info: %w", err)
			}
			fi := fileInfo{
				Name:      file.Name(),
				Size:      info.Size(),
				Mode:      info.Mode(),
				ModTime:   info.ModTime(),
				IsDir:     file.IsDir(),
				IsSymlink: file.Type()&os.ModeSymlink != 0,
			}
			if computeOwner || computeGroup {
				uid := info.Sys().(*syscall.Stat_t).Uid
				gid := info.Sys().(*syscall.Stat_t).Gid
				if computeOwner {
					fi.Owner = strconv.Itoa(int(uid))
					if owner, err := user.LookupId(strconv.Itoa(int(uid))); err == nil {
						fi.Owner = owner.Username
					}
				}
				if computeGroup {
					fi.Group = strconv.Itoa(int(gid))
					if group, err := user.LookupGroupId(strconv.Itoa(int(gid))); err == nil {
						fi.Group = group.Name
					}
				}
			}
			if fi.IsSymlink && computeSymlinkPath {
				fi.SymlinkPath, err = os.Readlink(filepath.Join(abs, file.Name()))
				if err != nil && !os.IsNotExist(err) {
					return nil, nil, fmt.Errorf("failed to read symlink in file %s: %w", file.Name(), err)
				}
			}
			body.WriteString(fi.detailedRow(columns))
		} else {
			body.WriteString(fileInfo{Name: file.Name()}.simpleRow())
		}
	}

	header := toolmeta.ListHeader{
		Path:      abs,
		Returned:  limit,
		Total:     total,
		Truncated: truncated,
	}
	if truncated {
		header.Next = limit
	}
	if args.List {
		header.Columns = strings.Join(columns, ",")
	}
	text := toolmeta.Render(header, &body)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}, nil, nil
}

func AddListFilesTool(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list",
		Description: ListToolDescription,
	}, ListFiles)
}
