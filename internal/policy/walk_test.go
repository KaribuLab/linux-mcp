package policy_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/KaribuLab/linux-mcp/internal/policy"
)

func TestWalkSkipsDeniedNodeAndContinues(t *testing.T) {
	dir := t.TempDir()
	// "kcore" matches the basename denylist regardless of directory.
	if err := os.WriteFile(filepath.Join(dir, "kcore"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ok.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	var seen []string
	_, truncated, err := policy.Walk(dir, policy.WalkLimits{}, 0, 0, func(path string, info os.FileInfo, depth int) error {
		seen = append(seen, filepath.Base(path))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if truncated {
		t.Fatal("did not expect truncation")
	}
	for _, name := range seen {
		if name == "kcore" {
			t.Fatalf("denied node reported: %v", seen)
		}
	}
	found := false
	for _, name := range seen {
		if name == "ok.txt" {
			found = true
		}
	}
	if !found {
		t.Fatalf("walk stopped instead of continuing past denied node: %v", seen)
	}
}

func TestWalkDoesNotDescendSymlinkToDirectory(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "file.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(sub, link); err != nil {
		t.Fatal(err)
	}

	visited, _, err := policy.Walk(dir, policy.WalkLimits{}, 0, 0, func(path string, info os.FileInfo, depth int) error {
		if filepath.Base(filepath.Dir(path)) == "link" {
			t.Fatalf("walk descended into symlink target via link: %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// root, sub, sub/file.txt, link = 4 nodes; link's target must not be re-visited.
	if visited != 4 {
		t.Fatalf("visited = %d, want 4", visited)
	}
}

func TestWalkStopsAtNodeBudget(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 20; i++ {
		if err := os.WriteFile(filepath.Join(dir, "f"+string(rune('a'+i))), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	visited, truncated, err := policy.Walk(dir, policy.WalkLimits{MaxNodes: 5}, 0, 0, func(path string, info os.FileInfo, depth int) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if visited != 5 {
		t.Fatalf("visited = %d, want exactly 5", visited)
	}
	if !truncated {
		t.Fatal("expected truncated=true at node budget")
	}
}

func TestWalkPrunesByMinAndMaxDepth(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	b := filepath.Join(a, "b")
	c := filepath.Join(b, "c")
	if err := os.MkdirAll(c, 0o755); err != nil {
		t.Fatal(err)
	}

	var reported []int
	_, _, err := policy.Walk(dir, policy.WalkLimits{}, 0, 1, func(path string, info os.FileInfo, depth int) error {
		reported = append(reported, depth)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range reported {
		if d > 1 {
			t.Fatalf("maxDepth=1 did not prune: reported depth %d, all depths %v", d, reported)
		}
	}
	if len(reported) != 2 { // root (depth 0) + "a" (depth 1)
		t.Fatalf("maxDepth=1 reported %d nodes, want 2: %v", len(reported), reported)
	}

	reported = nil
	_, _, err = policy.Walk(dir, policy.WalkLimits{}, 2, 0, func(path string, info os.FileInfo, depth int) error {
		reported = append(reported, depth)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range reported {
		if d < 2 {
			t.Fatalf("minDepth=2 did not filter: reported depth %d, all depths %v", d, reported)
		}
	}
	if len(reported) != 2 { // "b" (depth 2) + "c" (depth 3)
		t.Fatalf("minDepth=2 reported %d nodes, want 2: %v", len(reported), reported)
	}
}

func TestWalkDeniedRootReturnsStatError(t *testing.T) {
	_, _, err := policy.Walk("/no/such/path/at/all", policy.WalkLimits{}, 0, 0, func(string, os.FileInfo, int) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected error for nonexistent root")
	}
}
