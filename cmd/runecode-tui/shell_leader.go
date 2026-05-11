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

type leaderChoiceAggregate struct {
	label       string
	description string
	completes   bool
}

const leaderPrefixSeparator = "::"

type shellLeaderState struct {
	active          bool
	prefix          []string
	bindings        []shellLeaderBinding
	choices         []shellLeaderChoice
	exactActions    map[string]paletteActionMsg
	choicesByPrefix map[string][]shellLeaderChoice
}

func newShellLeaderState(bindings []shellLeaderBinding) shellLeaderState {
	state := shellLeaderState{}
	state.Rebind(bindings)
	return state
}

func (s shellLeaderState) Active() bool {
	return s.active
}

func (s *shellLeaderState) Start() {
	s.active = true
	s.prefix = nil
	s.choices = cloneLeaderChoices(s.choicesByPrefix[""])
}

func (s *shellLeaderState) Rebind(bindings []shellLeaderBinding) {
	s.bindings = append([]shellLeaderBinding(nil), bindings...)
	s.exactActions, s.choicesByPrefix = buildShellLeaderIndexes(s.bindings)
	if s.active {
		choices := cloneLeaderChoices(s.choicesByPrefix[leaderPrefixKey(s.prefix)])
		if len(choices) == 0 {
			s.Abort()
			return
		}
		s.choices = choices
		return
	}
	s.choices = cloneLeaderChoices(s.choicesByPrefix[""])
}

func (s *shellLeaderState) Abort() {
	s.active = false
	s.prefix = nil
	s.choices = cloneLeaderChoices(s.choicesByPrefix[""])
}

func (s *shellLeaderState) Step(token string) (paletteActionMsg, bool) {
	token = normalizeLeaderToken(token)
	if token == "" {
		s.Abort()
		return paletteActionMsg{}, false
	}
	nextPrefix := append(append([]string(nil), s.prefix...), token)
	if action, ok := s.exactActions[leaderPrefixKey(nextPrefix)]; ok {
		s.Abort()
		return action, true
	}
	choices := cloneLeaderChoices(s.choicesByPrefix[leaderPrefixKey(nextPrefix)])
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

func buildShellLeaderIndexes(bindings []shellLeaderBinding) (map[string]paletteActionMsg, map[string][]shellLeaderChoice) {
	exactActions := map[string]paletteActionMsg{}
	choicesByPrefix := map[string]map[string]leaderChoiceAggregate{"": {}}
	for _, binding := range bindings {
		registerLeaderBindingIndexes(binding, exactActions, choicesByPrefix)
	}
	choices := make(map[string][]shellLeaderChoice, len(choicesByPrefix))
	for prefixKey, byKey := range choicesByPrefix {
		keys := make([]string, 0, len(byKey))
		for key := range byKey {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		entries := make([]shellLeaderChoice, 0, len(keys))
		for _, key := range keys {
			entry := byKey[key]
			entries = append(entries, shellLeaderChoice{Key: key, Label: entry.label, Description: entry.description, Completes: entry.completes})
		}
		choices[prefixKey] = entries
	}
	if _, ok := choices[""]; !ok {
		choices[""] = nil
	}
	return exactActions, choices
}

func registerLeaderBindingIndexes(binding shellLeaderBinding, exactActions map[string]paletteActionMsg, choicesByPrefix map[string]map[string]leaderChoiceAggregate) {
	sequence := normalizeLeaderSequence(binding.Sequence)
	if len(sequence) == 0 {
		return
	}
	exactActions[leaderPrefixKey(sequence)] = binding.Action
	for i := 0; i < len(sequence); i++ {
		prefixKey := leaderPrefixKey(sequence[:i])
		next := sequence[i]
		choicesByPrefix[prefixKey] = mergeLeaderChoiceAggregate(choicesByPrefix[prefixKey], next, binding, len(sequence) == i+1)
	}
}

func mergeLeaderChoiceAggregate(byKey map[string]leaderChoiceAggregate, next string, binding shellLeaderBinding, completes bool) map[string]leaderChoiceAggregate {
	if byKey == nil {
		byKey = map[string]leaderChoiceAggregate{}
	}
	existing := byKey[next]
	candidate := leaderChoiceAggregate{
		label:       firstNonEmpty(binding.Group, binding.Label),
		description: binding.Description,
		completes:   completes,
	}
	if strings.TrimSpace(existing.label) == "" {
		byKey[next] = candidate
		return byKey
	}
	if existing.label != candidate.label {
		existing.label = "(group)"
	}
	if existing.description != candidate.description {
		existing.description = "multiple actions"
	}
	existing.completes = existing.completes || candidate.completes
	byKey[next] = existing
	return byKey
}

func cloneLeaderChoices(in []shellLeaderChoice) []shellLeaderChoice {
	if len(in) == 0 {
		return nil
	}
	out := make([]shellLeaderChoice, len(in))
	copy(out, in)
	return out
}

func normalizeLeaderSequence(sequence []string) []string {
	out := make([]string, 0, len(sequence))
	for _, token := range sequence {
		if normalized := normalizeLeaderToken(token); normalized != "" {
			out = append(out, normalized)
		}
	}
	return out
}

func leaderPrefixKey(prefix []string) string {
	if len(prefix) == 0 {
		return ""
	}
	return strings.Join(prefix, leaderPrefixSeparator)
}

func shellLeaderBindingsSignature(bindings []shellLeaderBinding) string {
	if len(bindings) == 0 {
		return ""
	}
	parts := make([]string, 0, len(bindings))
	for _, binding := range bindings {
		parts = append(parts, strings.Join([]string{
			strings.Join(normalizeLeaderSequence(binding.Sequence), "/"),
			strings.TrimSpace(binding.Group),
			strings.TrimSpace(binding.Label),
			strings.TrimSpace(binding.Description),
			string(binding.Action.Verb),
			binding.Action.Target.Kind,
			string(binding.Action.Target.RouteID),
			strings.TrimSpace(binding.Action.Target.CommandID),
			strings.Join(binding.Action.Target.CommandArgs, "/"),
		}, "|"))
	}
	return strings.Join(parts, leaderPrefixSeparator)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if text := strings.TrimSpace(value); text != "" {
			return text
		}
	}
	return ""
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
