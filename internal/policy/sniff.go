package policy

import (
	"bufio"
	"bytes"
	"strings"
	"unicode"
)

// ClassifyPrefix inspects a bounded prefix: NUL → binary; first useful line → private key.
// Returns a Denied class or empty class if allowed.
func ClassifyPrefix(prefix []byte) Class {
	if bytes.IndexByte(prefix, 0) >= 0 {
		return ClassBinary
	}
	line := firstUsefulLine(prefix)
	if line == "" {
		return ""
	}
	if isPrivateKeyHeader(line) {
		return ClassPrivateKey
	}
	return ""
}

func firstUsefulLine(prefix []byte) string {
	data := prefix
	// UTF-8 BOM
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	sc := bufio.NewScanner(bytes.NewReader(data))
	// Allow long first line within prefix.
	sc.Buffer(make([]byte, 0, 1024), len(data)+1)
	for sc.Scan() {
		line := strings.TrimRightFunc(sc.Text(), unicode.IsSpace)
		if strings.TrimSpace(line) == "" {
			continue
		}
		return line
	}
	return ""
}

func isPrivateKeyHeader(line string) bool {
	s := strings.TrimSpace(line)
	upper := strings.ToUpper(s)
	switch {
	case strings.Contains(upper, "BEGIN OPENSSH PRIVATE KEY"):
		return true
	case strings.Contains(upper, "BEGIN ENCRYPTED PRIVATE KEY"):
		return true
	case strings.Contains(upper, "BEGIN PGP PRIVATE KEY BLOCK"):
		return true
	case strings.HasPrefix(s, "PuTTY-User-Key-File"):
		return true
	case strings.Contains(upper, "PRIVATE KEY") && strings.Contains(upper, "BEGIN") && !strings.Contains(upper, "PUBLIC"):
		return true
	default:
		return false
	}
}
