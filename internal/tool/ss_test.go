package tool

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/KaribuLab/linux-mcp/internal/policy"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func withFakeSockets(t *testing.T, rows []socketRow) {
	t.Helper()
	old := socketLister
	socketLister = func(state, family string) ([]socketRow, error) {
		var out []socketRow
		for _, r := range rows {
			if family == "inet4" && r.Family != "inet4" {
				continue
			}
			if family == "inet6" && r.Family != "inet6" {
				continue
			}
			if family == "unix" && r.Family != "unix" {
				continue
			}
			if family == "inet" && (r.Family != "inet4" && r.Family != "inet6") {
				continue
			}
			if state == "LISTEN" && r.State != "LISTEN" && r.Proto != "udp" {
				continue
			}
			if state == "ESTAB" && r.State != "ESTAB" {
				continue
			}
			out = append(out, r)
		}
		return out, nil
	}
	t.Cleanup(func() { socketLister = old })
}

func TestSsDefaultListen(t *testing.T) {
	withFakeSockets(t, []socketRow{
		{socketInfo: socketInfo{Proto: "tcp", Local: "0.0.0.0:22", Peer: "0.0.0.0:0", State: "LISTEN", Family: "inet4"}, Inode: 1},
		{socketInfo: socketInfo{Proto: "tcp", Local: "10.0.0.1:12345", Peer: "1.2.3.4:443", State: "ESTAB", Family: "inet4"}, Inode: 2},
	})
	res, _, err := Ss(context.Background(), nil, SsArgs{})
	if err != nil {
		t.Fatal(err)
	}
	text := res.Content[0].(*mcp.TextContent).Text
	if !strings.HasPrefix(text, "[ss ") {
		t.Fatalf("meta: %s", text)
	}
	if !strings.Contains(text, "0.0.0.0:22") {
		t.Fatalf("expected listen: %s", text)
	}
	if strings.Contains(text, "1.2.3.4:443") {
		t.Fatalf("ESTAB should be omitted by default: %s", text)
	}
}

func TestSsWidenEstab(t *testing.T) {
	withFakeSockets(t, []socketRow{
		{socketInfo: socketInfo{Proto: "tcp", Local: "10.0.0.1:12345", Peer: "1.2.3.4:443", State: "ESTAB", Family: "inet4"}, Inode: 2},
	})
	res, _, err := Ss(context.Background(), nil, SsArgs{State: "ESTAB"})
	if err != nil {
		t.Fatal(err)
	}
	text := res.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "1.2.3.4:443") {
		t.Fatalf("expected estab: %s", text)
	}
}

func TestSsHidePeer(t *testing.T) {
	withFakeSockets(t, []socketRow{
		{socketInfo: socketInfo{Proto: "tcp", Local: "127.0.0.1:80", Peer: "0.0.0.0:0", State: "LISTEN", Family: "inet4"}, Inode: 1},
	})
	f := false
	res, _, err := Ss(context.Background(), nil, SsArgs{ShowPeer: &f})
	if err != nil {
		t.Fatal(err)
	}
	text := res.Content[0].(*mcp.TextContent).Text
	header := strings.Split(text, "\n")[1]
	if strings.Contains(header, "Peer") {
		t.Fatalf("Peer should be hidden: %s", header)
	}
	if !strings.Contains(header, "Proto") || !strings.Contains(header, "Local") {
		t.Fatalf("identity required: %s", header)
	}
}

func TestSsInvalidState(t *testing.T) {
	_, _, err := Ss(context.Background(), nil, SsArgs{State: "wat"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSsTruncates(t *testing.T) {
	rows := make([]socketRow, 0, policy.MaxListEntries+3)
	for i := 0; i < policy.MaxListEntries+3; i++ {
		rows = append(rows, socketRow{socketInfo: socketInfo{
			Proto: "tcp", Local: "0.0.0.0:" + strconv.Itoa(i+1), Peer: "0.0.0.0:0", State: "LISTEN", Family: "inet4",
		}, Inode: uint32(i + 1)})
	}
	withFakeSockets(t, rows)
	res, _, err := Ss(context.Background(), nil, SsArgs{})
	if err != nil {
		t.Fatal(err)
	}
	text := res.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "truncated=true") {
		t.Fatalf("expected truncated: %s", text)
	}
}
