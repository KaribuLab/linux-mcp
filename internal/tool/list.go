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
const ListToolDescription = `List directory entries with a bounded markdown response (max 1000 rows). On success the first line is metadata: [list path=... entries=returned/total truncated=bool] followed by a markdown table. With list=false columns are |File|; with list=true columns are |Name|Size|Mode|Owner|Group|ModTime|IsDir|IsSymlink|SymlinkPath|. On path policy block returns a single line [blocked class=... path=...] with no rows. Does not dump unbounded directory listings.`

type ListFilesArgs struct {
	Path string `json:"path" jsonschema:"the path to list the files from"`
	All  bool   `json:"all" jsonschema:"whether to list all files, including hidden files"`
	List bool   `json:"list" jsonschema:"whether to list detailed columns (true) or names only (false)"`
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

func (f fileInfo) detailedRow() string {
	return fmt.Sprintf("|%s|%d|%s|%s|%s|%s|%t|%t|%s|\n",
		f.Name, f.Size, f.Mode, f.Owner, f.Group, f.ModTime, f.IsDir, f.IsSymlink, f.SymlinkPath)
}

func (f fileInfo) simpleRow() string {
	return fmt.Sprintf("|%s|\n", f.Name)
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

	var body strings.Builder
	if args.List {
		body.WriteString("|Name|Size|Mode|Owner|Group|ModTime|IsDir|IsSymlink|SymlinkPath|\n")
		body.WriteString("|---|---|---|---|---|---|---|---|---|\n")
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
			uid := info.Sys().(*syscall.Stat_t).Uid
			gid := info.Sys().(*syscall.Stat_t).Gid
			ownerName := strconv.Itoa(int(uid))
			if owner, err := user.LookupId(strconv.Itoa(int(uid))); err == nil {
				ownerName = owner.Username
			}
			groupName := strconv.Itoa(int(gid))
			if group, err := user.LookupGroupId(strconv.Itoa(int(gid))); err == nil {
				groupName = group.Name
			}
			var symlinkPath string
			isSymlink := file.Type()&os.ModeSymlink != 0
			if isSymlink {
				symlinkPath, err = os.Readlink(filepath.Join(abs, file.Name()))
				if err != nil && !os.IsNotExist(err) {
					return nil, nil, fmt.Errorf("failed to read symlink in file %s: %w", file.Name(), err)
				}
			}
			body.WriteString(fileInfo{
				Name:        file.Name(),
				Size:        info.Size(),
				Mode:        info.Mode(),
				ModTime:     info.ModTime(),
				IsDir:       file.IsDir(),
				IsSymlink:   isSymlink,
				SymlinkPath: symlinkPath,
				Owner:       ownerName,
				Group:       groupName,
			}.detailedRow())
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
