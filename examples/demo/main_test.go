package main

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

// TestApp_UpdateDoesNotPanic drives the root model the way the Bubble Tea
// runtime does: it stores the model behind the tea.Model interface and feeds it
// messages. This guards against the type-assertion panic that occurs when the
// wrapper stores TabModel's value-receiver result behind the wrong type.
func TestApp_UpdateDoesNotPanic(t *testing.T) {
	var m tea.Model = newApp()
	m.Init()

	msgs := []tea.Msg{
		tea.KeyPressMsg{Code: tea.KeyTab},                     // next tab
		tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift},  // previous tab
		tea.KeyPressMsg{Code: '2'},                            // jump to tab 2
		tea.KeyPressMsg{Code: 'a'},                            // forwarded to body
		tea.WindowSizeMsg{Width: 80, Height: 24},              // resize
	}

	var cmd tea.Cmd
	for _, msg := range msgs {
		m, cmd = m.Update(msg)
		_ = cmd
		_ = m.View()
	}
}
