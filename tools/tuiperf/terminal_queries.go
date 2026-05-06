//go:build linux

package main

import (
	"io"
	"strings"
)

const stableTUITerm = "xterm-256color"

const (
	terminalCPRQuery                = "\x1b[6n"
	terminalCPRResponse             = "\x1b[1;1R"
	terminalBackgroundColorQuery    = "\x1b]11;?\x1b\\"
	terminalBackgroundColorResponse = "\x1b]11;rgb:0000/0000/0000\x1b\\"
)

type terminalQueryResponder struct {
	io.ReadCloser
	w       io.Writer
	pending string
}

func newTerminalQueryResponder(r io.ReadCloser, w io.Writer) io.ReadCloser {
	return &terminalQueryResponder{ReadCloser: r, w: w}
}

func (r *terminalQueryResponder) Read(p []byte) (int, error) {
	n, err := r.ReadCloser.Read(p)
	if n > 0 {
		r.respondToTerminalQueries(p[:n])
	}
	return n, err
}

func (r *terminalQueryResponder) respondToTerminalQueries(chunk []byte) {
	text := r.pending + string(chunk)
	if strings.Contains(text, terminalCPRQuery) {
		_, _ = io.WriteString(r.w, terminalCPRResponse)
	}
	if strings.Contains(text, terminalBackgroundColorQuery) {
		_, _ = io.WriteString(r.w, terminalBackgroundColorResponse)
	}
	r.pending = terminalQueryTail(text)
}

func terminalQueryTail(text string) string {
	keep := len(terminalBackgroundColorQuery) - 1
	if len(terminalCPRQuery) > len(terminalBackgroundColorQuery) {
		keep = len(terminalCPRQuery) - 1
	}
	if len(text) <= keep {
		return text
	}
	return text[len(text)-keep:]
}

func stableTTYEnv(base []string) []string {
	filtered := make([]string, 0, len(base)+1)
	hasTerm := false
	for _, entry := range base {
		if strings.HasPrefix(entry, "TERM=") {
			filtered = append(filtered, "TERM="+stableTUITerm)
			hasTerm = true
			continue
		}
		filtered = append(filtered, entry)
	}
	if !hasTerm {
		filtered = append(filtered, "TERM="+stableTUITerm)
	}
	return filtered
}
