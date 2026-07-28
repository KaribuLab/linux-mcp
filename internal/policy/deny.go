package policy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Class identifies why a read/list was blocked.
type Class string

const (
	ClassPathDenied Class = "path_denied"
	ClassTypeDenied Class = "type_denied"
	ClassPrivateKey Class = "private_key"
	ClassBinary     Class = "binary"
)

// Denied is a policy rejection (path, type, or content).
type Denied struct {
	Class Class
	Path  string
}

func (d *Denied) Error() string {
	return fmt.Sprintf("blocked class=%s path=%s", d.Class, d.Path)
}

var exactDenied = map[string]struct{}{
	"/etc/shadow":  {},
	"/etc/gshadow": {},
}

var deniedBasenames = map[string]struct{}{
	"mem":   {},
	"kcore": {},
}

// ResolvePath returns a cleaned absolute path. When the path exists, EvalSymlinks
// is preferred so denylist checks follow the final target.
func ResolvePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty path")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved, nil
	}
	return abs, nil
}

// CheckPath evaluates the denylist against the resolved path (and the cleaned
// absolute form when they differ). Does not depend on systemd.
func CheckPath(path string) (abs string, err error) {
	abs, err = ResolvePath(path)
	if err != nil {
		return "", err
	}
	candidates := []string{abs}
	if cleaned, e := filepath.Abs(path); e == nil {
		cleaned = filepath.Clean(cleaned)
		if cleaned != abs {
			candidates = append(candidates, cleaned)
		}
	}
	for _, p := range candidates {
		if denied, why := matchDenied(p); denied {
			return abs, &Denied{Class: why, Path: p}
		}
	}
	return abs, nil
}

func matchDenied(path string) (bool, Class) {
	if _, ok := exactDenied[path]; ok {
		return true, ClassPathDenied
	}
	base := filepath.Base(path)
	if _, ok := deniedBasenames[base]; ok {
		return true, ClassPathDenied
	}
	// /proc/<pid>/mem and similar
	if strings.HasPrefix(path, "/proc/") && base == "mem" {
		return true, ClassPathDenied
	}
	// Extra path-layer for private keys (content sniff is primary).
	if isKeyPath(path) {
		return true, ClassPathDenied
	}
	return false, ""
}

func isKeyPath(path string) bool {
	base := filepath.Base(path)
	dir := filepath.Base(filepath.Dir(path))
	if dir == ".ssh" && strings.HasPrefix(base, "id_") && !strings.HasSuffix(base, ".pub") {
		return true
	}
	lower := strings.ToLower(base)
	if strings.HasSuffix(lower, ".pem") {
		return true
	}
	return false
}

// CheckReadableType rejects device nodes, sockets, and other non-regular special
// files for content reading. Directories are also rejected for cat-style reads.
func CheckReadableType(path string, info os.FileInfo) error {
	mode := info.Mode()
	switch {
	case mode.IsRegular():
		return nil
	case mode&os.ModeCharDevice != 0, mode&os.ModeDevice != 0:
		return &Denied{Class: ClassTypeDenied, Path: path}
	case mode&os.ModeNamedPipe != 0, mode&os.ModeSocket != 0:
		return &Denied{Class: ClassTypeDenied, Path: path}
	case mode.IsDir():
		return &Denied{Class: ClassTypeDenied, Path: path}
	default:
		// Symlinks should already be followed by Open; treat unknown specials as deny.
		if !mode.IsRegular() && mode&os.ModeSymlink == 0 {
			return &Denied{Class: ClassTypeDenied, Path: path}
		}
		return nil
	}
}
