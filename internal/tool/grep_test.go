package tool_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/KaribuLab/linux-mcp/internal/tool"
)

func TestGrepLiteralModeDoesNotInterpretMetacharacters(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(path, []byte("foo.bar\nfooXbar\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, _, err := tool.Grep(context.Background(), nil, tool.GrepArgs{Path: path, Pattern: "foo.bar"})
	if err != nil {
		t.Fatal(err)
	}
	got := textOf(t, res)
	if !strings.Contains(got, "foo.bar") || strings.Contains(got, "fooXbar") {
		t.Fatalf("literal mode should not treat . as wildcard: %s", got)
	}
}

func TestGrepExtendedModeCompilesRegex(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(path, []byte("foo.bar\nfooXbar\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, _, err := tool.Grep(context.Background(), nil, tool.GrepArgs{Path: path, Pattern: "foo.bar", Extended: true})
	if err != nil {
		t.Fatal(err)
	}
	got := textOf(t, res)
	if !strings.Contains(got, "foo.bar") || !strings.Contains(got, "fooXbar") {
		t.Fatalf("extended mode should treat . as wildcard: %s", got)
	}
}

func TestGrepIgnoreCase(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(path, []byte("Hello World\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, _, err := tool.Grep(context.Background(), nil, tool.GrepArgs{Path: path, Pattern: "hello", IgnoreCase: true})
	if err != nil {
		t.Fatal(err)
	}
	got := textOf(t, res)
	if !strings.Contains(got, "Hello World") {
		t.Fatalf("ignoreCase failed: %s", got)
	}
}

func TestGrepSingleFileBinaryIsBlocked(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bin")
	if err := os.WriteFile(path, []byte("hello\x00world"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, _, err := tool.Grep(context.Background(), nil, tool.GrepArgs{Path: path, Pattern: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError || !strings.Contains(textOf(t, res), "class=binary") {
		t.Fatalf("want blocked binary, got %v %s", res.IsError, textOf(t, res))
	}
}

func TestGrepSingleFilePrivateKeyIsSearchedAndRedacted(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "key.pem.txt") // avoid the .pem path-denylist so the content sniff is what triggers redaction
	content := "-----BEGIN OPENSSH PRIVATE KEY-----\nsecret-material-here\n-----END OPENSSH PRIVATE KEY-----\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	res, _, err := tool.Grep(context.Background(), nil, tool.GrepArgs{Path: path, Pattern: "BEGIN", Extended: false})
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("private-key target must not be blocked, got %s", textOf(t, res))
	}
	got := textOf(t, res)
	if strings.Contains(got, "secret-material-here") {
		t.Fatalf("real key content leaked: %s", got)
	}
	if !strings.Contains(got, "[private-key content redacted]") {
		t.Fatalf("expected redaction placeholder: %s", got)
	}
	meta := strings.SplitN(got, "\n", 2)[0]
	if !strings.Contains(meta, "redacted=1") {
		t.Fatalf("expected redacted=1 in header: %s", meta)
	}
}

func TestGrepDirectorySkipsBinaryWithoutAborting(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "bin"), []byte("hello\x00world"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "text.txt"), []byte("hello there\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, _, err := tool.Grep(context.Background(), nil, tool.GrepArgs{Path: dir, Pattern: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	got := textOf(t, res)
	if res.IsError {
		t.Fatalf("recursive grep must not abort on a binary file: %s", got)
	}
	if !strings.Contains(got, "text.txt") {
		t.Fatalf("legitimate match missing: %s", got)
	}
}

func TestGrepDirectorySearchesAndRedactsPrivateKey(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "notakey.txt")
	content := "-----BEGIN RSA PRIVATE KEY-----\nsecret-material\n-----END RSA PRIVATE KEY-----\n"
	if err := os.WriteFile(keyPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "other.txt"), []byte("nothing interesting\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, _, err := tool.Grep(context.Background(), nil, tool.GrepArgs{Path: dir, Pattern: "BEGIN.*PRIVATE KEY", Extended: true})
	if err != nil {
		t.Fatal(err)
	}
	got := textOf(t, res)
	if strings.Contains(got, "secret-material") {
		t.Fatalf("real key content leaked during recursive grep: %s", got)
	}
	if !strings.Contains(got, keyPath) || !strings.Contains(got, "[private-key content redacted]") {
		t.Fatalf("expected redacted row for private-key file: %s", got)
	}
	meta := strings.SplitN(got, "\n", 2)[0]
	if !strings.Contains(meta, "redacted=1") {
		t.Fatalf("expected redacted=1: %s", meta)
	}
}

func TestGrepDeniedRootPath(t *testing.T) {
	res, _, err := tool.Grep(context.Background(), nil, tool.GrepArgs{Path: "/etc/shadow", Pattern: "root"})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError || !strings.Contains(textOf(t, res), "path_denied") {
		t.Fatalf("got %v %s", res.IsError, textOf(t, res))
	}
}

func TestGrepTruncatesLongLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "wide.txt")
	line := "needle-" + strings.Repeat("x", 100*1024) + "\n"
	if err := os.WriteFile(path, []byte(line), 0o644); err != nil {
		t.Fatal(err)
	}

	res, _, err := tool.Grep(context.Background(), nil, tool.GrepArgs{Path: path, Pattern: "needle"})
	if err != nil {
		t.Fatal(err)
	}
	got := textOf(t, res)
	if res.IsError {
		t.Fatalf("long matching line must not fail the request: %s", got)
	}
	if len(got) > 70*1024 {
		t.Fatalf("row was not truncated: %d bytes", len(got))
	}
}

func TestGrepTruncatesByMatchCap(t *testing.T) {
	dir := t.TempDir()
	var b strings.Builder
	for i := 0; i < 1100; i++ {
		b.WriteString("needle\n")
	}
	path := filepath.Join(dir, "many.txt")
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	res, _, err := tool.Grep(context.Background(), nil, tool.GrepArgs{Path: path, Pattern: "needle"})
	if err != nil {
		t.Fatal(err)
	}
	got := textOf(t, res)
	meta := strings.SplitN(got, "\n", 2)[0]
	if !strings.Contains(meta, "truncated=true") {
		t.Fatalf("want truncated by match cap: %s", meta)
	}
}

func TestGrepToolDescriptionContract(t *testing.T) {
	for _, needle := range []string{"[grep", "RE2", "redacted", "[blocked"} {
		if !strings.Contains(tool.GrepToolDescription, needle) {
			t.Fatalf("description missing %q", needle)
		}
	}
}
