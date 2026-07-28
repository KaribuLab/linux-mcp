package handler

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// browserCall sends a request that looks like it came from a browser, with an
// optional Origin and a valid token, and reports the response plus the log.
func browserCall(t *testing.T, origins []string, origin, host string) (*http.Response, string) {
	t.Helper()

	parsed, err := ParseOrigins(origins)
	if err != nil {
		t.Fatalf("ParseOrigins(%v): %v", origins, err)
	}
	var logged bytes.Buffer
	signer := testSigner(t)
	handler := newTestHandler(t, Config{
		Verifier: signer,
		Origins:  parsed,
		Logger:   slog.New(slog.NewTextHandler(&logged, nil)),
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(initializeBody))
	req.Host = host
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Authorization", "Bearer "+issue(t, signer, time.Hour))
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	return recorder.Result(), logged.String()
}

func TestUnknownOriginRejected(t *testing.T) {
	resp, logged := browserCall(t, []string{"http://localhost:6274"}, "http://evil.example", testAddr)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
	if allowed := resp.Header.Get("Access-Control-Allow-Origin"); allowed != "" {
		t.Errorf("Access-Control-Allow-Origin = %q, want it absent", allowed)
	}
	if !strings.Contains(logged, "http://evil.example") {
		t.Errorf("rejected origin missing from the log: %s", logged)
	}
}

func TestDefaultPolicyRejectsEveryOrigin(t *testing.T) {
	resp, _ := browserCall(t, nil, "http://localhost:6274", testAddr)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; the default must allow no browser origin", resp.StatusCode)
	}
}

func TestAllowedOriginEchoed(t *testing.T) {
	resp, _ := browserCall(t, []string{"http://localhost:6274"}, "http://localhost:6274", testAddr)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if allowed := resp.Header.Get("Access-Control-Allow-Origin"); allowed != "http://localhost:6274" {
		t.Errorf("Access-Control-Allow-Origin = %q, want the request origin", allowed)
	}
	if vary := resp.Header.Get("Vary"); !strings.Contains(vary, "Origin") {
		t.Errorf("Vary = %q, want it to include Origin", vary)
	}
}

func TestCommaSeparatedOriginsBothAllowed(t *testing.T) {
	// The flag splits on commas, so the handler receives one value per origin.
	origins := []string{"http://localhost:6274", "https://app.example.com"}
	for _, origin := range origins {
		resp, _ := browserCall(t, origins, origin, testAddr)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("origin %q: status = %d, want 200", origin, resp.StatusCode)
		}
	}
}

func TestRequestWithoutOriginIsServed(t *testing.T) {
	// A non-browser client, which is how the Inspector proxy and every CLI
	// agent reach the server.
	resp, _ := browserCall(t, nil, "", testAddr)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestLoopbackPortWildcardMatchesAnyPort(t *testing.T) {
	for _, origin := range []string{"http://127.0.0.1:41235", "http://127.0.0.1:6274"} {
		resp, _ := browserCall(t, []string{"http://127.0.0.1:*"}, origin, testAddr)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("origin %q: status = %d, want 200", origin, resp.StatusCode)
		}
	}
}

func TestLoopbackPortWildcardStillRejectsExternalHosts(t *testing.T) {
	resp, _ := browserCall(t, []string{"http://127.0.0.1:*"}, "http://evil.example:41235", testAddr)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
}

func TestParseOriginsRejectsUnenforceableValues(t *testing.T) {
	cases := map[string]string{
		"host wildcard":                "http://*",
		"host wildcard with port":      "http://*.example.com:8080",
		"bare wildcard":                "*",
		"wildcard port on public host": "https://app.example.com:*",
		"missing scheme":               "localhost:6274",
		"unsupported scheme":           "ftp://localhost:6274",
		"origin with path":             "http://localhost:6274/mcp",
	}
	for name, value := range cases {
		if _, err := ParseOrigins([]string{value}); err == nil {
			t.Errorf("%s: ParseOrigins(%q) succeeded, want a startup error", name, value)
		}
	}
}

func TestParseOriginsAcceptsLoopbackPortWildcard(t *testing.T) {
	for _, value := range []string{"http://localhost:*", "http://127.0.0.1:*", "http://[::1]:*"} {
		if _, err := ParseOrigins([]string{value}); err != nil {
			t.Errorf("ParseOrigins(%q): %v", value, err)
		}
	}
}

func TestExternalHostHeaderRejected(t *testing.T) {
	// DNS rebinding: the browser considers this same-origin, so CORS never
	// runs and only the Host header reveals the attack.
	resp, logged := browserCall(t, []string{"http://localhost:6274"}, "", "attacker.example:5000")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
	if !strings.Contains(logged, "attacker.example") {
		t.Errorf("rejected host missing from the log: %s", logged)
	}
}

func TestLoopbackHostHeadersAccepted(t *testing.T) {
	for _, host := range []string{"127.0.0.1:5000", "localhost:5000", "[::1]:5000"} {
		resp, _ := browserCall(t, nil, "", host)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("host %q: status = %d, want 200", host, resp.StatusCode)
		}
	}
}

func TestHostOnAnotherPortRejected(t *testing.T) {
	resp, _ := browserCall(t, nil, "", "127.0.0.1:9999")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
}

func TestPreflightSucceedsWithoutToken(t *testing.T) {
	origins, err := ParseOrigins([]string{"http://localhost:6274"})
	if err != nil {
		t.Fatalf("ParseOrigins: %v", err)
	}
	handler := newTestHandler(t, Config{Verifier: testSigner(t), Origins: origins})

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Host = testAddr
	req.Header.Set("Origin", "http://localhost:6274")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "authorization, content-type")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	resp := recorder.Result()
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		t.Fatal("preflight answered 401; the browser would report a CORS failure")
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		t.Fatalf("status = %d, want a success status", resp.StatusCode)
	}
	if headers := resp.Header.Get("Access-Control-Allow-Headers"); !strings.Contains(strings.ToLower(headers), "authorization") {
		t.Errorf("Access-Control-Allow-Headers = %q, want it to include Authorization", headers)
	}
}
