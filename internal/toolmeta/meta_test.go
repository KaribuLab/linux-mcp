package toolmeta_test

import (
	"strings"
	"testing"

	"github.com/KaribuLab/linux-mcp/internal/toolmeta"
)

func TestBlockedString(t *testing.T) {
	got := toolmeta.Blocked{Class: "path_denied", Path: "/etc/shadow"}.String()
	want := "[blocked class=path_denied path=/etc/shadow]"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestCatHeaderTruncated(t *testing.T) {
	got := toolmeta.CatHeader{Path: "/tmp/a", Lines: 100, Truncated: true, NextByte: 4096}.String()
	if !strings.Contains(got, "truncated=true") || !strings.Contains(got, "next=4096") {
		t.Fatalf("bad header: %s", got)
	}
}

func TestCatHeaderComplete(t *testing.T) {
	got := toolmeta.CatHeader{Path: "/tmp/a", Lines: 3, Truncated: false}.String()
	if strings.Contains(got, "next=") {
		t.Fatalf("unexpected next: %s", got)
	}
}

func TestListHeader(t *testing.T) {
	got := toolmeta.ListHeader{Path: "/tmp", Returned: 10, Total: 10, Truncated: false}.String()
	want := "[list path=/tmp entries=10/10 truncated=false]"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestRender(t *testing.T) {
	var body strings.Builder
	body.WriteString("hello\nworld")
	got := toolmeta.Render(toolmeta.CatHeader{Path: "/x", Lines: 2}, &body)
	if !strings.HasPrefix(got, "[cat path=/x lines=2 truncated=false]\nhello\nworld") {
		t.Fatalf("got %q", got)
	}
}
