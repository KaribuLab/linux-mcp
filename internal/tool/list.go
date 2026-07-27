package tool

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ListFilesArgs struct {
	Path string `json:"path" jsonschema:"the path to list the files from"`
	All  bool   `json:"all" jsonschema:"whether to list all files, including hidden files"`
	List bool   `json:"list" jsonschema:"whether to list the files as a list"`
}

type fileInfo struct {
	Name        string      `json:"name" jsonschema:"the name of the file"`
	Size        int64       `json:"size,omitempty" jsonschema:"the size of the file in bytes"`
	Mode        os.FileMode `json:"mode,omitempty" jsonschema:"the mode of the file"`
	Owner       string      `json:"owner,omitempty" jsonschema:"the owner of the file"`
	Group       string      `json:"group,omitempty" jsonschema:"the group of the file"`
	ModTime     time.Time   `json:"modTime,omitzero" jsonschema:"the modification time of the file"`
	IsDir       bool        `json:"isDir" jsonschema:"whether the file is a directory"`
	IsSymlink   bool        `json:"isSymlink,omitempty" jsonschema:"whether the file is a symlink"`
	SymlinkPath string      `json:"symlinkPath,omitempty" jsonschema:"the path of the symlink"`
}

func (f fileInfo) String() string {
	return fmt.Sprintf("|%s|%d|%s|%s|%s|%s|%t|%t|%s|\n", f.Name, f.Size, f.Mode, f.Owner, f.Group, f.ModTime, f.IsDir, f.IsSymlink, f.SymlinkPath)
}

func ListFiles(ctx context.Context, req *mcp.CallToolRequest, args ListFilesArgs) (*mcp.CallToolResult, any, error) {
	files, err := os.ReadDir(args.Path)
	if err != nil {
		return nil, nil, err
	}
	var content strings.Builder
	if args.List {
		content.WriteString("|Name|Size|Mode|Owner|Group|ModTime|IsDir|IsSymlink|SymlinkPath|\n")
		content.WriteString("|---|---|---|---|---|---|---|---|---|\n")
	} else {
		content.WriteString("|File|\n")
		content.WriteString("|---|\n")
	}
	for _, file := range files {
		if args.All || !strings.HasPrefix(file.Name(), ".") {
			if args.List {
				info, err := file.Info()
				if err != nil {
					return nil, nil, fmt.Errorf("failed to get file info: %v", err)
				}
				uid := info.Sys().(*syscall.Stat_t).Uid
				gid := info.Sys().(*syscall.Stat_t).Gid
				owner, err := user.LookupId(strconv.Itoa(int(uid)))
				if err != nil {
					return nil, nil, fmt.Errorf("failed to lookup owner in file %s: %v", file.Name(), err)
				}
				group, err := user.LookupId(strconv.Itoa(int(gid)))
				if err != nil {
					return nil, nil, fmt.Errorf("failed to lookup group in file %s: %v", file.Name(), err)
				}
				var symlinkPath string
				if file.Type() == os.ModeSymlink {
					symlinkPath, err = os.Readlink(file.Name())
					if err != nil && !os.IsNotExist(err) {
						return nil, nil, fmt.Errorf("failed to read symlink in file %s: %v", file.Name(), err)
					}
				}
				content.WriteString(fileInfo{
					Name:        file.Name(),
					Size:        info.Size(),
					Mode:        info.Mode(),
					ModTime:     info.ModTime(),
					IsDir:       file.IsDir(),
					IsSymlink:   file.Type() == os.ModeSymlink,
					SymlinkPath: symlinkPath,
					Owner:       owner.Username,
					Group:       group.Name,
				}.String())
			} else {
				content.WriteString(fileInfo{
					Name: file.Name(),
				}.String())
			}
		}
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: content.String()},
		},
	}, nil, nil
}

func AddListFilesTool(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list",
		Description: "List the files in a directory in markdown format",
	}, ListFiles)
}
