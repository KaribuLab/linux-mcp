package tool_test

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/KaribuLab/linux-mcp/internal/policy"
	"github.com/KaribuLab/linux-mcp/internal/tool"
)

func TestListGrepFiltersSimpleNames(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.txt", "b.md", "c.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	res, _, err := tool.ListGrep(context.Background(), nil, tool.ListGrepArgs{
		Path:    dir,
		Pattern: ".txt",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := textOf(t, res)
	if !strings.HasPrefix(got, "[list_grep path=") {
		t.Fatalf("missing meta: %s", got)
	}
	if !strings.Contains(got, "|a.txt|") || !strings.Contains(got, "|c.txt|") {
		t.Fatalf("want .txt rows: %s", got)
	}
	if strings.Contains(got, "|b.md|") {
		t.Fatalf("md should be filtered out: %s", got)
	}
	if strings.Contains(got, "|File|\n|File|") {
		t.Fatalf("header duplicated as data: %s", got)
	}
}

func TestListGrepGlobStarTxtMatchesNamesNotContents(t *testing.T) {
	dir := t.TempDir()
	// Content would match "*.txt" literally only if we searched file bodies — we must not.
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("no-glob-here\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.md"), []byte("*.txt\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "c.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, _, err := tool.ListGrep(context.Background(), nil, tool.ListGrepArgs{
		Path:    dir,
		Pattern: "*.txt",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := textOf(t, res)
	if !strings.Contains(got, "|a.txt|") || !strings.Contains(got, "|c.txt|") {
		t.Fatalf("glob *.txt should match names: %s", got)
	}
	if strings.Contains(got, "|b.md|") {
		t.Fatalf("b.md content has *.txt but name must not match glob: %s", got)
	}
}

func TestListGrepDetailedRowFullLineMatch(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "keep.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "drop.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, _, err := tool.ListGrep(context.Background(), nil, tool.ListGrepArgs{
		Path:    dir,
		List:    true,
		Pattern: "keep.txt",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := textOf(t, res)
	if !strings.Contains(got, "columns=") {
		t.Fatalf("want columns= in detailed meta: %s", got)
	}
	if !strings.Contains(got, "|keep.txt|") {
		t.Fatalf("want keep row: %s", got)
	}
	if strings.Contains(got, "|drop.txt|") {
		t.Fatalf("drop should be filtered: %s", got)
	}
}

func TestListGrepLiteralVsExtended(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"foo.bar", "fooXbar"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	lit, _, err := tool.ListGrep(context.Background(), nil, tool.ListGrepArgs{
		Path: dir, Pattern: "foo.bar",
	})
	if err != nil {
		t.Fatal(err)
	}
	gotLit := textOf(t, lit)
	if !strings.Contains(gotLit, "|foo.bar|") || strings.Contains(gotLit, "|fooXbar|") {
		t.Fatalf("literal: %s", gotLit)
	}

	ext, _, err := tool.ListGrep(context.Background(), nil, tool.ListGrepArgs{
		Path: dir, Pattern: "foo.bar", Extended: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	gotExt := textOf(t, ext)
	if !strings.Contains(gotExt, "|foo.bar|") || !strings.Contains(gotExt, "|fooXbar|") {
		t.Fatalf("extended: %s", gotExt)
	}
}

func TestListGrepDeniedPath(t *testing.T) {
	res, _, err := tool.ListGrep(context.Background(), nil, tool.ListGrepArgs{
		Path: "/etc/shadow", Pattern: "x",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError || !strings.Contains(textOf(t, res), "path_denied") {
		t.Fatalf("got %v %s", res.IsError, textOf(t, res))
	}
}

func TestListGrepTruncatedBaseListing(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < policy.MaxListEntries+50; i++ {
		name := filepath.Join(dir, "f"+strconv.Itoa(i)+".txt")
		if err := os.WriteFile(name, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	res, _, err := tool.ListGrep(context.Background(), nil, tool.ListGrepArgs{
		Path: dir, Pattern: ".txt",
	})
	if err != nil {
		t.Fatal(err)
	}
	meta := strings.SplitN(textOf(t, res), "\n", 2)[0]
	if !strings.Contains(meta, "truncated=true") {
		t.Fatalf("want truncated from base listing: %s", meta)
	}
}

func TestListGrepMarkdownHeadersAreNotMatches(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ok"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Pattern that would match the markdown separator if it were treated as data.
	res, _, err := tool.ListGrep(context.Background(), nil, tool.ListGrepArgs{
		Path: dir, Pattern: "---",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := textOf(t, res)
	lines := strings.Split(got, "\n")
	// meta + |File| + |---| + maybe data; no extra |---| data row for "---" alone.
	dataRows := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "|") && line != "|File|" && line != "|---|" {
			dataRows++
		}
	}
	if dataRows != 0 {
		t.Fatalf("separator/header must not count as matches: %s", got)
	}
}
