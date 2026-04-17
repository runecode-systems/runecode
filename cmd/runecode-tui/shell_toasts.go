package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
)

type toastLevel string

const (
	toastInfo  toastLevel = "info"
	toastWarn  toastLevel = "warn"
	toastError toastLevel = "error"
)

type toastMessage struct {
	Level toastLevel
	Text  string
}

type shellToastService struct {
	items   []toastMessage
	spin    spinner.Model
	active  bool
	maxSize int
}

func newShellToastService() shellToastService {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	return shellToastService{spin: sp, maxSize: 8}
}

func (s *shellToastService) SetActivity(active bool) {
	s.active = active
}

func (s *shellToastService) Push(level toastLevel, text string) {
	text = sanitizeUIText(text)
	if text == "" {
		return
	}
	s.items = append(s.items, toastMessage{Level: level, Text: text})
	if len(s.items) > s.maxSize {
		s.items = s.items[len(s.items)-s.maxSize:]
	}
}

func (s *shellToastService) Latest() string {
	if len(s.items) == 0 {
		return ""
	}
	last := s.items[len(s.items)-1]
	return fmt.Sprintf("%s: %s", strings.ToUpper(string(last.Level)), last.Text)
}

func (s *shellToastService) ActivityIndicator() string {
	if !s.active {
		return ""
	}
	return s.spin.View() + " running"
}
