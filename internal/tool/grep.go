package tool

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/KaribuLab/linux-mcp/internal/policy"
	"github.com/KaribuLab/linux-mcp/internal/toolmeta"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// GrepToolDescription is the MCP tool description (agent-facing response contract).
const GrepToolDescription = `Search a file or, recursively, a directory tree for a pattern. Default mode (extended=false) matches the pattern as literal text (like grep -F); extended=true compiles it as an RE2 regular expression (Go's regexp package, linear time, no backtracking, not PCRE). ignoreCase makes either mode case-insensitive. Directory targets use the same shared walk as find (denylist per node, no symlink following, node budget); maxDepth bounds recursion (0/omitted = unlimited). Binary content (NUL byte in the first bytes) is skipped silently during a recursive search, and a single binary file target returns [blocked class=binary path=...] without searching. Private-key-classified content (PEM/OpenSSH/PuTTY headers) is NOT skipped or blocked: it is searched like any other file, but every matching row's content is replaced with the fixed placeholder "[private-key content redacted]" instead of the real text, so an agent can still detect a misplaced private key by path/line without the key material ever being returned; these rows are counted in the redacted=<n> header field. On success the first line is metadata: [grep path=... matches=returned/total truncated=bool filesScanned=... redacted=...] followed by raw <path>:<line>:<content> rows (not markdown). Each row's content is capped at 64KiB; truncated=true when the row cap, per-line cap, or the walk's node budget is hit. On a denied root path returns a single line [blocked class=... path=...] with no rows.`

const redactedPlaceholder = "[private-key content redacted]"

type GrepArgs struct {
	Path       string `json:"path" jsonschema:"file or directory to search"`
	Pattern    string `json:"pattern" jsonschema:"literal text (default) or RE2 regular expression (extended=true) to search for"`
	Extended   bool   `json:"extended,omitempty" jsonschema:"if true, compile pattern as an RE2 regular expression instead of matching it as literal text"`
	IgnoreCase bool   `json:"ignoreCase,omitempty" jsonschema:"case-insensitive match"`
	MaxDepth   int    `json:"maxDepth,omitempty" jsonschema:"maximum recursion depth when path is a directory (0 or omitted = unlimited)"`
}

func compileGrepPattern(pattern string, extended, ignoreCase bool) (*regexp.Regexp, error) {
	p := pattern
	if !extended {
		p = regexp.QuoteMeta(p)
	}
	if ignoreCase {
		p = "(?i)" + p
	}
	return regexp.Compile(p)
}

// grepScan streams path line by line, appending "<path>:<line>:<content>\n"
// rows to *matches for every line matching re, up to policy.MaxGrepMatches
// rows kept (all matches still count toward *total). Lines from a file
// classified as private-key are appended with content replaced by
// redactedPlaceholder and counted in *redacted; lines from a binary-classified
// file are not read at all (scanned=false, no rows, no error). Returns
// scanned=false without error when the file is skipped as binary.
func grepScan(path string, re *regexp.Regexp, matches *[]string, total, redacted *int) (scanned bool, class policy.Class, err error) {
	f, err := os.Open(path)
	if err != nil {
		return false, "", err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return false, "", err
	}
	if err := policy.CheckReadableType(path, info); err != nil {
		return false, "", err
	}

	prefix := make([]byte, policy.PrefixBytes)
	n, err := io.ReadFull(f, prefix)
	if err == io.ErrUnexpectedEOF || err == io.EOF {
		prefix = prefix[:n]
		err = nil
	}
	if err != nil {
		return false, "", err
	}
	class = policy.ClassifyPrefix(prefix)
	if class == policy.ClassBinary {
		return false, class, nil
	}
	redact := class == policy.ClassPrivateKey

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return false, class, err
	}

	br := bufio.NewReaderSize(f, 64*1024)
	lineNo := 0
	for {
		lineNo++
		raw, readErr := br.ReadBytes('\n')
		if len(raw) > 0 {
			line := strings.TrimRight(string(raw), "\r\n")
			if re.MatchString(line) {
				*total++
				var content string
				if redact {
					content = redactedPlaceholder
					*redacted++
				} else {
					content = line
					if len(content) > policy.MaxBytes {
						content = content[:policy.MaxBytes]
					}
				}
				if len(*matches) < policy.MaxGrepMatches {
					*matches = append(*matches, fmt.Sprintf("%s:%d:%s\n", path, lineNo, content))
				}
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return true, class, readErr
		}
	}
	return true, class, nil
}

func Grep(ctx context.Context, req *mcp.CallToolRequest, args GrepArgs) (*mcp.CallToolResult, any, error) {
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

	info, err := os.Lstat(abs)
	if err != nil {
		return nil, nil, err
	}

	re, err := compileGrepPattern(args.Pattern, args.Extended, args.IgnoreCase)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid pattern: %w", err)
	}

	var (
		matches       []string
		totalMatches  int
		redacted      int
		filesScanned  int
		walkTruncated bool
	)

	if info.IsDir() {
		_, trunc, walkErr := policy.Walk(abs, policy.WalkLimits{}, 0, args.MaxDepth, func(path string, nodeInfo os.FileInfo, depth int) error {
			if nodeInfo.IsDir() || nodeInfo.Mode()&os.ModeSymlink != 0 || !nodeInfo.Mode().IsRegular() {
				return nil
			}
			scanned, _, scanErr := grepScan(path, re, &matches, &totalMatches, &redacted)
			if scanErr != nil {
				// Unreadable file (permissions, race): skip it, keep walking.
				return nil
			}
			if scanned {
				filesScanned++
			}
			return nil
		})
		if walkErr != nil {
			return nil, nil, walkErr
		}
		walkTruncated = trunc
	} else {
		if err := policy.CheckReadableType(abs, info); err != nil {
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
		scanned, class, scanErr := grepScan(abs, re, &matches, &totalMatches, &redacted)
		if scanErr != nil {
			return nil, nil, scanErr
		}
		if class == policy.ClassBinary {
			text := toolmeta.Blocked{Class: string(policy.ClassBinary), Path: abs}.String()
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: text}},
				IsError: true,
			}, nil, nil
		}
		if scanned {
			filesScanned = 1
		}
	}

	truncated := walkTruncated || totalMatches > len(matches)

	var body strings.Builder
	for _, row := range matches {
		body.WriteString(row)
	}

	header := toolmeta.GrepHeader{
		Path:         abs,
		Returned:     len(matches),
		Total:        totalMatches,
		Truncated:    truncated,
		FilesScanned: filesScanned,
		Redacted:     redacted,
	}
	text := toolmeta.Render(header, &body)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}, nil, nil
}

func AddGrepTool(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "grep",
		Description: GrepToolDescription,
	}, Grep)
}
