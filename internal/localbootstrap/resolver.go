package localbootstrap

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

const (
	defaultSocketName = "broker.sock"
	repoIDPrefix      = "repo-"
)

type ResolveInput struct {
	RepositoryRoot string
}

type RepoScope struct {
	RepositoryRoot  string
	ProductInstance string
	StateRoot       string
	AuditLedgerRoot string
	LocalRuntimeDir string
	LocalSocketName string
}

func ResolveRepoScope(input ResolveInput) (RepoScope, error) {
	repoRoot, err := ResolveAuthoritativeRepoRoot(input.RepositoryRoot)
	if err != nil {
		return RepoScope{}, err
	}
	instanceID := deriveProductInstanceID(repoRoot)
	cacheBase := userCacheBaseDir()
	runtimeBase, err := userRuntimeBaseDir()
	if err != nil {
		return RepoScope{}, err
	}
	repoBase := filepath.Join(cacheBase, "runecode", "repos", instanceID)
	return RepoScope{
		RepositoryRoot:  repoRoot,
		ProductInstance: instanceID,
		StateRoot:       filepath.Join(repoBase, "artifact-store"),
		AuditLedgerRoot: filepath.Join(repoBase, "audit-ledger"),
		LocalRuntimeDir: filepath.Join(runtimeBase, "runecode", "repos", instanceID, "brokerapi"),
		LocalSocketName: defaultSocketName,
	}, nil
}

func ResolveAuthoritativeRepoRoot(explicitRoot string) (string, error) {
	root := strings.TrimSpace(explicitRoot)
	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("determine repository root: %w", err)
		}
		root = cwd
	}
	if !filepath.IsAbs(root) {
		return "", fmt.Errorf("repository root must be an absolute directory")
	}
	clean := filepath.Clean(root)
	info, err := os.Stat(clean)
	if err != nil {
		return "", fmt.Errorf("repository root must be an absolute directory")
	}
	if !info.IsDir() {
		return "", fmt.Errorf("repository root must be an absolute directory")
	}
	if vcsRoot, ok := discoverVCSRoot(clean); ok {
		return vcsRoot, nil
	}
	return clean, nil
}

func discoverVCSRoot(start string) (string, bool) {
	current := filepath.Clean(start)
	for {
		gitAnchor := filepath.Join(current, ".git")
		if info, err := os.Lstat(gitAnchor); err == nil {
			if info.Mode()&os.ModeSymlink == 0 && (info.IsDir() || info.Mode().IsRegular()) {
				return current, true
			}
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", false
		}
		current = parent
	}
}

func deriveProductInstanceID(repoRoot string) string {
	sum := sha256.Sum256([]byte("runecode.local-product.v1:" + filepath.ToSlash(repoRoot)))
	encoded := hex.EncodeToString(sum[:])
	if len(encoded) > 24 {
		encoded = encoded[:24]
	}
	return repoIDPrefix + encoded
}

func userCacheBaseDir() string {
	cacheDir, err := os.UserCacheDir()
	if err == nil && strings.TrimSpace(cacheDir) != "" {
		return cacheDir
	}
	configDir, configErr := os.UserConfigDir()
	if configErr == nil && strings.TrimSpace(configDir) != "" {
		return configDir
	}
	return os.TempDir()
}

func userRuntimeBaseDir() (string, error) {
	runtimeBase := strings.TrimSpace(os.Getenv("XDG_RUNTIME_DIR"))
	if strings.TrimSpace(runtimeBase) == "" {
		if cacheDir, err := os.UserCacheDir(); err == nil && strings.TrimSpace(cacheDir) != "" {
			runtimeBase = filepath.Join(cacheDir, "runecode-runtime")
		} else if configDir, err := os.UserConfigDir(); err == nil && strings.TrimSpace(configDir) != "" {
			runtimeBase = filepath.Join(configDir, "runecode-runtime")
		} else {
			runtimeBase = filepath.Join(os.TempDir(), runtimeUserNamespace())
		}
	}
	if strings.TrimSpace(runtimeBase) == "" {
		return "", fmt.Errorf("resolve user runtime directory: empty path")
	}
	return runtimeBase, nil
}

func runtimeUserNamespace() string {
	if current, err := user.Current(); err == nil {
		if uid := sanitizePathToken(strings.TrimSpace(current.Uid)); uid != "" {
			return "runecode-" + uid
		}
		if name := sanitizePathToken(strings.TrimSpace(current.Username)); name != "" {
			return "runecode-" + name
		}
	}
	if value := sanitizePathToken(strings.TrimSpace(os.Getenv("USER"))); value != "" {
		return "runecode-" + value
	}
	if value := sanitizePathToken(strings.TrimSpace(os.Getenv("USERNAME"))); value != "" {
		return "runecode-" + value
	}
	return "runecode-user"
}

func sanitizePathToken(value string) string {
	if value == "" {
		return ""
	}
	b := strings.Builder{}
	b.Grow(len(value))
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_':
			b.WriteRune(r)
		}
	}
	return strings.Trim(b.String(), "-_")
}
