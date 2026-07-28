package tool_test

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/KaribuLab/linux-mcp/internal/tool"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func textOf(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	if res == nil || len(res.Content) == 0 {
		t.Fatal("empty result")
	}
	tc, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("want TextContent, got %T", res.Content[0])
	}
	return tc.Text
}

func TestCatFileSmall(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(path, []byte("one\ntwo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, _, err := tool.CatFile(context.Background(), nil, tool.CatFileArgs{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("unexpected error result: %s", textOf(t, res))
	}
	got := textOf(t, res)
	if !strings.HasPrefix(got, "[cat path=") || !strings.Contains(got, "truncated=false") {
		t.Fatalf("bad meta: %s", got)
	}
	if !strings.Contains(got, "\none\ntwo\n") && !strings.HasSuffix(got, "\none\ntwo\n") {
		// body after meta
		parts := strings.SplitN(got, "\n", 2)
		if len(parts) < 2 || !strings.Contains(parts[1], "one") {
			t.Fatalf("missing body: %q", got)
		}
	}
}

func TestCatFileTruncatesAndResumes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "big.txt")
	var b strings.Builder
	for i := 0; i < 250; i++ {
		b.WriteString("line-")
		b.WriteString(strconv.Itoa(i))
		b.WriteByte('\n')
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	res1, _, err := tool.CatFile(context.Background(), nil, tool.CatFileArgs{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	t1 := textOf(t, res1)
	if !strings.Contains(t1, "truncated=true") || !strings.Contains(t1, "next=") {
		t.Fatalf("want truncated+next: %s", strings.SplitN(t1, "\n", 2)[0])
	}
	meta := strings.SplitN(t1, "\n", 2)[0]
	idx := strings.Index(meta, "next=")
	nextStr := strings.TrimSuffix(meta[idx+5:], "]")
	next, err := strconv.ParseInt(nextStr, 10, 64)
	if err != nil {
		t.Fatalf("parse next: %v from %q", err, nextStr)
	}
	res2, _, err := tool.CatFile(context.Background(), nil, tool.CatFileArgs{Path: path, Offset: next})
	if err != nil {
		t.Fatal(err)
	}
	t2 := textOf(t, res2)
	body1 := strings.SplitN(t1, "\n", 2)[1]
	body2 := strings.SplitN(t2, "\n", 2)[1]
	if !strings.HasPrefix(body1, "line-0\n") {
		t.Fatalf("page1 should start at line-0: %q", body1[:32])
	}
	if !strings.HasPrefix(body2, "line-100\n") {
		t.Fatalf("page2 should start at line-100 after seek: %q", body2[:32])
	}
}

func TestCatFileBlocksPEM(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "k.txt")
	if err := os.WriteFile(path, []byte("-----BEGIN RSA PRIVATE KEY-----\nxx\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	res, _, err := tool.CatFile(context.Background(), nil, tool.CatFileArgs{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Fatal("expected IsError")
	}
	got := textOf(t, res)
	if !strings.Contains(got, "class=private_key") {
		t.Fatalf("got %s", got)
	}
}

func TestCatFileBlocksShadow(t *testing.T) {
	res, _, err := tool.CatFile(context.Background(), nil, tool.CatFileArgs{Path: "/etc/shadow"})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Fatal("expected IsError")
	}
	if !strings.Contains(textOf(t, res), "path_denied") {
		t.Fatalf("got %s", textOf(t, res))
	}
}

func TestCatFileBlocksBinary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bin.dat")
	if err := os.WriteFile(path, []byte{'a', 0, 'b'}, 0o644); err != nil {
		t.Fatal(err)
	}
	res, _, err := tool.CatFile(context.Background(), nil, tool.CatFileArgs{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError || !strings.Contains(textOf(t, res), "binary") {
		t.Fatalf("got %v %s", res.IsError, textOf(t, res))
	}
}

func TestCatToolDescriptionContract(t *testing.T) {
	for _, needle := range []string{"[cat", "[blocked", "offset", "100", "64", "raw text"} {
		if !strings.Contains(tool.CatToolDescription, needle) {
			t.Fatalf("description missing %q", needle)
		}
	}
}
