//go:build !linux

package brokerapi

import (
	"errors"
	"net"
)

var (
	ErrLocalRuntimeDirPermissions = errors.New("local ipc runtime directory permission checks are unsupported on this platform")
	ErrLocalSocketPermissions     = errors.New("local ipc socket permission checks are unsupported on this platform")
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
	return AdmissionPolicy{RequireSameUID: true}
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

func (l *LocalIPCListener) Close() error { return nil }

func DefaultLocalIPCConfig() (LocalIPCConfig, error) {
	return LocalIPCConfig{}, errors.New("local ipc listener is linux-only for MVP")
}

func ListenLocalIPC(_ LocalIPCConfig) (*LocalIPCListener, error) {
	return nil, errors.New("local ipc listener is linux-only for MVP")
}

func AuthenticateLocalPeer(_ net.Conn, _ AdmissionPolicy) (PeerCredentials, error) {
	return PeerCredentials{}, ErrPeerCredentialsUnavailable
}
