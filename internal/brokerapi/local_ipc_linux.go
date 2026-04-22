//go:build linux

package brokerapi

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/runecode-ai/runecode/internal/localbootstrap"
	"golang.org/x/sys/unix"
)

const (
	localRuntimeDirPerm = os.FileMode(0o700)
	localSocketPerm     = os.FileMode(0o600)
	defaultSocketName   = "broker.sock"
)

var (
	ErrLocalRuntimeDirPermissions = errors.New("broker local runtime directory permissions must be 0700")
	ErrLocalSocketPermissions     = errors.New("broker local socket permissions must be 0600")
	ErrPeerCredentialsUnavailable = errors.New("peer credentials unavailable")
	ErrPeerUIDMismatch            = errors.New("peer uid does not match broker uid")
)

type PeerCredentials struct {
	PID int
	UID uint32
	GID uint32
}

type AdmissionPolicy struct {
	RequireSameUID bool
	AllowedUID     uint32
}

func DefaultAdmissionPolicy() AdmissionPolicy {
	return AdmissionPolicy{
		RequireSameUID: true,
		AllowedUID:     uint32(os.Getuid()),
	}
}

type LocalIPCConfig struct {
	RuntimeDir     string
	SocketName     string
	RepositoryRoot string
}

type LocalIPCListener struct {
	Listener   net.Listener
	SocketPath string
	RuntimeDir string
}

func (l *LocalIPCListener) Close() error {
	if l == nil {
		return nil
	}
	if l.Listener != nil {
		if err := l.Listener.Close(); err != nil {
			return err
		}
	}
	if l.SocketPath != "" {
		if err := os.Remove(l.SocketPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func DefaultLocalIPCConfig() (LocalIPCConfig, error) {
	scope, err := localbootstrap.ResolveRepoScope(localbootstrap.ResolveInput{})
	if err != nil {
		return LocalIPCConfig{}, err
	}
	return LocalIPCConfig{
		RuntimeDir:     scope.LocalRuntimeDir,
		SocketName:     scope.LocalSocketName,
		RepositoryRoot: scope.RepositoryRoot,
	}, nil
}

func (c LocalIPCConfig) withDefaults() LocalIPCConfig {
	resolved := c
	if strings.TrimSpace(resolved.SocketName) == "" {
		resolved.SocketName = defaultSocketName
	}
	return resolved
}

func (c LocalIPCConfig) socketPath() (string, error) {
	if strings.TrimSpace(c.RuntimeDir) == "" {
		return "", fmt.Errorf("runtime directory is required")
	}
	if strings.TrimSpace(c.SocketName) == "" {
		return "", fmt.Errorf("socket name is required")
	}
	if strings.ContainsRune(c.SocketName, filepath.Separator) {
		return "", fmt.Errorf("socket name must not include path separators")
	}
	return filepath.Join(c.RuntimeDir, c.SocketName), nil
}

func ensureLocalRuntimeDir(path string) error {
	if err := os.MkdirAll(path, localRuntimeDirPerm); err != nil {
		return err
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}
	if mode := info.Mode().Perm(); mode != localRuntimeDirPerm {
		return fmt.Errorf("%w: got %o", ErrLocalRuntimeDirPermissions, mode)
	}
	return nil
}

func ensureLocalSocketPermissions(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("%s is not a unix socket", path)
	}
	if mode := info.Mode().Perm(); mode != localSocketPerm {
		return fmt.Errorf("%w: got %o", ErrLocalSocketPermissions, mode)
	}
	return nil
}

var umaskMu sync.Mutex

func withRestrictedUmask(mask int, fn func() error) error {
	umaskMu.Lock()
	defer umaskMu.Unlock()
	old := syscall.Umask(mask)
	defer syscall.Umask(old)
	return fn()
}

func ListenLocalIPC(cfg LocalIPCConfig) (*LocalIPCListener, error) {
	resolved := cfg.withDefaults()
	socketPath, err := resolved.socketPath()
	if err != nil {
		return nil, err
	}
	listener, err := listenLocalUnixSocket(resolved.RuntimeDir, socketPath)
	if err != nil {
		return nil, err
	}
	if err := postListenLocalIPCChecks(resolved.RuntimeDir, socketPath); err != nil {
		_ = listener.Close()
		return nil, err
	}
	return &LocalIPCListener{Listener: listener, SocketPath: socketPath, RuntimeDir: resolved.RuntimeDir}, nil
}

func listenLocalUnixSocket(runtimeDir, socketPath string) (net.Listener, error) {
	var listener net.Listener
	err := withRestrictedUmask(0o077, func() error {
		if err := ensureLocalRuntimeDir(runtimeDir); err != nil {
			return err
		}
		if err := removeExistingSocketFile(socketPath); err != nil {
			return err
		}
		created, listenErr := net.Listen("unix", socketPath)
		if listenErr != nil {
			return listenErr
		}
		listener = created
		return nil
	})
	if err != nil {
		return nil, err
	}
	return listener, nil
}

func removeExistingSocketFile(socketPath string) error {
	info, statErr := os.Lstat(socketPath)
	if errors.Is(statErr, os.ErrNotExist) {
		return nil
	}
	if statErr != nil {
		return statErr
	}
	if info.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("existing socket path %q is not a socket", socketPath)
	}
	if info.Mode().Perm() != localSocketPerm {
		return fmt.Errorf("%w: got %o", ErrLocalSocketPermissions, info.Mode().Perm())
	}
	return os.Remove(socketPath)
}

func postListenLocalIPCChecks(runtimeDir, socketPath string) error {
	if err := os.Chmod(socketPath, localSocketPerm); err != nil {
		return err
	}
	if err := ensureLocalRuntimeDir(runtimeDir); err != nil {
		return err
	}
	if err := ensureLocalSocketPermissions(socketPath); err != nil {
		return err
	}
	return nil
}

func PeerCredentialsFromConn(conn net.Conn) (PeerCredentials, error) {
	unixConn, ok := conn.(*net.UnixConn)
	if !ok {
		return PeerCredentials{}, ErrPeerCredentialsUnavailable
	}
	raw, err := unixConn.SyscallConn()
	if err != nil {
		return PeerCredentials{}, fmt.Errorf("%w: syscall conn: %v", ErrPeerCredentialsUnavailable, err)
	}
	var ucred *unix.Ucred
	controlErr := raw.Control(func(fd uintptr) {
		ucred, err = unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
	})
	if controlErr != nil {
		return PeerCredentials{}, fmt.Errorf("%w: control: %v", ErrPeerCredentialsUnavailable, controlErr)
	}
	if err != nil {
		return PeerCredentials{}, fmt.Errorf("%w: getsockopt ucred: %v", ErrPeerCredentialsUnavailable, err)
	}
	if ucred == nil {
		return PeerCredentials{}, ErrPeerCredentialsUnavailable
	}
	return PeerCredentials{PID: int(ucred.Pid), UID: ucred.Uid, GID: ucred.Gid}, nil
}

func AuthenticateLocalPeer(conn net.Conn, policy AdmissionPolicy) (PeerCredentials, error) {
	creds, err := PeerCredentialsFromConn(conn)
	if err != nil {
		return PeerCredentials{}, err
	}
	if policy.RequireSameUID && creds.UID != policy.AllowedUID {
		return PeerCredentials{}, fmt.Errorf("%w: got uid=%d want uid=%d", ErrPeerUIDMismatch, creds.UID, policy.AllowedUID)
	}
	return creds, nil
}
