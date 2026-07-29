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

func TestListFilesMetaAndTable(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".hidden"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, _, err := tool.ListFiles(context.Background(), nil, tool.ListFilesArgs{Path: dir, All: false, List: false})
	if err != nil {
		t.Fatal(err)
	}
	got := textOf(t, res)
	if !strings.HasPrefix(got, "[list path=") {
		t.Fatalf("missing meta: %s", got)
	}
	if !strings.Contains(got, "|File|") || !strings.Contains(got, "|a.txt|") {
		t.Fatalf("missing table: %s", got)
	}
	if strings.Contains(got, ".hidden") {
		t.Fatalf("hidden shown without all=true: %s", got)
	}
}

func TestListFilesDeniesShadow(t *testing.T) {
	res, _, err := tool.ListFiles(context.Background(), nil, tool.ListFilesArgs{Path: "/etc/shadow"})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError || !strings.Contains(textOf(t, res), "path_denied") {
		t.Fatalf("got %v %s", res.IsError, textOf(t, res))
	}
}

func TestListFilesTruncates(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < policy.MaxListEntries+50; i++ {
		name := filepath.Join(dir, "f"+strconv.Itoa(i))
		if err := os.WriteFile(name, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	res, _, err := tool.ListFiles(context.Background(), nil, tool.ListFilesArgs{Path: dir})
	if err != nil {
		t.Fatal(err)
	}
	got := textOf(t, res)
	meta := strings.SplitN(got, "\n", 2)[0]
	if !strings.Contains(meta, "truncated=true") {
		t.Fatalf("want truncated: %s", meta)
	}
}

func TestListFilesSymlinkJoin(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	if err := os.WriteFile(target, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	other := t.TempDir()
	if err := os.Chdir(other); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	res, _, err := tool.ListFiles(context.Background(), nil, tool.ListFilesArgs{Path: dir, List: true})
	if err != nil {
		t.Fatal(err)
	}
	got := textOf(t, res)
	if !strings.Contains(got, "link.txt") || !strings.Contains(got, target) {
		t.Fatalf("symlink path missing: %s", got)
	}
}

func TestListToolDescriptionContract(t *testing.T) {
	for _, needle := range []string{"[list", "markdown", "|File|", "[blocked", "1000"} {
		if !strings.Contains(tool.ListToolDescription, needle) {
			t.Fatalf("description missing %q", needle)
		}
	}
}

func boolPtr(b bool) *bool { return &b }

func TestListFilesDefaultColumnsUnchanged(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, _, err := tool.ListFiles(context.Background(), nil, tool.ListFilesArgs{Path: dir, List: true})
	if err != nil {
		t.Fatal(err)
	}
	got := textOf(t, res)
	if !strings.Contains(got, "|Name|Size|Mode|Owner|Group|ModTime|IsDir|IsSymlink|SymlinkPath|") {
		t.Fatalf("missing default header: %s", got)
	}
	if !strings.Contains(got, "columns=Name,Size,Mode,Owner,Group,ModTime,IsDir,IsSymlink,SymlinkPath") {
		t.Fatalf("missing default columns in meta: %s", got)
	}
}

func TestListFilesHidesSelectedColumns(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, _, err := tool.ListFiles(context.Background(), nil, tool.ListFilesArgs{
		Path: dir, List: true,
		ShowSize:    boolPtr(false),
		ShowModTime: boolPtr(false),
	})
	if err != nil {
		t.Fatal(err)
	}
	got := textOf(t, res)
	if !strings.Contains(got, "|Name|Mode|Owner|Group|IsDir|IsSymlink|SymlinkPath|") {
		t.Fatalf("missing filtered header: %s", got)
	}
	if !strings.Contains(got, "columns=Name,Mode,Owner,Group,IsDir,IsSymlink,SymlinkPath") {
		t.Fatalf("missing filtered columns in meta: %s", got)
	}
}

func TestListFilesExplicitTrueEqualsDefault(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, _, err := tool.ListFiles(context.Background(), nil, tool.ListFilesArgs{Path: dir, List: true, ShowSize: boolPtr(true)})
	if err != nil {
		t.Fatal(err)
	}
	got := textOf(t, res)
	if !strings.Contains(got, "|Name|Size|Mode|Owner|Group|ModTime|IsDir|IsSymlink|SymlinkPath|") {
		t.Fatalf("explicit true should match default header: %s", got)
	}
}

func TestListFilesAllColumnsHiddenKeepsName(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, _, err := tool.ListFiles(context.Background(), nil, tool.ListFilesArgs{
		Path: dir, List: true,
		ShowSize:        boolPtr(false),
		ShowMode:        boolPtr(false),
		ShowOwner:       boolPtr(false),
		ShowGroup:       boolPtr(false),
		ShowModTime:     boolPtr(false),
		ShowIsDir:       boolPtr(false),
		ShowIsSymlink:   boolPtr(false),
		ShowSymlinkPath: boolPtr(false),
	})
	if err != nil {
		t.Fatal(err)
	}
	got := textOf(t, res)
	if !strings.Contains(got, "|Name|\n") {
		t.Fatalf("expected Name-only header: %s", got)
	}
	if !strings.Contains(got, "columns=Name") || strings.Contains(got, "columns=Name,") {
		t.Fatalf("expected columns=Name only: %s", got)
	}
	if !strings.Contains(got, "|a.txt|\n") {
		t.Fatalf("expected Name-only row: %s", got)
	}
}

func TestListFilesShowFlagsIgnoredInSimpleMode(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, _, err := tool.ListFiles(context.Background(), nil, tool.ListFilesArgs{Path: dir, List: false, ShowSize: boolPtr(false)})
	if err != nil {
		t.Fatal(err)
	}
	got := textOf(t, res)
	if !strings.Contains(got, "|File|\n") || !strings.Contains(got, "|a.txt|\n") {
		t.Fatalf("simple mode table changed: %s", got)
	}
	if strings.Contains(got, "columns=") {
		t.Fatalf("simple mode meta must not include columns field: %s", got)
	}
}
