package main

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type shellLeaderBinding struct {
	Sequence    []string
	Group       string
	Label       string
	Description string
	Action      paletteActionMsg
}

type shellLeaderChoice struct {
	Key         string
	Label       string
	Description string
	Completes   bool
}

type shellLeaderState struct {
	active   bool
	prefix   []string
	bindings []shellLeaderBinding
	choices  []shellLeaderChoice
}

func newShellLeaderState(bindings []shellLeaderBinding) shellLeaderState {
	state := shellLeaderState{bindings: append([]shellLeaderBinding(nil), bindings...)}
	state.choices = state.choicesForPrefix(nil)
	return state
}

func (s shellLeaderState) Active() bool {
	return s.active
}

func (s *shellLeaderState) Start() {
	s.active = true
	s.prefix = nil
	s.choices = s.choicesForPrefix(nil)
}

func (s *shellLeaderState) Rebind(bindings []shellLeaderBinding) {
	s.bindings = append([]shellLeaderBinding(nil), bindings...)
	if s.active {
		s.choices = s.choicesForPrefix(s.prefix)
		if len(s.choices) == 0 {
			s.Abort()
		}
		return
	}
	s.choices = s.choicesForPrefix(nil)
}

func (s *shellLeaderState) Abort() {
	s.active = false
	s.prefix = nil
	s.choices = s.choicesForPrefix(nil)
}

func (s *shellLeaderState) Step(token string) (paletteActionMsg, bool) {
	token = normalizeLeaderToken(token)
	if token == "" {
		s.Abort()
		return paletteActionMsg{}, false
	}
	nextPrefix := append(append([]string(nil), s.prefix...), token)
	for _, binding := range s.bindings {
		if sequenceEqual(binding.Sequence, nextPrefix) {
			s.Abort()
			return binding.Action, true
		}
	}
	choices := s.choicesForPrefix(nextPrefix)
	if len(choices) == 0 {
		s.Abort()
		return paletteActionMsg{}, false
	}
	s.prefix = nextPrefix
	s.choices = choices
	return paletteActionMsg{}, false
}

func (s shellLeaderState) SequenceLabel() string {
	if len(s.prefix) == 0 {
		return "(none)"
	}
	return strings.Join(s.prefix, " ")
}

func (s shellLeaderState) Choices() []shellLeaderChoice {
	out := make([]shellLeaderChoice, len(s.choices))
	copy(out, s.choices)
	return out
}

func (s shellLeaderState) choicesForPrefix(prefix []string) []shellLeaderChoice {
	type agg struct {
		label       string
		description string
		completes   bool
	}
	byKey := map[string]agg{}
	for _, binding := range s.bindings {
		if len(binding.Sequence) <= len(prefix) || !sequenceHasPrefix(binding.Sequence, prefix) {
			continue
		}
		next := binding.Sequence[len(prefix)]
		existing := byKey[next]
		candidate := agg{
			label:       firstNonEmpty(binding.Group, binding.Label),
			description: binding.Description,
			completes:   len(binding.Sequence) == len(prefix)+1,
		}
		if strings.TrimSpace(existing.label) == "" {
			byKey[next] = candidate
			continue
		}
		if existing.label != candidate.label {
			existing.label = "(group)"
		}
		if existing.description != candidate.description {
			existing.description = "multiple actions"
		}
		existing.completes = existing.completes || candidate.completes
		byKey[next] = existing
	}
	keys := make([]string, 0, len(byKey))
	for key := range byKey {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	choices := make([]shellLeaderChoice, 0, len(keys))
	for _, key := range keys {
		entry := byKey[key]
		choices = append(choices, shellLeaderChoice{Key: key, Label: entry.label, Description: entry.description, Completes: entry.completes})
	}
	return choices
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if text := strings.TrimSpace(value); text != "" {
			return text
		}
	}
	return ""
}

func sequenceHasPrefix(sequence []string, prefix []string) bool {
	if len(prefix) > len(sequence) {
		return false
	}
	for i := range prefix {
		if normalizeLeaderToken(sequence[i]) != normalizeLeaderToken(prefix[i]) {
			return false
		}
	}
	return true
}

func sequenceEqual(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if normalizeLeaderToken(a[i]) != normalizeLeaderToken(b[i]) {
			return false
		}
	}
	return true
}

func normalizeLeaderToken(token string) string {
	token = strings.ToLower(strings.TrimSpace(token))
	if token == " " {
		return "space"
	}
	return token
}

func leaderTokenFromKey(key tea.KeyMsg) (string, bool) {
	if key.Type == tea.KeyRunes && len(key.Runes) == 1 {
		r := key.Runes[0]
		if r >= 32 && r != 127 {
			return normalizeLeaderToken(string(r)), true
		}
	}
	if key.Type == tea.KeySpace {
		return "space", true
	}
	if label := normalizeLeaderToken(key.String()); label != "" {
		return label, true
	}
	return "", false
}

func formatLeaderInvalidKeyMessage(sequence []string, keyLabel string) string {
	prefix := ""
	if len(sequence) > 0 {
		prefix = strings.Join(sequence, " ") + " "
	}
	return fmt.Sprintf("Leader aborted: %q is invalid after sequence %s", keyLabel, strings.TrimSpace(prefix))
}

func (m *shellModel) setLeaderKey(configured string) error {
	configured = strings.ToLower(strings.TrimSpace(configured))
	if configured == "" {
		configured = "space"
	}
	binding, err := shellLeaderStartKeyBinding(configured)
	if err != nil {
		return err
	}
	m.leaderKeyConfig = configured
	m.leaderKeyInvalid = ""
	m.keys.LeaderStart = binding
	return nil
}

func normalizeLeaderKeyConfigValue(raw string) string {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if normalized == "" || normalized == "default" {
		return "space"
	}
	return normalized
}

func (m *shellModel) configureLeaderKey(raw string) error {
	configured := normalizeLeaderKeyConfigValue(raw)
	if err := m.setLeaderKey(configured); err != nil {
		return err
	}
	m.persistWorkbenchState()
	return nil
}
