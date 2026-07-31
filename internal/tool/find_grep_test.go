package tool_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/KaribuLab/linux-mcp/internal/tool"
)

func TestFindGrepNameAndContent(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte("TODO fix me\nok\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("TODO in txt\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "c.go"), []byte("nothing here\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, _, err := tool.FindGrep(context.Background(), nil, tool.FindGrepArgs{
		Path:    dir,
		Name:    "*.go",
		Pattern: "TODO",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := textOf(t, res)
	if !strings.HasPrefix(got, "[find_grep path=") {
		t.Fatalf("missing meta: %s", got)
	}
	if !strings.Contains(got, "a.go:") || !strings.Contains(got, "TODO fix me") {
		t.Fatalf("want match in a.go: %s", got)
	}
	if strings.Contains(got, "b.txt") {
		t.Fatalf("b.txt outside name filter must not appear: %s", got)
	}
	if strings.Contains(got, "c.go") {
		t.Fatalf("c.go has no TODO: %s", got)
	}
}

func TestFindGrepSkipsDirectoryMatches(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "subdir")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "nested.txt"), []byte("secret-hit\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// type=d matches only directories — no content rows.
	res, _, err := tool.FindGrep(context.Background(), nil, tool.FindGrepArgs{
		Path:    dir,
		Type:    "d",
		Pattern: "secret-hit",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := textOf(t, res)
	if strings.Contains(got, "secret-hit") {
		t.Fatalf("directory matches must not be content-scanned: %s", got)
	}
	if !strings.Contains(got, "matches=0/0") {
		t.Fatalf("want zero matches: %s", got)
	}
}

func TestFindGrepPrivateKeyRedacted(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "key.pem.txt")
	content := "-----BEGIN OPENSSH PRIVATE KEY-----\nsecret-material-here\n-----END OPENSSH PRIVATE KEY-----\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	res, _, err := tool.FindGrep(context.Background(), nil, tool.FindGrepArgs{
		Path: dir, Pattern: "BEGIN",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := textOf(t, res)
	if strings.Contains(got, "secret-material-here") {
		t.Fatalf("key leaked: %s", got)
	}
	if !strings.Contains(got, "[private-key content redacted]") {
		t.Fatalf("want redaction: %s", got)
	}
	if !strings.Contains(strings.SplitN(got, "\n", 2)[0], "redacted=") {
		t.Fatalf("want redacted= in meta: %s", got)
	}
}

func TestFindGrepDeniedRoot(t *testing.T) {
	res, _, err := tool.FindGrep(context.Background(), nil, tool.FindGrepArgs{
		Path: "/etc/shadow", Pattern: "x",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError || !strings.Contains(textOf(t, res), "path_denied") {
		t.Fatalf("got %v %s", res.IsError, textOf(t, res))
	}
}

func TestFindGrepExtendedRegex(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("foo.bar\nfooXbar\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, _, err := tool.FindGrep(context.Background(), nil, tool.FindGrepArgs{
		Path: dir, Pattern: "foo.bar", Extended: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	got := textOf(t, res)
	if !strings.Contains(got, "foo.bar") || !strings.Contains(got, "fooXbar") {
		t.Fatalf("extended content search: %s", got)
	}
}
