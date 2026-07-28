// Package issuer serves token requests over a unix socket.
//
// Authorization is delegated to filesystem permissions: connecting to a unix
// socket requires write permission on it, so the socket mode and group decide
// who may obtain a token. The kernel enforces that, and no application logic
// can be tricked into bypassing it. SO_PEERCRED then supplies the requester
// identity, which the client cannot forge.
package issuer

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/KaribuLab/linux-mcp/internal/token"
)

const (
	// socketMode keeps the socket owner-only until a group is applied.
	socketMode = 0o600
	// groupSocketMode lets members of the administrative group connect.
	groupSocketMode = 0o660
	// exchangeTimeout bounds a single request/response so a stalled client
	// cannot pin a goroutine.
	exchangeTimeout = 5 * time.Second
	// maxRequestBytes bounds the request body; requests are a few dozen bytes.
	maxRequestBytes = 4 << 10
)

// Request is the client side of the issuance exchange. It deliberately carries
// no identity: the subject comes from SO_PEERCRED, never from the client.
type Request struct {
	TTL string `json:"ttl"`
}

// Response is the server side of the issuance exchange.
type Response struct {
	Token     string    `json:"token,omitempty"`
	Subject   string    `json:"subject,omitempty"`
	ExpiresAt time.Time `json:"expires_at,omitzero"`
	Capped    bool      `json:"capped,omitempty"`
	Error     string    `json:"error,omitempty"`
}

// Server mints tokens for peers allowed to reach the socket.
type Server struct {
	Signer *token.Signer
	MaxTTL time.Duration
	Logger *slog.Logger
}

// Listen creates the issuance socket. When group is empty no ownership change
// is attempted and the socket stays owner-only, which is what local
// development needs: the developer runs the server as themselves.
func Listen(path, group string) (net.Listener, error) {
	if strings.HasPrefix(path, "@") {
		return nil, fmt.Errorf("abstract socket %q is not allowed: abstract sockets carry no permission checks", path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create socket directory: %w", err)
	}
	if err := removeStaleSocket(path); err != nil {
		return nil, err
	}

	// Set the mode through umask instead of a post-listen chmod, which would
	// leave the socket briefly more permissive than intended.
	previous := syscall.Umask(0o777 &^ socketMode)
	listener, err := net.Listen("unix", path)
	syscall.Umask(previous)
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", path, err)
	}

	if group != "" {
		if err := applyGroup(path, group); err != nil {
			listener.Close()
			return nil, err
		}
	}
	return listener, nil
}

// removeStaleSocket clears a leftover socket from an unclean shutdown. It
// refuses to touch anything that is not a socket.
func removeStaleSocket(path string) error {
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("stat socket path: %w", err)
	}
	if info.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("%s exists and is not a socket", path)
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("remove stale socket: %w", err)
	}
	return nil
}

func applyGroup(path, group string) error {
	grp, err := user.LookupGroup(group)
	if err != nil {
		return fmt.Errorf("lookup group %q: %w", group, err)
	}
	gid, err := strconv.Atoi(grp.Gid)
	if err != nil {
		return fmt.Errorf("parse gid of group %q: %w", group, err)
	}
	if err := os.Chown(path, -1, gid); err != nil {
		return fmt.Errorf("set socket group to %q: %w", group, err)
	}
	if err := os.Chmod(path, groupSocketMode); err != nil {
		return fmt.Errorf("set socket mode: %w", err)
	}
	return nil
}

// Serve accepts issuance requests until the listener is closed.
func (s *Server) Serve(listener net.Listener) error {
	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			return fmt.Errorf("accept issuance connection: %w", err)
		}
		go s.handle(conn)
	}
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(exchangeTimeout))

	unixConn, ok := conn.(*net.UnixConn)
	if !ok {
		s.reply(conn, Response{Error: "issuance requires a unix socket connection"})
		return
	}
	cred, err := peerCredentials(unixConn)
	if err != nil {
		s.logger().Error("cannot read peer credentials", "error", err)
		s.reply(conn, Response{Error: "cannot determine requester identity"})
		return
	}

	var req Request
	if err := json.NewDecoder(io.LimitReader(conn, maxRequestBytes)).Decode(&req); err != nil {
		s.reply(conn, Response{Error: "malformed request"})
		return
	}

	ttl, capped, err := s.effectiveTTL(req.TTL)
	if err != nil {
		s.reply(conn, Response{Error: err.Error()})
		return
	}

	subject := usernameForUID(cred.Uid)
	raw, grant, err := s.Signer.Issue(subject, cred.Uid, ttl)
	if err != nil {
		s.logger().Error("cannot issue token", "error", err, "uid", cred.Uid)
		s.reply(conn, Response{Error: "cannot issue token"})
		return
	}

	// Audit record. The jti correlates this line with every later usage line.
	s.logger().Info("token issued",
		"jti", grant.ID,
		"uid", cred.Uid,
		"sub", grant.Subject,
		"pid", cred.Pid,
		"exp", grant.ExpiresAt,
	)
	s.reply(conn, Response{
		Token:     raw,
		Subject:   grant.Subject,
		ExpiresAt: grant.ExpiresAt,
		Capped:    capped,
	})
}

// effectiveTTL honours the requested lifetime up to the configured maximum.
// Without a cap the expiration would stop being a control.
func (s *Server) effectiveTTL(requested string) (time.Duration, bool, error) {
	maxTTL := s.MaxTTL
	if maxTTL <= 0 {
		return 0, false, errors.New("server has no maximum token lifetime configured")
	}
	if requested == "" {
		return maxTTL, false, nil
	}
	ttl, err := time.ParseDuration(requested)
	if err != nil {
		return 0, false, fmt.Errorf("invalid ttl %q", requested)
	}
	if ttl <= 0 {
		return 0, false, fmt.Errorf("ttl must be positive, got %q", requested)
	}
	if ttl > maxTTL {
		return maxTTL, true, nil
	}
	return ttl, false, nil
}

func (s *Server) reply(conn net.Conn, resp Response) {
	if err := json.NewEncoder(conn).Encode(resp); err != nil {
		s.logger().Error("cannot write issuance response", "error", err)
	}
}

func (s *Server) logger() *slog.Logger {
	if s.Logger != nil {
		return s.Logger
	}
	return slog.Default()
}

// peerCredentials asks the kernel who is on the other end. These values are
// recorded by the kernel at connect time and cannot be set by the client.
func peerCredentials(conn *net.UnixConn) (*syscall.Ucred, error) {
	raw, err := conn.SyscallConn()
	if err != nil {
		return nil, fmt.Errorf("access connection: %w", err)
	}
	var cred *syscall.Ucred
	var credErr error
	if err := raw.Control(func(fd uintptr) {
		cred, credErr = syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	}); err != nil {
		return nil, fmt.Errorf("read SO_PEERCRED: %w", err)
	}
	if credErr != nil {
		return nil, fmt.Errorf("read SO_PEERCRED: %w", credErr)
	}
	return cred, nil
}

// usernameForUID resolves a readable subject, falling back to the numeric uid
// when the user is not in the local database. The uid travels in the token
// either way, since usernames can be recycled.
func usernameForUID(uid uint32) string {
	id := strconv.FormatUint(uint64(uid), 10)
	if u, err := user.LookupId(id); err == nil {
		return u.Username
	}
	return id
}
