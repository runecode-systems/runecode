//go:build linux

package launcherdaemon

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/sys/execabs"
)

var helloWorldGoBinaryCandidates = []string{"/usr/bin/go", "/usr/local/go/bin/go"}

func buildHelloInitramfs(ctx context.Context, launchDir string) (string, error) {
	goBin, err := resolveHelloWorldGoBinary()
	if err != nil {
		return "", fmt.Errorf("resolve hello-world go toolchain: %w", err)
	}
	src := filepath.Join(launchDir, "init.go")
	if err := os.WriteFile(src, []byte(helloInitProgram()), 0o600); err != nil {
		return "", err
	}
	binPath := filepath.Join(launchDir, "init")
	if err := buildHelloInitBinary(ctx, goBin, binPath, src); err != nil {
		return "", err
	}
	binary, err := os.ReadFile(binPath)
	if err != nil {
		return "", err
	}
	archive, err := buildCPIONewc(map[string]cpioEntry{"init": {Mode: 0o755, Data: binary}})
	if err != nil {
		return "", err
	}
	initrdPath := filepath.Join(launchDir, "initramfs.cpio")
	if err := os.WriteFile(initrdPath, archive, 0o600); err != nil {
		return "", err
	}
	return initrdPath, nil
}

func helloInitProgram() string {
	return `package main
import (
  "fmt"
  "syscall"
)
func main() {
  fmt.Println("` + helloWorldToken + `")
  _ = syscall.Reboot(syscall.LINUX_REBOOT_CMD_POWER_OFF)
}`
}

func buildHelloInitBinary(ctx context.Context, goBin, binPath, src string) error {
	workDir := filepath.Dir(src)
	if err := os.MkdirAll(filepath.Join(workDir, "gocache"), 0o700); err != nil {
		return err
	}
	build := execabs.CommandContext(ctx, goBin, "build", "-trimpath", "-ldflags", "-buildid=", "-o", binPath, src)
	build.Env = helloWorldGoBuildEnv(workDir)
	if out, err := build.CombinedOutput(); err != nil {
		return fmt.Errorf("go build init failed: %w: %s", err, string(out))
	}
	return nil
}

func resolveHelloWorldGoBinary() (string, error) {
	for _, candidate := range helloWorldGoBinaryCandidates {
		if err := validateHelloWorldGoBinary(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("host go compiler not found in approved paths")
}

func validateHelloWorldGoBinary(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("go compiler path is a directory")
	}
	if info.Mode().Perm()&0o111 == 0 {
		return fmt.Errorf("go compiler path is not executable")
	}
	return nil
}

func helloWorldGoBuildEnv(workDir string) []string {
	return []string{
		"CGO_ENABLED=0",
		"GOOS=linux",
		"GOARCH=amd64",
		"GOTOOLCHAIN=local",
		"GOCACHE=" + filepath.Join(workDir, "gocache"),
	}
}

type cpioEntry struct {
	Mode uint32
	Data []byte
}

func buildCPIONewc(entries map[string]cpioEntry) ([]byte, error) {
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	sort.Strings(names)
	var out bytes.Buffer
	ino := uint32(1)
	for _, name := range names {
		e := entries[name]
		if err := writeCPIOEntry(&out, ino, name, e.Mode, e.Data); err != nil {
			return nil, err
		}
		ino++
	}
	if err := writeCPIOEntry(&out, ino, "TRAILER!!!", 0, nil); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func writeCPIOEntry(w *bytes.Buffer, ino uint32, name string, mode uint32, data []byte) error {
	mode, err := normalizeCPIOEntryMode(name, mode)
	if err != nil {
		return err
	}
	if err := writeCPIOHeaderAndName(w, ino, name, mode, len(data)); err != nil {
		return err
	}
	return writeCPIOData(w, data)
}

func normalizeCPIOEntryMode(name string, mode uint32) (uint32, error) {
	if strings.TrimSpace(name) == "" {
		return 0, errors.New("cpio entry name required")
	}
	if name == "TRAILER!!!" {
		return 0, nil
	}
	if mode == 0 {
		return 0o100644, nil
	}
	if mode&0o170000 == 0 {
		return mode | 0o100000, nil
	}
	return mode, nil
}

func writeCPIOHeaderAndName(w *bytes.Buffer, ino uint32, name string, mode uint32, dataLen int) error {
	header := fmt.Sprintf("070701%08x%08x%08x%08x%08x%08x%08x%08x%08x%08x%08x%08x%08x",
		ino,
		mode,
		0,
		0,
		1,
		0,
		uint32(dataLen),
		0,
		0,
		0,
		0,
		uint32(len(name)+1),
		0,
	)
	if _, err := w.WriteString(header); err != nil {
		return err
	}
	if _, err := w.WriteString(name); err != nil {
		return err
	}
	if err := w.WriteByte(0); err != nil {
		return err
	}
	pad4(w)
	return nil
}

func writeCPIOData(w *bytes.Buffer, data []byte) error {
	if len(data) == 0 {
		return nil
	}
	if _, err := w.Write(data); err != nil {
		return err
	}
	pad4(w)
	return nil
}

func pad4(w *bytes.Buffer) {
	for w.Len()%4 != 0 {
		_ = w.WriteByte(0)
	}
}
