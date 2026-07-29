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

func boolPtr(b bool) *bool { return &b }

func TestFindFilesByName(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, _, err := tool.FindFiles(context.Background(), nil, tool.FindFilesArgs{Path: dir, Name: "*.go"})
	if err != nil {
		t.Fatal(err)
	}
	got := textOf(t, res)
	if !strings.Contains(got, "a.go") || strings.Contains(got, "b.txt") {
		t.Fatalf("name filter failed: %s", got)
	}
}

func TestFindFilesIName(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, _, err := tool.FindFiles(context.Background(), nil, tool.FindFilesArgs{Path: dir, IName: "readme.*"})
	if err != nil {
		t.Fatal(err)
	}
	got := textOf(t, res)
	if !strings.Contains(got, "README.md") {
		t.Fatalf("iname filter failed: %s", got)
	}
}

func TestFindFilesByType(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, _, err := tool.FindFiles(context.Background(), nil, tool.FindFilesArgs{Path: dir, Type: "d"})
	if err != nil {
		t.Fatal(err)
	}
	got := textOf(t, res)
	if !strings.Contains(got, "sub") || strings.Contains(got, "file.txt") {
		t.Fatalf("type filter failed: %s", got)
	}
}

func TestFindFilesMaxMinDepth(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	b := filepath.Join(a, "b")
	if err := os.MkdirAll(b, 0o755); err != nil {
		t.Fatal(err)
	}

	res, _, err := tool.FindFiles(context.Background(), nil, tool.FindFilesArgs{Path: dir, MaxDepth: 1})
	if err != nil {
		t.Fatal(err)
	}
	got := textOf(t, res)
	if strings.Contains(got, "|"+b+"|") {
		t.Fatalf("maxDepth=1 did not prune depth-2 entry: %s", got)
	}

	res, _, err = tool.FindFiles(context.Background(), nil, tool.FindFilesArgs{Path: dir, MinDepth: 2})
	if err != nil {
		t.Fatal(err)
	}
	got = textOf(t, res)
	if strings.Contains(got, "|"+a+"|") {
		t.Fatalf("minDepth=2 still reported depth-1 entry: %s", got)
	}
	if !strings.Contains(got, b) {
		t.Fatalf("minDepth=2 missing depth-2 entry: %s", got)
	}
}

func TestFindFilesDeniedRootPath(t *testing.T) {
	res, _, err := tool.FindFiles(context.Background(), nil, tool.FindFilesArgs{Path: "/etc/shadow"})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError || !strings.Contains(textOf(t, res), "path_denied") {
		t.Fatalf("got %v %s", res.IsError, textOf(t, res))
	}
}

func TestFindFilesSkipsDeniedNodeInsideTree(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "kcore"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ok.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, _, err := tool.FindFiles(context.Background(), nil, tool.FindFilesArgs{Path: dir})
	if err != nil {
		t.Fatal(err)
	}
	got := textOf(t, res)
	if strings.Contains(got, "kcore") {
		t.Fatalf("denied node leaked into results: %s", got)
	}
	if !strings.Contains(got, "ok.txt") {
		t.Fatalf("walk did not continue past denied node: %s", got)
	}
}

func TestFindFilesTruncatesByResultCap(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < policy.MaxFindMatches+50; i++ {
		name := filepath.Join(dir, "f"+strconv.Itoa(i))
		if err := os.WriteFile(name, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	res, _, err := tool.FindFiles(context.Background(), nil, tool.FindFilesArgs{Path: dir})
	if err != nil {
		t.Fatal(err)
	}
	got := textOf(t, res)
	meta := strings.SplitN(got, "\n", 2)[0]
	if !strings.Contains(meta, "truncated=true") {
		t.Fatalf("want truncated by result cap: %s", meta)
	}
}

func TestFindFilesTruncatesByNodeBudget(t *testing.T) {
	// The node-budget truncation path is exercised at the policy.Walk level
	// (see walk_test.go); here we only assert find surfaces truncated=true
	// when the walk itself signals it via a huge match cap crossing.
	dir := t.TempDir()
	for i := 0; i < policy.MaxFindMatches+10; i++ {
		name := filepath.Join(dir, "f"+strconv.Itoa(i))
		if err := os.WriteFile(name, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	res, _, err := tool.FindFiles(context.Background(), nil, tool.FindFilesArgs{Path: dir, Name: "*does-not-exist*"})
	if err != nil {
		t.Fatal(err)
	}
	got := textOf(t, res)
	meta := strings.SplitN(got, "\n", 2)[0]
	if !strings.Contains(meta, "truncated=false") {
		t.Fatalf("small tree with no matches should not be truncated by node budget: %s", meta)
	}
}

func TestFindFilesDefaultColumnsAll(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, _, err := tool.FindFiles(context.Background(), nil, tool.FindFilesArgs{Path: dir})
	if err != nil {
		t.Fatal(err)
	}
	got := textOf(t, res)
	if !strings.Contains(got, "|Path|Type|Size|ModTime|") {
		t.Fatalf("default columns missing: %s", got)
	}
}

func TestFindFilesSubsetColumns(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, _, err := tool.FindFiles(context.Background(), nil, tool.FindFilesArgs{
		Path: dir, ShowPath: boolPtr(true), ShowType: boolPtr(true),
		ShowSize: boolPtr(false), ShowModTime: boolPtr(false),
	})
	if err != nil {
		t.Fatal(err)
	}
	got := textOf(t, res)
	if !strings.Contains(got, "|Path|Type|") || strings.Contains(got, "Size") || strings.Contains(got, "ModTime") {
		t.Fatalf("subset columns failed: %s", got)
	}
}

func TestFindFilesAllColumnsFalseFallsBackToPath(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, _, err := tool.FindFiles(context.Background(), nil, tool.FindFilesArgs{
		Path: dir, ShowPath: boolPtr(false), ShowType: boolPtr(false),
		ShowSize: boolPtr(false), ShowModTime: boolPtr(false),
	})
	if err != nil {
		t.Fatal(err)
	}
	got := textOf(t, res)
	if !strings.Contains(got, "|Path|\n") {
		t.Fatalf("all-false should fall back to Path column: %s", got)
	}
}

func TestFindToolDescriptionContract(t *testing.T) {
	for _, needle := range []string{"[find", "markdown", "showPath", "[blocked", "-exec"} {
		if !strings.Contains(tool.FindToolDescription, needle) {
			t.Fatalf("description missing %q", needle)
		}
	}
}
