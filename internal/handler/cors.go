package handler

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
)

// allowedRequestHeaders must list Authorization, or a browser preflight would
// forbid the very header that carries the token.
const allowedRequestHeaders = "Authorization, Content-Type, Accept, Mcp-Session-Id, Mcp-Protocol-Version, Last-Event-ID"

// Origins is the parsed allowlist of browser origins. The zero value allows
// none, which is the intended default: a browser client is opt-in.
type Origins struct {
	exact map[string]struct{}
	// anyPort holds "scheme://host" entries that match on any port. Only
	// loopback hosts may appear here.
	anyPort map[string]struct{}
}

// ParseOrigins validates operator-supplied origins and fails on anything it
// cannot enforce, so a typo becomes a startup error instead of a silently
// unreachable rule.
//
// Values must match what a browser puts in the Origin header: scheme and host,
// plus a port when the browser sends one. A trailing ":*" matches any port, and
// is only accepted for loopback hosts, where the port is often unpredictable.
func ParseOrigins(values []string) (Origins, error) {
	origins := Origins{
		exact:   map[string]struct{}{},
		anyPort: map[string]struct{}{},
	}
	for _, raw := range values {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		if wildcardPort := strings.HasSuffix(value, ":*"); wildcardPort {
			key, err := parseWildcardOrigin(strings.TrimSuffix(value, ":*"))
			if err != nil {
				return Origins{}, err
			}
			origins.anyPort[key] = struct{}{}
			continue
		}
		if err := validateOrigin(value); err != nil {
			return Origins{}, err
		}
		origins.exact[value] = struct{}{}
	}
	return origins, nil
}

func parseWildcardOrigin(base string) (string, error) {
	if err := validateOrigin(base); err != nil {
		return "", err
	}
	parsed, _ := url.Parse(base)
	host := parsed.Hostname()
	if !isLoopbackHost(host) {
		return "", fmt.Errorf("origin %q: a wildcard port is only allowed for loopback hosts", base+":*")
	}
	return parsed.Scheme + "://" + host, nil
}

func validateOrigin(value string) error {
	parsed, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("origin %q: %w", value, err)
	}
	switch {
	case parsed.Scheme != "http" && parsed.Scheme != "https":
		return fmt.Errorf("origin %q must start with http:// or https://", value)
	case parsed.Host == "":
		return fmt.Errorf("origin %q has no host", value)
	case strings.Contains(parsed.Host, "*"):
		return fmt.Errorf("origin %q: the host cannot be wildcarded, only the port of a loopback host", value)
	case parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.User != nil:
		return fmt.Errorf("origin %q must be just scheme, host and port", value)
	}
	return nil
}

func (o Origins) allows(origin string) bool {
	if _, ok := o.exact[origin]; ok {
		return true
	}
	if len(o.anyPort) == 0 {
		return false
	}
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	_, ok := o.anyPort[parsed.Scheme+"://"+parsed.Hostname()]
	return ok
}

// withCORS answers preflights and enforces the origin allowlist. It wraps
// authentication so that an OPTIONS request, which browsers send without
// credentials, is not answered with a 401 the browser would report as a CORS
// failure.
func withCORS(next http.Handler, origins Origins, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The response varies by origin even when the request has none.
		w.Header().Add("Vary", "Origin")

		if origin := r.Header.Get("Origin"); origin != "" {
			if !origins.allows(origin) {
				// Logged so an operator integrating a new client can add the
				// exact origin instead of widening the policy by guessing.
				logger.Warn("origin rejected", "origin", origin, "remote", r.RemoteAddr)
				http.Error(w, "origin not allowed", http.StatusForbidden)
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", allowedRequestHeaders)
			w.Header().Set("Access-Control-Expose-Headers", "Mcp-Session-Id")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// hostCheck defends against DNS rebinding: a page on an attacker domain that
// resolves to 127.0.0.1 is same-origin from the browser's point of view, so
// CORS never applies and only the Host header gives the server away.
type hostCheck struct {
	host string
	port string
}

func newHostCheck(addr string) (hostCheck, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return hostCheck{}, fmt.Errorf("listen address %q must be host:port: %w", addr, err)
	}
	if port == "" {
		return hostCheck{}, fmt.Errorf("listen address %q has no port", addr)
	}
	return hostCheck{host: host, port: port}, nil
}

// allows accepts the port it serves on, reached either through a loopback name
// or through the exact address it was told to bind. An operator binding a
// non-loopback address must use that same address in the client, since the
// server cannot tell a legitimate alias from a rebound one.
func (h hostCheck) allows(requestHost string) bool {
	name, port, err := net.SplitHostPort(requestHost)
	if err != nil {
		// No port in the Host header; the server always listens on one.
		return false
	}
	if port != h.port {
		return false
	}
	return isLoopbackHost(name) || name == h.host
}

func withHostCheck(next http.Handler, check hostCheck, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !check.allows(r.Host) {
			logger.Warn("host rejected", "host", r.Host, "remote", r.RemoteAddr)
			http.Error(w, "host not allowed", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}
