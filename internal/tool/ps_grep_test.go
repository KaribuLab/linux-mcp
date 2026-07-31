package tool

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestPsGrepFiltersByCommGlob(t *testing.T) {
	withFakeProc(t, func(root string) {
		writeFakeProc(t, root, 1, "nginx", "S", 0, "nginx: master", 100, 0)
		writeFakeProc(t, root, 2, "sshd", "S", 0, "/usr/sbin/sshd", 50, 0)
	})

	res, _, err := PsGrep(context.Background(), nil, PsGrepArgs{Pattern: "nginx*"})
	if err != nil {
		t.Fatal(err)
	}
	text := res.Content[0].(*mcp.TextContent).Text
	if !strings.HasPrefix(text, "[ps_grep ") {
		t.Fatalf("meta: %s", text)
	}
	if !strings.Contains(text, "nginx") {
		t.Fatalf("expected nginx: %s", text)
	}
	if strings.Contains(text, "sshd") {
		t.Fatalf("sshd should be filtered: %s", text)
	}
}

func TestPsGrepLiteralCmdline(t *testing.T) {
	withFakeProc(t, func(root string) {
		writeFakeProc(t, root, 1, "python", "S", 0, "python app.py --debug", 10, 0)
		writeFakeProc(t, root, 2, "python", "S", 0, "python worker.py", 10, 0)
	})
	res, _, err := PsGrep(context.Background(), nil, PsGrepArgs{Pattern: "--debug"})
	if err != nil {
		t.Fatal(err)
	}
	text := res.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "--debug") {
		t.Fatalf("expected match: %s", text)
	}
	if strings.Count(text, "|python|") != 1 {
		t.Fatalf("want one python row: %s", text)
	}
}

func TestPsGrepShowFlags(t *testing.T) {
	withFakeProc(t, func(root string) {
		writeFakeProc(t, root, 7, "redis", "S", 1, "redis-server", 99, 0)
	})
	f := false
	res, _, err := PsGrep(context.Background(), nil, PsGrepArgs{Pattern: "redis", ShowMem: &f})
	if err != nil {
		t.Fatal(err)
	}
	text := res.Content[0].(*mcp.TextContent).Text
	header := strings.Split(text, "\n")[1]
	if strings.Contains(header, "Mem") {
		t.Fatalf("Mem should be hidden: %s", header)
	}
	if !strings.Contains(header, "Pid") || !strings.Contains(header, "Comm") {
		t.Fatalf("identity required: %s", header)
	}
}
