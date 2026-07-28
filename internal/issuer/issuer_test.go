package issuer

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/KaribuLab/linux-mcp/internal/token"
)

const testAudience = "http://127.0.0.1:5000"

// startServer brings up an issuance socket owned by the test process, with no
// administrative group. This is exactly the local development configuration.
func startServer(t *testing.T, maxTTL time.Duration) (string, *token.Signer) {
	t.Helper()

	signer, err := token.NewSigner(testAudience)
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}
	path := filepath.Join(t.TempDir(), "issue.sock")
	listener, err := Listen(path, "")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	t.Cleanup(func() { listener.Close() })

	server := &Server{Signer: signer, MaxTTL: maxTTL}
	go func() { _ = server.Serve(listener) }()

	return path, signer
}

// request performs one issuance exchange, sending the given raw JSON body so
// tests can submit fields the Request type does not define.
func request(t *testing.T, path, body string) Response {
	t.Helper()

	conn, err := net.DialTimeout("unix", path, 2*time.Second)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))

	if _, err := conn.Write([]byte(body)); err != nil {
		t.Fatalf("write request: %v", err)
	}
	var resp Response
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp
}

func TestListenRejectsAbstractSocket(t *testing.T) {
	// Abstract sockets have no permission checks, so any process in the
	// network namespace could obtain a token.
	if _, err := Listen("@linux-mcp-test", ""); err == nil {
		t.Fatal("Listen accepted an abstract socket path")
	}
}

func TestListenCreatesOwnerOnlySocketWithoutGroup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "issue.sock")
	listener, err := Listen(path, "")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer listener.Close()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat socket: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("socket mode = %o, want 600", perm)
	}
}

func TestListenReplacesStaleSocket(t *testing.T) {
	path := filepath.Join(t.TempDir(), "issue.sock")
	first, err := Listen(path, "")
	if err != nil {
		t.Fatalf("first Listen: %v", err)
	}
	// Leave the file behind as an unclean shutdown would.
	first.(*net.UnixListener).SetUnlinkOnClose(false)
	first.Close()

	second, err := Listen(path, "")
	if err != nil {
		t.Fatalf("second Listen over stale socket: %v", err)
	}
	second.Close()
}

func TestListenRefusesToReplaceRegularFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-a-socket")
	if err := os.WriteFile(path, []byte("important"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	if _, err := Listen(path, ""); err == nil {
		t.Fatal("Listen removed a regular file instead of refusing")
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("existing file was removed: %v", err)
	}
}

func TestIssueUsesKernelIdentity(t *testing.T) {
	path, signer := startServer(t, time.Hour)

	resp := request(t, path, `{"ttl":"5m"}`)
	if resp.Error != "" {
		t.Fatalf("issuance failed: %s", resp.Error)
	}

	grant, err := signer.Verify(resp.Token)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if got, want := int(grant.UID), os.Getuid(); got != want {
		t.Errorf("uid = %d, want %d (must come from SO_PEERCRED)", got, want)
	}
	if grant.Subject != usernameForUID(uint32(os.Getuid())) {
		t.Errorf("subject = %q, want the current user", grant.Subject)
	}
}

func TestClientCannotChooseSubject(t *testing.T) {
	path, signer := startServer(t, time.Hour)

	// A client that tries to declare an identity must be ignored entirely.
	resp := request(t, path, `{"ttl":"5m","sub":"someone-else","uid":4242}`)
	if resp.Error != "" {
		t.Fatalf("issuance failed: %s", resp.Error)
	}

	grant, err := signer.Verify(resp.Token)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if grant.Subject == "someone-else" || grant.UID == 4242 {
		t.Fatalf("client-supplied identity was honoured: sub=%q uid=%d", grant.Subject, grant.UID)
	}
	if int(grant.UID) != os.Getuid() {
		t.Errorf("uid = %d, want %d", grant.UID, os.Getuid())
	}
}

func TestRequestedTTLIsCapped(t *testing.T) {
	path, signer := startServer(t, time.Hour)

	resp := request(t, path, `{"ttl":"100h"}`)
	if resp.Error != "" {
		t.Fatalf("issuance failed: %s", resp.Error)
	}
	if !resp.Capped {
		t.Error("response does not report the lifetime was capped")
	}

	grant, err := signer.Verify(resp.Token)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if remaining := time.Until(grant.ExpiresAt); remaining > time.Hour+time.Minute {
		t.Errorf("token lives for %s, want at most the 1h maximum", remaining)
	}
}

func TestShorterTTLIsHonoured(t *testing.T) {
	path, signer := startServer(t, time.Hour)

	resp := request(t, path, `{"ttl":"5m"}`)
	if resp.Capped {
		t.Error("a lifetime below the maximum must not be reported as capped")
	}

	grant, err := signer.Verify(resp.Token)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if remaining := time.Until(grant.ExpiresAt); remaining > 6*time.Minute {
		t.Errorf("token lives for %s, want about 5m", remaining)
	}
}

func TestEveryTokenExpires(t *testing.T) {
	path, signer := startServer(t, time.Hour)

	// No ttl requested: the server must still bound the token.
	resp := request(t, path, `{}`)
	if resp.Error != "" {
		t.Fatalf("issuance failed: %s", resp.Error)
	}
	grant, err := signer.Verify(resp.Token)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if grant.ExpiresAt.IsZero() {
		t.Fatal("issued token has no expiration")
	}
}

func TestInvalidTTLRejected(t *testing.T) {
	path, _ := startServer(t, time.Hour)

	for _, ttl := range []string{"nonsense", "-5m", "0s"} {
		resp := request(t, path, `{"ttl":"`+ttl+`"}`)
		if resp.Error == "" {
			t.Errorf("ttl %q was accepted", ttl)
		}
		if resp.Token != "" {
			t.Errorf("ttl %q produced a token", ttl)
		}
	}
}

func TestMalformedRequestRejected(t *testing.T) {
	path, _ := startServer(t, time.Hour)

	resp := request(t, path, `{not json`)
	if resp.Error == "" {
		t.Error("malformed request was accepted")
	}
}

func TestUsernameForUIDFallsBackToNumericID(t *testing.T) {
	// A uid with no entry in the user database must still yield a subject.
	const unlikelyUID = 4294967294
	if got := usernameForUID(unlikelyUID); got != strconv.Itoa(unlikelyUID) {
		if !strings.EqualFold(got, "nobody") {
			t.Errorf("usernameForUID(%d) = %q, want the numeric uid", unlikelyUID, got)
		}
	}
}
