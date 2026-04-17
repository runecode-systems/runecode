package main

import (
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

type composeTextarea struct {
	model textarea.Model
	set   bool
}

func newComposeTextarea() composeTextarea {
	t := textarea.New()
	t.Placeholder = "Type a message"
	t.CharLimit = 4000
	t.Prompt = "┃ "
	t.SetHeight(3)
	return composeTextarea{model: t, set: true}
}

func (c *composeTextarea) ensure() {
	if c.set {
		return
	}
	*c = newComposeTextarea()
}

func (c *composeTextarea) Value() string {
	c.ensure()
	return c.model.Value()
}

func (c *composeTextarea) SetValue(value string) {
	c.ensure()
	c.model.SetValue(value)
}

func (c *composeTextarea) Focus() {
	c.ensure()
	c.model.Focus()
}

func (c *composeTextarea) Blur() {
	c.ensure()
	c.model.Blur()
}

func (c *composeTextarea) BubbleUpdate(msg tea.Msg) {
	c.ensure()
	c.model, _ = c.model.Update(msg)
}

func (c *composeTextarea) View() string {
	c.ensure()
	return c.model.View()
}
