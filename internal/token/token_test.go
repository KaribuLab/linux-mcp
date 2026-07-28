package token

import (
	"errors"
	"strings"
	"testing"
	"time"
)

const testAudience = "http://127.0.0.1:5000"

func newTestSigner(t *testing.T, audience string) *Signer {
	t.Helper()
	signer, err := NewSigner(audience)
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}
	return signer
}

func TestIssueRoundTrip(t *testing.T) {
	signer := newTestSigner(t, testAudience)

	raw, issued, err := signer.Issue("maria", 1002, time.Hour)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	verified, err := signer.Verify(raw)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if verified.Subject != "maria" {
		t.Errorf("subject = %q, want %q", verified.Subject, "maria")
	}
	if verified.UID != 1002 {
		t.Errorf("uid = %d, want 1002", verified.UID)
	}
	if verified.Scope != ScopeRead {
		t.Errorf("scope = %q, want %q", verified.Scope, ScopeRead)
	}
	if verified.ID != issued.ID {
		t.Errorf("jti = %q, want %q", verified.ID, issued.ID)
	}
	if verified.ExpiresAt.IsZero() {
		t.Error("expiration must always be set")
	}
}

func TestIssueGeneratesUniqueIDs(t *testing.T) {
	signer := newTestSigner(t, testAudience)

	_, first, err := signer.Issue("maria", 1002, time.Hour)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	_, second, err := signer.Issue("maria", 1002, time.Hour)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if first.ID == second.ID {
		t.Error("two tokens share the same jti; audit records could not be told apart")
	}
}

func TestVerifyRejectsExpiredToken(t *testing.T) {
	signer := newTestSigner(t, testAudience)

	raw, _, err := signer.Issue("maria", 1002, -time.Minute)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	if _, err := signer.Verify(raw); !errors.Is(err, ErrInvalid) {
		t.Fatalf("Verify error = %v, want ErrInvalid", err)
	}
}

func TestVerifyRejectsForeignAudience(t *testing.T) {
	issuing := newTestSigner(t, "http://127.0.0.1:5000")
	raw, _, err := issuing.Issue("maria", 1002, time.Hour)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	// Same key, different resource: the MCP spec requires rejecting tokens
	// that were not minted for this server.
	other := &Signer{key: issuing.key, audience: "http://127.0.0.1:9999", jose: issuing.jose}
	if _, err := other.Verify(raw); !errors.Is(err, ErrInvalid) {
		t.Fatalf("Verify error = %v, want ErrInvalid", err)
	}
}

func TestVerifyRejectsTokenFromAnotherKey(t *testing.T) {
	raw, _, err := newTestSigner(t, testAudience).Issue("maria", 1002, time.Hour)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	// A fresh signer stands in for the server after a restart.
	if _, err := newTestSigner(t, testAudience).Verify(raw); !errors.Is(err, ErrInvalid) {
		t.Fatalf("Verify error = %v, want ErrInvalid", err)
	}
}

func TestVerifyRejectsTamperedPayload(t *testing.T) {
	signer := newTestSigner(t, testAudience)
	raw, _, err := signer.Issue("maria", 1002, time.Hour)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		t.Fatalf("unexpected token shape: %d parts", len(parts))
	}
	// Flip a byte of the payload while keeping the original signature.
	payload := []byte(parts[1])
	payload[0] ^= 'a' ^ 'b'
	tampered := parts[0] + "." + string(payload) + "." + parts[2]

	if _, err := signer.Verify(tampered); !errors.Is(err, ErrInvalid) {
		t.Fatalf("Verify error = %v, want ErrInvalid", err)
	}
}

func TestVerifyRejectsGarbage(t *testing.T) {
	signer := newTestSigner(t, testAudience)

	if _, err := signer.Verify("not-a-token"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("Verify error = %v, want ErrInvalid", err)
	}
}
