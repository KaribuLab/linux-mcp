package policy_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/KaribuLab/linux-mcp/internal/policy"
)

func TestCheckPathDeniesShadow(t *testing.T) {
	_, err := policy.CheckPath("/etc/shadow")
	d, ok := err.(*policy.Denied)
	if !ok || d.Class != policy.ClassPathDenied {
		t.Fatalf("want path_denied, got %v", err)
	}
}

func TestCheckPathAllowsProcStat(t *testing.T) {
	if _, err := policy.CheckPath("/proc/self/status"); err != nil {
		t.Fatalf("unexpected deny: %v", err)
	}
}

func TestCheckPathDeniesSSHKey(t *testing.T) {
	p := filepath.Join(t.TempDir(), ".ssh", "id_ed25519")
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := policy.CheckPath(p)
	d, ok := err.(*policy.Denied)
	if !ok || d.Class != policy.ClassPathDenied {
		t.Fatalf("want path_denied for ssh key, got %v", err)
	}
}

func TestClassifyPrefixPrivateKey(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want policy.Class
	}{
		{"rsa", "-----BEGIN RSA PRIVATE KEY-----\nMII...\n", policy.ClassPrivateKey},
		{"openssh", "\n\n-----BEGIN OPENSSH PRIVATE KEY-----\n", policy.ClassPrivateKey},
		{"bom", "\ufeff-----BEGIN PRIVATE KEY-----\n", policy.ClassPrivateKey},
		{"putty", "PuTTY-User-Key-File-2: ssh-rsa\n", policy.ClassPrivateKey},
		{"public", "-----BEGIN PUBLIC KEY-----\nMII...\n", ""},
		{"midfile", "See example:\n-----BEGIN RSA PRIVATE KEY-----\n", ""},
		{"binary", "hello\x00world", policy.ClassBinary},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := policy.ClassifyPrefix([]byte(tc.in))
			if got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestReadPageCapsAndSeek(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "big.txt")
	var b strings.Builder
	for i := 0; i < 250; i++ {
		b.WriteString("line-")
		b.WriteByte(byte('A' + (i % 26)))
		b.WriteByte('\n')
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	page1, err := policy.ReadPage(path, 0, policy.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	if page1.Lines != policy.MaxLines || !page1.Truncated || page1.NextByte <= 0 {
		t.Fatalf("page1 lines=%d trunc=%v next=%d", page1.Lines, page1.Truncated, page1.NextByte)
	}

	page2, err := policy.ReadPage(path, page1.NextByte, policy.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	if page2.Lines == 0 {
		t.Fatal("page2 empty")
	}
	if strings.Contains(page2.Body.String(), page1.Body.String()[:20]) {
		// Unlikely exact prefix overlap if seek worked; soft check: first line of page2 differs.
	}
	first1 := strings.SplitN(page1.Body.String(), "\n", 2)[0]
	first2 := strings.SplitN(page2.Body.String(), "\n", 2)[0]
	if first1 == first2 {
		t.Fatalf("seek failed: both pages start with %q", first1)
	}
}

func TestReadPageBlocksPEM(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "key.txt")
	content := "-----BEGIN OPENSSH PRIVATE KEY-----\nAAAA\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := policy.ReadPage(path, 0, policy.Limits{})
	d, ok := err.(*policy.Denied)
	if !ok || d.Class != policy.ClassPrivateKey {
		t.Fatalf("want private_key, got %v", err)
	}
}

func TestReadPageByteCap(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "wide.txt")
	line := strings.Repeat("x", 1000) + "\n"
	if err := os.WriteFile(path, []byte(strings.Repeat(line, 100)), 0o644); err != nil {
		t.Fatal(err)
	}
	page, err := policy.ReadPage(path, 0, policy.Limits{MaxLines: 1000, MaxBytes: 2048})
	if err != nil {
		t.Fatal(err)
	}
	if page.Body.Len() > 2048 {
		t.Fatalf("body %d > 2048", page.Body.Len())
	}
	if !page.Truncated {
		t.Fatal("expected truncated by bytes")
	}
}
