package handler

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/KaribuLab/linux-mcp/internal/token"
	"github.com/modelcontextprotocol/go-sdk/auth"
)

const (
	testAddr       = "127.0.0.1:5000"
	initializeBody = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":` +
		`{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`
)

func testSigner(t *testing.T) *token.Signer {
	t.Helper()
	signer, err := token.NewSigner("http://" + testAddr)
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}
	return signer
}

func newTestHandler(t *testing.T, cfg Config) http.Handler {
	t.Helper()
	if cfg.Addr == "" {
		cfg.Addr = testAddr
	}
	handler, err := NewHandler(cfg)
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	return handler
}

func issue(t *testing.T, signer *token.Signer, ttl time.Duration) string {
	t.Helper()
	raw, _, err := signer.Issue("maria", 1002, ttl)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	return raw
}

// call sends an initialize request, with the bearer token when one is given.
func call(t *testing.T, handler http.Handler, bearer string) *http.Response {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(initializeBody))
	req.Host = testAddr
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	return recorder.Result()
}

func TestValidTokenReachesTheMCPEndpoint(t *testing.T) {
	signer := testSigner(t)
	handler := newTestHandler(t, Config{Verifier: signer})

	resp := call(t, handler, issue(t, signer, time.Hour))
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if !strings.Contains(string(body), "serverInfo") {
		t.Errorf("response does not look like an MCP initialize result: %s", body)
	}
}

func TestRequestWithoutTokenIsRejected(t *testing.T) {
	resp := call(t, newTestHandler(t, Config{Verifier: testSigner(t)}), "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
	// Without an authorization server to point at, the challenge must not
	// advertise resource metadata that does not exist.
	if challenge := resp.Header.Get("WWW-Authenticate"); strings.Contains(challenge, "resource_metadata") {
		t.Errorf("WWW-Authenticate advertises OAuth metadata: %q", challenge)
	}
}

func TestExpiredTokenIsRejected(t *testing.T) {
	signer := testSigner(t)
	handler := newTestHandler(t, Config{Verifier: signer})

	resp := call(t, handler, issue(t, signer, -time.Minute))
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestTokenFromAnotherServerIsRejected(t *testing.T) {
	// A fresh signer stands in for the server after a restart: the in-memory
	// key is gone and previously issued tokens must stop working.
	previous := testSigner(t)
	handler := newTestHandler(t, Config{Verifier: testSigner(t)})

	resp := call(t, handler, issue(t, previous, time.Hour))
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestTamperedTokenIsRejected(t *testing.T) {
	signer := testSigner(t)
	handler := newTestHandler(t, Config{Verifier: signer})

	raw := issue(t, signer, time.Hour)
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		t.Fatalf("unexpected token shape: %d parts", len(parts))
	}
	payload := []byte(parts[1])
	payload[0] ^= 'a' ^ 'b'

	resp := call(t, handler, parts[0]+"."+string(payload)+"."+parts[2])
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestTokenWithoutRequiredScopeIsRejected(t *testing.T) {
	// The signer only ever grants mcp:read, so the scope requirement is
	// exercised with a verifier that returns a different scope.
	stub := func(context.Context, string, *http.Request) (*auth.TokenInfo, error) {
		return &auth.TokenInfo{
			Scopes:     []string{"mcp:write"},
			Expiration: time.Now().Add(time.Hour),
			UserID:     "maria",
		}, nil
	}
	handler := auth.RequireBearerToken(stub, bearerOptions())(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))

	resp := call(t, handler, "any-token")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
}

func TestVerifierExposesIdentityAndCorrelationID(t *testing.T) {
	signer := testSigner(t)
	raw, grant, err := signer.Issue("maria", 1002, time.Hour)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	info, err := tokenVerifier(signer)(context.Background(), raw, nil)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if info.UserID != "maria" {
		t.Errorf("UserID = %q, want %q; session binding depends on it", info.UserID, "maria")
	}
	if len(info.Scopes) != 1 || info.Scopes[0] != token.ScopeRead {
		t.Errorf("Scopes = %v, want [%s]", info.Scopes, token.ScopeRead)
	}
	if info.Expiration.IsZero() {
		t.Error("Expiration is zero; the SDK rejects tokens without one")
	}
	if info.Extra[jtiKey] != grant.ID {
		t.Errorf("jti = %v, want %q", info.Extra[jtiKey], grant.ID)
	}
}

func TestVerifierReportsInvalidTokensAsUnauthenticated(t *testing.T) {
	_, err := tokenVerifier(testSigner(t))(context.Background(), "garbage", nil)
	if err == nil {
		t.Fatal("garbage token accepted")
	}
	// The SDK maps this error to 401; anything else becomes a 500.
	if !strings.Contains(err.Error(), auth.ErrInvalidToken.Error()) {
		t.Errorf("error = %v, want it to wrap ErrInvalidToken", err)
	}
}
