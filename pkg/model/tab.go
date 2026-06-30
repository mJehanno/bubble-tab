// Package model defines the Tab type and its options, which are the building
// blocks assembled by the root bubbletab package to form a tabbed interface.
// Each Tab wraps an optional child tea.Model (its "body") and tracks rendering
// metadata: display name, visual state, access permission, and whether the
// body's Init has already run.
package model

import tea "charm.land/bubbletea/v2"

// TabState is the visual and behavioral state of a Tab.
// The three valid values are Active, Inactive, and Disabled.
type TabState string

const (
	// Active is the state of the currently focused tab. Only one tab is Active
	// at a time; it receives forwarded Update messages and its body is rendered.
	Active TabState = "active"
	// Inactive is the default state for tabs that are not currently focused but
	// can be navigated to.
	Inactive TabState = "inactive"
	// Disabled marks a tab that cannot be selected. Navigation (Tab, Shift+Tab,
	// number-key jumps, and mouse clicks) skips over Disabled tabs.
	Disabled TabState = "disabled"
)

// Tab is a single entry in a tabbed interface. It pairs a display name with an
// optional child tea.Model body, tracks visual state, and records whether the
// body has been initialized.
type Tab struct {
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

// TabOption is a functional option applied to a Tab during construction by
// NewTab. Use the provided With* constructors to build option values.
type TabOption func(*Tab)

// Name returns the display label rendered in the tab header.
func (t Tab) Name() string {
	return t.name
}

// State returns the current visual state of the tab (Active, Inactive, or Disabled).
func (t Tab) State() TabState {
	return t.state
}

// Body returns the tab's child model, or nil if no body has been set or built yet.
// For lazily-constructed tabs (created with WithBodyFunc), Body returns nil until
// EnsureBody is called for the first time.
func (t Tab) Body() tea.Model {
	return t.content
}

// HasPermission reports whether the tab's body should be rendered. When false,
// the header is still shown but the body area is left blank regardless of state.
func (t Tab) HasPermission() bool {
	return t.hasPermission
}

// Initialized reports whether the tab's body Init() has already been run.
func (t Tab) Initialized() bool {
	return t.initialized
}

// SetState updates the tab's visual state. TabModel calls this to mark a tab
// Active on activation and Inactive when focus moves away.
func (t *Tab) SetState(state TabState) {
	t.state = state
}

// SetHasPermission controls whether the tab's body is rendered. Passing false
// hides the body while keeping the header visible, which is useful for
// permission-gated content.
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

// WithName sets the display label shown in the tab header.
func WithName(name string) TabOption {
	return func(t *Tab) {
		t.name = name
	}
}

// WithState sets the initial visual state of the tab. In practice callers
// use this only to set a tab to Disabled; the Active/Inactive transitions are
// managed automatically by TabModel.
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

// WithHasPermission controls whether the tab's body is rendered. Defaults to
// true. Pass false to show the tab header but hide its content (e.g. for
// permission-gated views).
func WithHasPermission(hasPerm bool) TabOption {
	return func(t *Tab) {
		t.hasPermission = hasPerm
	}
}
