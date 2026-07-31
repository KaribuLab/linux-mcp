package tool

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestSsGrepWildcardBind(t *testing.T) {
	withFakeSockets(t, []socketRow{
		{socketInfo: socketInfo{Proto: "tcp", Local: "0.0.0.0:3306", Peer: "0.0.0.0:0", State: "LISTEN", Family: "inet4"}, Inode: 1},
		{socketInfo: socketInfo{Proto: "tcp", Local: "127.0.0.1:3306", Peer: "0.0.0.0:0", State: "LISTEN", Family: "inet4"}, Inode: 2},
	})
	res, _, err := SsGrep(context.Background(), nil, SsGrepArgs{Pattern: "0.0.0.0:3306"})
	if err != nil {
		t.Fatal(err)
	}
	text := res.Content[0].(*mcp.TextContent).Text
	if !strings.HasPrefix(text, "[ss_grep ") {
		t.Fatalf("meta: %s", text)
	}
	if !strings.Contains(text, "0.0.0.0:3306") {
		t.Fatalf("expected wildcard bind: %s", text)
	}
	if strings.Contains(text, "127.0.0.1:3306") {
		t.Fatalf("loopback should be filtered: %s", text)
	}
}
