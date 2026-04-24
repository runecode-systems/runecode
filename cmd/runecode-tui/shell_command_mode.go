package main

import (
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
)

const commandModeDisplayLimit = 120

type shellCommandModeState struct {
	active bool
	draft  string
	error  string
}

func (s shellCommandModeState) Active() bool {
	return s.active
}

func (s shellCommandModeState) Open() shellCommandModeState {
	s.active = true
	s.draft = ""
	s.error = ""
	return s
}

func (s shellCommandModeState) Abort() shellCommandModeState {
	s.active = false
	s.draft = ""
	s.error = ""
	return s
}

func (s shellCommandModeState) SetError(err string) shellCommandModeState {
	s.active = true
	s.error = clampCommandModeDisplayText(err)
	return s
}

func (s shellCommandModeState) Append(r string) shellCommandModeState {
	if r == "" {
		return s
	}
	s.draft += r
	s.error = ""
	return s
}

func (s shellCommandModeState) Backspace() shellCommandModeState {
	if len(s.draft) == 0 {
		return s
	}
	_, size := utf8.DecodeLastRuneInString(s.draft)
	if size <= 0 {
		return s
	}
	s.draft = s.draft[:len(s.draft)-size]
	s.error = ""
	return s
}

func (s shellCommandModeState) RenderPrompt() string {
	draft := clampCommandModeDisplayText(redactSecrets(s.draft))
	errText := clampCommandModeDisplayText(s.error)
	if s.active {
		if strings.TrimSpace(errText) != "" {
			return ":[input hidden]  [error: " + errText + "]"
		}
		return ":" + draft
	}
	if strings.TrimSpace(errText) != "" {
		return "command error: " + errText
	}
	return ""
}

func clampCommandModeDisplayText(text string) string {
	text = sanitizeUIText(strings.TrimSpace(text))
	if len(text) <= commandModeDisplayLimit {
		return text
	}
	return strings.TrimSpace(text[:commandModeDisplayLimit-3]) + "..."
}

func (m shellModel) handleOpenCommandModeKey(key tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if key.String() != ":" {
		return m, nil, false
	}
	if !m.shellPowerKeysAllowed() {
		return m, nil, false
	}
	if m.commandMode.Active() {
		return m, nil, true
	}
	m.commandMode = m.commandMode.Open()
	if m.palette.IsOpen() {
		m.palette = m.palette.Close()
	}
	m.syncOverlayStack()
	return m, nil, true
}

func (m shellModel) handleCommandModeActiveKey(key tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if !m.commandMode.Active() {
		return m, nil, false
	}
	switch {
	case key.Type == tea.KeyEsc:
		m.commandMode = m.commandMode.Abort()
		return m, nil, true
	case key.Type == tea.KeyBackspace || key.Type == tea.KeyDelete:
		m.commandMode = m.commandMode.Backspace()
		return m, nil, true
	case key.Type == tea.KeyEnter:
		action, err := m.actions.resolveCommandModeDraft(m.commandMode.draft, m)
		if err != nil {
			m.commandMode = m.commandMode.SetError(err.Error())
			return m, nil, true
		}
		updated, cmd := m.applyPaletteAction(action)
		shell := updated.(shellModel)
		if strings.TrimSpace(shell.commandMode.error) != "" {
			return shell, cmd, true
		}
		if shell.quitConfirm.active {
			return shell, cmd, true
		}
		shell.commandMode = shell.commandMode.Abort()
		return shell, cmd, true
	case isTypingKey(key):
		m.commandMode = m.commandMode.Append(key.String())
		return m, nil, true
	default:
		return m, nil, true
	}
}
