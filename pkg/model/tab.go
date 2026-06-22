package model

import tea "charm.land/bubbletea/v2"

const (
	Active   TabState = "active"
	Inactive TabState = "inactive"
	Disabled TabState = "disabled"
)

type (
	TabState string
	Tab      struct {
		name          string
		state         TabState
		content       tea.Model
		hasPermission bool
	}
	TabOption func(*Tab)
)

func (t Tab) Name() string {
	return t.name
}
func (t Tab) State() TabState {
	return t.state
}
func (t Tab) Body() tea.Model {
	return t.content
}

func (t Tab) HasPermission() bool {
	return t.hasPermission
}

func (t *Tab) SetState(state TabState) {
	t.state = state
}

func (t *Tab) SetHasPermission(hasPerm bool) {
	t.hasPermission = hasPerm
}

func NewTab(options ...TabOption) *Tab {
	tab := new(Tab)
	for _, o := range options {
		o(tab)
	}
	return tab
}

func WithName(name string) TabOption {
	return func(t *Tab) {
		t.name = name
	}
}
func WithState(state TabState) TabOption {
	return func(t *Tab) {
		t.state = state
	}
}

func WithBody(body tea.Model) TabOption {
	return func(t *Tab) {
		t.content = body
	}
}

func WithHasPermission(hasPerm bool) TabOption {
	return func(t *Tab) {
		t.hasPermission = hasPerm
	}
}
