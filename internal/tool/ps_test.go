package tool

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/KaribuLab/linux-mcp/internal/policy"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func writeFakeProc(t *testing.T, root string, pid int, comm, state string, ppid int, cmdline string, rssKiB uint64, uid int) {
	t.Helper()
	dir := filepath.Join(root, strconv.Itoa(pid))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// /proc/pid/stat: pid (comm) state ppid ...
	stat := strconv.Itoa(pid) + " (" + comm + ") " + state + " " + strconv.Itoa(ppid) + " 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0\n"
	if err := os.WriteFile(filepath.Join(dir, "stat"), []byte(stat), 0o644); err != nil {
		t.Fatal(err)
	}
	var cmd []byte
	if cmdline != "" {
		parts := strings.Fields(cmdline)
		cmd = []byte(strings.Join(parts, "\x00") + "\x00")
	}
	if err := os.WriteFile(filepath.Join(dir, "cmdline"), cmd, 0o644); err != nil {
		t.Fatal(err)
	}
	status := "Name:\t" + comm + "\nUid:\t" + strconv.Itoa(uid) + "\t" + strconv.Itoa(uid) + "\t" + strconv.Itoa(uid) + "\t" + strconv.Itoa(uid) + "\nVmRSS:\t" + strconv.FormatUint(rssKiB, 10) + " kB\n"
	if err := os.WriteFile(filepath.Join(dir, "status"), []byte(status), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/usr/bin/"+comm, filepath.Join(dir, "exe")); err != nil {
		t.Fatal(err)
	}
}

func withFakeProc(t *testing.T, setup func(root string)) {
	t.Helper()
	root := t.TempDir()
	setup(root)
	old := procRoot
	procRoot = root
	t.Cleanup(func() { procRoot = old })
}

func TestPsOmitsKernelByDefault(t *testing.T) {
	withFakeProc(t, func(root string) {
		writeFakeProc(t, root, 1, "init", "S", 0, "/sbin/init", 100, 0)
		writeFakeProc(t, root, 2, "kthreadd", "S", 0, "", 0, 0)
	})

	res, _, err := Ps(context.Background(), nil, PsArgs{})
	if err != nil {
		t.Fatal(err)
	}
	text := res.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "|1|init|") {
		t.Fatalf("expected init: %s", text)
	}
	if strings.Contains(text, "kthreadd") {
		t.Fatalf("kernel thread should be omitted: %s", text)
	}
	if !strings.HasPrefix(text, "[ps ") {
		t.Fatalf("meta: %s", text)
	}
}

func TestPsIncludeKernel(t *testing.T) {
	withFakeProc(t, func(root string) {
		writeFakeProc(t, root, 2, "kthreadd", "S", 0, "", 0, 0)
	})

	res, _, err := Ps(context.Background(), nil, PsArgs{IncludeKernel: true})
	if err != nil {
		t.Fatal(err)
	}
	text := res.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "kthreadd") {
		t.Fatalf("expected kernel thread: %s", text)
	}
}

func TestPsHideColumns(t *testing.T) {
	withFakeProc(t, func(root string) {
		writeFakeProc(t, root, 10, "nginx", "S", 1, "nginx: master", 2048, 1000)
	})
	f := false
	res, _, err := Ps(context.Background(), nil, PsArgs{ShowCmdline: &f, ShowExe: &f, ShowMem: &f})
	if err != nil {
		t.Fatal(err)
	}
	text := res.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "columns=Pid,Comm,Ppid,User,Stat,Cpu") {
		t.Fatalf("columns meta: %s", text)
	}
	if strings.Contains(text, "Cmdline") && strings.Contains(strings.Split(text, "\n")[1], "Cmdline") {
		// header row should not include Cmdline
		header := strings.Split(text, "\n")[1]
		if strings.Contains(header, "Cmdline") {
			t.Fatalf("Cmdline still in header: %s", header)
		}
	}
	if !strings.Contains(text, "|10|nginx|") {
		t.Fatalf("identity missing: %s", text)
	}
}

func TestPsTruncates(t *testing.T) {
	withFakeProc(t, func(root string) {
		for i := 1; i <= policy.MaxListEntries+5; i++ {
			writeFakeProc(t, root, i, "p"+strconv.Itoa(i), "S", 1, "/bin/p"+strconv.Itoa(i), 1, 0)
		}
	})
	res, _, err := Ps(context.Background(), nil, PsArgs{})
	if err != nil {
		t.Fatal(err)
	}
	text := res.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "truncated=true") {
		t.Fatalf("expected truncated: %s", text)
	}
	lines := strings.Split(strings.TrimSpace(text), "\n")
	// meta + header + sep + 1000 rows
	if len(lines) != 3+policy.MaxListEntries {
		t.Fatalf("rows=%d want %d; head=%q", len(lines)-3, policy.MaxListEntries, lines[0])
	}
}

func TestParseProcStatWithSpacesInComm(t *testing.T) {
	pid, comm, state, ppid, ok := parseProcStat([]byte("42 (my app) S 1 0 0\n"))
	if !ok || pid != 42 || comm != "my app" || state != "S" || ppid != 1 {
		t.Fatalf("got pid=%d comm=%q state=%q ppid=%d ok=%v", pid, comm, state, ppid, ok)
	}
}
