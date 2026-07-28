// Package e2e exercises the whole path an operator follows: ask the issuance
// socket for a token, then use that token to call a tool over HTTP.
package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/KaribuLab/linux-mcp/internal/handler"
	"github.com/KaribuLab/linux-mcp/internal/issuer"
	"github.com/KaribuLab/linux-mcp/internal/token"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// syncBuffer collects log output written from the issuance and HTTP goroutines.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// bearerTransport attaches the token the way an MCP client would.
type bearerTransport struct {
	token string
}

func (t bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	authorized := req.Clone(req.Context())
	authorized.Header.Set("Authorization", "Bearer "+t.token)
	return http.DefaultTransport.RoundTrip(authorized)
}

type deployment struct {
	socket   string
	endpoint string
	logs     *syncBuffer
}

// start runs both listeners the service exposes, wired to a single signer just
// as the serve command does.
func start(t *testing.T) deployment {
	t.Helper()

	httpListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := httpListener.Addr().String()

	signer, err := token.NewSigner("http://" + addr)
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}
	logs := &syncBuffer{}
	logger := slog.New(slog.NewTextHandler(logs, nil))

	socket := filepath.Join(t.TempDir(), "issue.sock")
	socketListener, err := issuer.Listen(socket, "")
	if err != nil {
		t.Fatalf("issuer.Listen: %v", err)
	}
	issuance := &issuer.Server{Signer: signer, MaxTTL: time.Hour, Logger: logger}
	go func() { _ = issuance.Serve(socketListener) }()

	mcpHandler, err := handler.NewHandler(handler.Config{Verifier: signer, Addr: addr, Logger: logger})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	server := &http.Server{Handler: mcpHandler}
	go func() { _ = server.Serve(httpListener) }()

	t.Cleanup(func() {
		server.Close()
		socketListener.Close()
	})
	return deployment{socket: socket, endpoint: "http://" + addr, logs: logs}
}

// requestToken performs the same exchange as the auth command.
func (d deployment) requestToken(t *testing.T, ttl string) issuer.Response {
	t.Helper()

	conn, err := net.DialTimeout("unix", d.socket, 2*time.Second)
	if err != nil {
		t.Fatalf("dial issuance socket: %v", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))

	if err := json.NewEncoder(conn).Encode(issuer.Request{TTL: ttl}); err != nil {
		t.Fatalf("send request: %v", err)
	}
	var resp issuer.Response
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		t.Fatalf("read response: %v", err)
	}
	if resp.Error != "" {
		t.Fatalf("issuance refused: %s", resp.Error)
	}
	return resp
}

func (d deployment) connect(t *testing.T, bearer string) *mcp.ClientSession {
	t.Helper()

	client := mcp.NewClient(&mcp.Implementation{Name: "e2e-test", Version: "1"}, nil)
	transport := &mcp.StreamableClientTransport{
		Endpoint:   d.endpoint,
		HTTPClient: &http.Client{Transport: bearerTransport{token: bearer}},
	}
	session, err := client.Connect(context.Background(), transport, nil)
	if err != nil {
		t.Fatalf("connect to MCP endpoint: %v", err)
	}
	t.Cleanup(func() { session.Close() })
	return session
}

func TestIssuedTokenCanCallATool(t *testing.T) {
	deployment := start(t)
	issued := deployment.requestToken(t, "5m")
	session := deployment.connect(t, issued.Token)

	path := filepath.Join(t.TempDir(), "note.txt")
	if err := os.WriteFile(path, []byte("hello from the agent\n"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "cat",
		Arguments: map[string]any{"path": path},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool reported an error: %+v", result.Content)
	}

	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("unexpected content type %T", result.Content[0])
	}
	if !strings.Contains(text.Text, "hello from the agent") {
		t.Errorf("tool response does not contain the file body: %q", text.Text)
	}
}

func TestConnectionWithoutTokenIsRefused(t *testing.T) {
	deployment := start(t)

	client := mcp.NewClient(&mcp.Implementation{Name: "e2e-test", Version: "1"}, nil)
	transport := &mcp.StreamableClientTransport{Endpoint: deployment.endpoint}
	session, err := client.Connect(context.Background(), transport, nil)
	if err == nil {
		session.Close()
		t.Fatal("connected without a token")
	}
}

func TestTokenSubjectIsTheCallingUser(t *testing.T) {
	deployment := start(t)

	// The request carries no identity at all; the server derives it from
	// SO_PEERCRED on the socket.
	issued := deployment.requestToken(t, "5m")

	uid := strconv.Itoa(os.Getuid())
	expected := uid
	if u, err := user.LookupId(uid); err == nil {
		expected = u.Username
	}
	if issued.Subject != expected {
		t.Errorf("subject = %q, want %q", issued.Subject, expected)
	}
}

func TestAuditTrailLinksIssuanceToUsage(t *testing.T) {
	deployment := start(t)
	issued := deployment.requestToken(t, "5m")
	session := deployment.connect(t, issued.Token)

	path := filepath.Join(t.TempDir(), "note.txt")
	if err := os.WriteFile(path, []byte("audited read\n"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if _, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "cat",
		Arguments: map[string]any{"path": path},
	}); err != nil {
		t.Fatalf("CallTool: %v", err)
	}

	logs := deployment.logs.String()
	issuedJTI := findJTI(t, logs, `msg="token issued"`)
	usageJTI := findJTI(t, logs, `msg="mcp call" method=tools/call tool=cat`)

	if issuedJTI != usageJTI {
		t.Errorf("jti in the usage record (%q) does not match the issuance record (%q)", usageJTI, issuedJTI)
	}
}

// findJTI returns the jti of the first log line matching prefix.
func findJTI(t *testing.T, logs, prefix string) string {
	t.Helper()

	pattern := regexp.MustCompile(regexp.QuoteMeta(prefix) + `.*?\bjti=(\S+)`)
	match := pattern.FindStringSubmatch(logs)
	if match == nil {
		t.Fatalf("no log line with a jti matching %q in:\n%s", prefix, logs)
	}
	return match[1]
}
