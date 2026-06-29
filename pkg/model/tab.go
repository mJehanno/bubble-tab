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
		// initialized reports whether the body's Init() has already been run.
		initialized bool
		// factory is an optional deferred constructor; when set and content is
		// nil, the body is constructed on first access.
		factory func() tea.Model
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

// Initialized reports whether the tab's body Init() has already been run.
func (t Tab) Initialized() bool {
	return t.initialized
}

func (t *Tab) SetState(state TabState) {
	t.state = state
}

func (t *Tab) SetHasPermission(hasPerm bool) {
	t.hasPermission = hasPerm
}

// SetInitialized records whether the tab's body Init() has already been run.
func (t *Tab) SetInitialized(initialized bool) {
	t.initialized = initialized
}

// SetBody replaces the tab's child model, allowing callers to persist an
// updated model after forwarding Update to it.
func (t *Tab) SetBody(m tea.Model) {
	t.content = m
}

// EnsureBody lazily constructs the tab's body from its factory the first time
// it is needed. It is safe to call multiple times: the body is only built once,
// and tabs without a factory (or with an eagerly set body) are left untouched.
func (t *Tab) EnsureBody() {
	if t.content == nil && t.factory != nil {
		t.content = t.factory()
	}
}

// NewTab builds a Tab with sensible defaults (empty name, hasPermission=true,
// state=Inactive) and then applies the given options, which may override those
// defaults.
func NewTab(options ...TabOption) *Tab {
	tab := &Tab{
		state:         Inactive,
		hasPermission: true,
	}
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

// WithBody sets the tab's body eagerly to the given model.
func WithBody(body tea.Model) TabOption {
	return func(t *Tab) {
		t.content = body
	}
}

// WithBodyFunc defers construction of the tab's body to the given factory,
// which is invoked on first activation (leaving content nil until then).
func WithBodyFunc(f func() tea.Model) TabOption {
	return func(t *Tab) {
		t.factory = f
	}
}

func WithHasPermission(hasPerm bool) TabOption {
	return func(t *Tab) {
		t.hasPermission = hasPerm
	}
}
