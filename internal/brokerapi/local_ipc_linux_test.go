//go:build linux

package brokerapi

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestListenLocalIPCPermissionPosture(t *testing.T) {
	base := shortBaseDir(t)
	t.Setenv("XDG_RUNTIME_DIR", base)
	runtimeDir := filepath.Join(base, "runtime")
	oldUmask := syscall.Umask(0)
	defer syscall.Umask(oldUmask)

	l, err := ListenLocalIPC(LocalIPCConfig{RuntimeDir: runtimeDir, SocketName: "broker.sock"})
	if err != nil {
		t.Fatalf("ListenLocalIPC returned error: %v", err)
	}
	defer l.Close()

	dirInfo, err := os.Stat(runtimeDir)
	if err != nil {
		t.Fatalf("Stat(runtimeDir) returned error: %v", err)
	}
	if got := dirInfo.Mode().Perm(); got != localRuntimeDirPerm {
		t.Fatalf("runtime dir perms = %o, want %o", got, localRuntimeDirPerm)
	}

	sockInfo, err := os.Stat(l.SocketPath)
	if err != nil {
		t.Fatalf("Stat(socket) returned error: %v", err)
	}
	if got := sockInfo.Mode().Perm(); got != localSocketPerm {
		t.Fatalf("socket perms = %o, want %o", got, localSocketPerm)
	}
}

func TestListenLocalIPCRejectsRuntimeDirWithLoosePermissions(t *testing.T) {
	base := shortBaseDir(t)
	t.Setenv("XDG_RUNTIME_DIR", base)
	runtimeDir := filepath.Join(base, "runtime")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.Chmod(runtimeDir, 0o755); err != nil {
		t.Fatalf("Chmod returned error: %v", err)
	}
	_, err := ListenLocalIPC(LocalIPCConfig{RuntimeDir: runtimeDir, SocketName: "broker.sock"})
	if err == nil {
		t.Fatal("ListenLocalIPC error = nil, want permission failure")
	}
	if !errors.Is(err, ErrLocalRuntimeDirPermissions) {
		t.Fatalf("error = %v, want ErrLocalRuntimeDirPermissions", err)
	}
}

func TestEnsureLocalSocketPermissionsRejectsLoosePermissions(t *testing.T) {
	base := shortBaseDir(t)
	t.Setenv("XDG_RUNTIME_DIR", base)
	runtimeDir := filepath.Join(base, "runtime")
	listener, err := ListenLocalIPC(LocalIPCConfig{RuntimeDir: runtimeDir, SocketName: "broker.sock"})
	if err != nil {
		t.Fatalf("ListenLocalIPC returned error: %v", err)
	}
	defer listener.Close()
	if err := os.Chmod(listener.SocketPath, 0o666); err != nil {
		t.Fatalf("Chmod returned error: %v", err)
	}
	err = ensureLocalSocketPermissions(listener.SocketPath)
	if err == nil {
		t.Fatal("ensureLocalSocketPermissions error = nil, want socket permission failure")
	}
	if !errors.Is(err, ErrLocalSocketPermissions) {
		t.Fatalf("error = %v, want ErrLocalSocketPermissions", err)
	}
}

func TestAuthenticateLocalPeerRejectsWrongUID(t *testing.T) {
	base := shortBaseDir(t)
	t.Setenv("XDG_RUNTIME_DIR", base)
	runtimeDir := filepath.Join(base, "runtime")
	listener, err := ListenLocalIPC(LocalIPCConfig{RuntimeDir: runtimeDir, SocketName: "broker.sock"})
	if err != nil {
		t.Fatalf("ListenLocalIPC returned error: %v", err)
	}
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() {
		conn, acceptErr := listener.Listener.Accept()
		if acceptErr != nil {
			errCh <- acceptErr
			return
		}
		defer conn.Close()
		_, authErr := AuthenticateLocalPeer(conn, AdmissionPolicy{RequireSameUID: true, AllowedUID: uint32(os.Getuid() + 1)})
		errCh <- authErr
	}()

	client, err := net.Dial("unix", listener.SocketPath)
	if err != nil {
		t.Fatalf("Dial returned error: %v", err)
	}
	_ = client.Close()

	if err := <-errCh; !errors.Is(err, ErrPeerUIDMismatch) {
		t.Fatalf("auth error = %v, want ErrPeerUIDMismatch", err)
	}
}

func TestAuthenticateLocalPeerHonorsExplicitZeroAllowedUID(t *testing.T) {
	base := shortBaseDir(t)
	t.Setenv("XDG_RUNTIME_DIR", base)
	runtimeDir := filepath.Join(base, "runtime")
	listener, err := ListenLocalIPC(LocalIPCConfig{RuntimeDir: runtimeDir, SocketName: "broker.sock"})
	if err != nil {
		t.Fatalf("ListenLocalIPC returned error: %v", err)
	}
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() {
		conn, acceptErr := listener.Listener.Accept()
		if acceptErr != nil {
			errCh <- acceptErr
			return
		}
		defer conn.Close()
		_, authErr := AuthenticateLocalPeer(conn, AdmissionPolicy{RequireSameUID: true, AllowedUID: 0})
		errCh <- authErr
	}()

	client, err := net.Dial("unix", listener.SocketPath)
	if err != nil {
		t.Fatalf("Dial returned error: %v", err)
	}
	_ = client.Close()

	authErr := <-errCh
	if os.Getuid() == 0 {
		if authErr != nil {
			t.Fatalf("auth error = %v, want nil for root uid with AllowedUID=0", authErr)
		}
		return
	}
	if !errors.Is(authErr, ErrPeerUIDMismatch) {
		t.Fatalf("auth error = %v, want ErrPeerUIDMismatch for explicit AllowedUID=0", authErr)
	}
}

func TestAuthenticateLocalPeerFailsClosedWhenCredentialsUnavailable(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", shortBaseDir(t))
	left, right := net.Pipe()
	defer left.Close()
	defer right.Close()
	_, err := AuthenticateLocalPeer(left, DefaultAdmissionPolicy())
	if err == nil {
		t.Fatal("AuthenticateLocalPeer error = nil, want peer credential failure")
	}
	if !errors.Is(err, ErrPeerCredentialsUnavailable) {
		t.Fatalf("error = %v, want ErrPeerCredentialsUnavailable", err)
	}
}

func TestDefaultLocalIPCConfigIsRepoScoped(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, ".git"), 0o755); err != nil {
		t.Fatalf("MkdirAll .git returned error: %v", err)
	}
	nested := filepath.Join(repoRoot, "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("MkdirAll nested returned error: %v", err)
	}
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}
	if err := os.Chdir(nested); err != nil {
		t.Fatalf("Chdir returned error: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })
	runtimeBase := shortBaseDir(t)
	t.Setenv("XDG_RUNTIME_DIR", runtimeBase)

	cfg, err := DefaultLocalIPCConfig()
	if err != nil {
		t.Fatalf("DefaultLocalIPCConfig returned error: %v", err)
	}
	if cfg.RepositoryRoot != repoRoot {
		t.Fatalf("repository root = %q, want %q", cfg.RepositoryRoot, repoRoot)
	}
	if !strings.HasPrefix(cfg.RuntimeDir, runtimeBase) {
		t.Fatalf("runtime dir = %q, want prefix %q", cfg.RuntimeDir, runtimeBase)
	}
	if !strings.Contains(filepath.ToSlash(cfg.RuntimeDir), "/runecode/repos/") {
		t.Fatalf("runtime dir = %q, want repo-scoped path segment", cfg.RuntimeDir)
	}
}

func shortBaseDir(t *testing.T) string {
	t.Helper()
	base, err := os.MkdirTemp("", "rcipc-")
	if err != nil {
		t.Fatalf("MkdirTemp returned error: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(base) })
	return base
}
