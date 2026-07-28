// Package token issues and verifies the bearer tokens that guard the MCP endpoint.
//
// The signing key is generated with crypto/rand at startup and never leaves
// process memory, so restarting the service rotates it and invalidates every
// token issued before the restart. That restart is also the only revocation
// mechanism: tokens are self-contained and the server keeps no token state.
package token

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

const (
	// Issuer is the "iss" claim of every token this package mints.
	Issuer = "linux-mcp"
	// ScopeRead is the only scope granted; the MCP endpoint requires it.
	ScopeRead = "mcp:read"

	keySize = 32
	idSize  = 16
)

// ErrInvalid wraps every verification failure so callers can answer 401
// without inspecting the underlying reason.
var ErrInvalid = errors.New("invalid token")

// Grant is what an issued or verified token asserts.
type Grant struct {
	Subject   string
	UID       uint32
	ID        string
	Scope     string
	ExpiresAt time.Time
}

// privateClaims carries the fields the JWT spec does not define.
type privateClaims struct {
	UID   uint32 `json:"uid"`
	Scope string `json:"scope"`
}

// Signer mints and verifies tokens for a single audience.
type Signer struct {
	key      []byte
	audience string
	jose     jose.Signer
}

// NewSigner generates an ephemeral HS256 key. Signing and verification happen
// in the same process, so a symmetric key is the right primitive: an asymmetric
// one would only pay off if the verifier had to be unable to sign.
func NewSigner(audience string) (*Signer, error) {
	key := make([]byte, keySize)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generate signing key: %w", err)
	}
	opts := (&jose.SignerOptions{}).WithType("JWT")
	joseSigner, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.HS256, Key: key}, opts)
	if err != nil {
		return nil, fmt.Errorf("build signer: %w", err)
	}
	return &Signer{key: key, audience: audience, jose: joseSigner}, nil
}

// Issue mints a token for the given identity. The caller is responsible for
// having derived subject and uid from the kernel rather than from client input.
func (s *Signer) Issue(subject string, uid uint32, ttl time.Duration) (string, Grant, error) {
	id, err := randomID()
	if err != nil {
		return "", Grant{}, err
	}
	now := time.Now()
	expiresAt := now.Add(ttl)

	standard := jwt.Claims{
		Issuer:    Issuer,
		Subject:   subject,
		Audience:  jwt.Audience{s.audience},
		Expiry:    jwt.NewNumericDate(expiresAt),
		NotBefore: jwt.NewNumericDate(now),
		IssuedAt:  jwt.NewNumericDate(now),
		ID:        id,
	}
	private := privateClaims{UID: uid, Scope: ScopeRead}

	raw, err := jwt.Signed(s.jose).Claims(standard).Claims(private).Serialize()
	if err != nil {
		return "", Grant{}, fmt.Errorf("sign token: %w", err)
	}
	return raw, Grant{
		Subject:   subject,
		UID:       uid,
		ID:        id,
		Scope:     ScopeRead,
		ExpiresAt: expiresAt.Truncate(time.Second),
	}, nil
}

// Verify checks the signature and every registered claim. Audience validation
// is mandatory: the MCP specification requires rejecting tokens minted for a
// different resource.
func (s *Signer) Verify(raw string) (Grant, error) {
	parsed, err := jwt.ParseSigned(raw, []jose.SignatureAlgorithm{jose.HS256})
	if err != nil {
		return Grant{}, fmt.Errorf("%w: %v", ErrInvalid, err)
	}

	var standard jwt.Claims
	var private privateClaims
	if err := parsed.Claims(s.key, &standard, &private); err != nil {
		return Grant{}, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	if standard.Expiry == nil {
		return Grant{}, fmt.Errorf("%w: missing expiration", ErrInvalid)
	}

	expected := jwt.Expected{
		Issuer:      Issuer,
		AnyAudience: jwt.Audience{s.audience},
		Time:        time.Now(),
	}
	// Issuer and verifier are the same process, so there is no clock skew to
	// forgive and leeway would only widen the window of an expired token.
	if err := standard.ValidateWithLeeway(expected, 0); err != nil {
		return Grant{}, fmt.Errorf("%w: %v", ErrInvalid, err)
	}

	return Grant{
		Subject:   standard.Subject,
		UID:       private.UID,
		ID:        standard.ID,
		Scope:     private.Scope,
		ExpiresAt: standard.Expiry.Time(),
	}, nil
}

func randomID() (string, error) {
	buf := make([]byte, idSize)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate token id: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
