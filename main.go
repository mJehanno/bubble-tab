// Package bubbletab provides a Bubble Tea model that renders a row of tabs and
// forwards messages to the body of the currently active tab. Tab bodies are
// initialized lazily on first activation and their state is preserved across
// switches.
package bubbletab

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/mJehanno/bubble-tab/pkg/model"
	thm "github.com/mJehanno/bubble-tab/pkg/theme"
)

type (
	// TabModel is a Bubble Tea model that displays a horizontal row of tab
	// headers and the body of the currently selected tab.
	TabModel struct {
		tabs    []model.Tab
		current int
		theme   thm.Theme
		styles  thm.Styles
		width   int
		height  int
		// haveSize reports whether a WindowSizeMsg has been received yet; until
		// then width/height are meaningless and must not be replayed.
		haveSize bool
		// paddingX and paddingY are the view's leading left/top offsets, used to
		// translate mouse coordinates during header hit-testing. They are 0 today
		// (the model applies no padding) but wired so future padding works.
		paddingX int
		paddingY int
		keymap   KeyMap
	}
	// TabModelOption configures a TabModel during construction with New.
	TabModelOption func(*TabModel)
)

// tabClickMsg is emitted by the view's mouse handler when a tab header is
// clicked; Update reacts by activating the corresponding tab.
type tabClickMsg struct{ index int }

// WithTabs sets the tabs displayed by the model.
func WithTabs(tabs []model.Tab) TabModelOption {
	return func(tm *TabModel) {
		tm.tabs = tabs
	}
}

// WithCurrent selects the initially active tab by index.
func WithCurrent(current int) TabModelOption {
	return func(tm *TabModel) {
		tm.current = current
	}
}

// WithTheme sets the theme used to derive the model's styles.
func WithTheme(theme thm.Theme) TabModelOption {
	return func(tm *TabModel) {
		tm.theme = theme
	}
}

// WithKeyMap overrides the default navigation key bindings.
func WithKeyMap(keymap KeyMap) TabModelOption {
	return func(tm *TabModel) {
		tm.keymap = keymap
	}
}

// New builds a TabModel from the given options. If no theme is supplied it
// defaults to Catppuccin, and the derived styles are always populated. The
// current index is clamped to a valid tab and that tab's state is set to
// model.Active so the model is consistent without extra caller setup.
func New(options ...TabModelOption) *TabModel {
	tabModel := &TabModel{
		theme:  thm.Catppuccin(),
		keymap: DefaultKeyMap(),
	}
	for _, o := range options {
		o(tabModel)
	}

	tabModel.styles = tabModel.theme.Styles()
	tabModel.clampCurrent()

	// The current tab is active by definition; mark it so via a pointer into the
	// slice element so the change persists.
	if len(tabModel.tabs) > 0 {
		(&tabModel.tabs[tabModel.current]).SetState(model.Active)
	}

	return tabModel
}

// clampCurrent keeps current within the bounds of the tabs slice.
func (t *TabModel) clampCurrent() {
	if len(t.tabs) == 0 {
		t.current = 0
		return
	}
	if t.current < 0 {
		t.current = 0
	}
	if t.current >= len(t.tabs) {
		t.current = len(t.tabs) - 1
	}
}

// buildAndPrime ensures the body of the tab at index exists and is ready to be
// shown for the first time. When the body is constructed lazily by this call
// (it was nil before EnsureBody and non-nil after) and a window size has
// already been observed, the cached size is replayed to the new body so it is
// sized correctly on its first paint — instead of waiting for the next resize.
// The body is then initialized exactly once. The returned command batches any
// size-update command with the body's Init command (nil-safe).
func (t *TabModel) buildAndPrime(index int) tea.Cmd {
	if index < 0 || index >= len(t.tabs) {
		return nil
	}

	tab := &t.tabs[index]
	wasNil := tab.Body() == nil
	tab.EnsureBody()
	if tab.Body() == nil {
		return nil
	}

	var sizeCmd, initCmd tea.Cmd

	// Replay the most recent size only to bodies we just built, and only once a
	// real size has arrived — never send a stale or zero size.
	if wasNil && t.haveSize {
		updated, cmd := tab.Body().Update(tea.WindowSizeMsg{Width: t.width, Height: t.height})
		tab.SetBody(updated)
		sizeCmd = cmd
	}

	if !tab.Initialized() {
		initCmd = tab.Body().Init()
		tab.SetInitialized(true)
	}

	return tea.Batch(sizeCmd, initCmd)
}

// Init initializes only the current tab's body, leaving other tabs to be
// initialized lazily the first time they are activated.
func (t TabModel) Init() tea.Cmd {
	if len(t.tabs) == 0 {
		return nil
	}
	return t.buildAndPrime(t.current)
}

// activate switches focus from the current tab to the tab at index. It marks
// the old tab inactive and the new one active, ensures the new body exists, and
// initializes that body exactly once (returning its Init command). When the tab
// was already initialized its cached state is reused and nil is returned.
func (t *TabModel) activate(index int) tea.Cmd {
	if index < 0 || index >= len(t.tabs) {
		return nil
	}

	(&t.tabs[t.current]).SetState(model.Inactive)
	t.current = index
	(&t.tabs[t.current]).SetState(model.Active)

	// Build the body if needed, prime it with the cached size, and init once.
	return t.buildAndPrime(t.current)
}

// moveForward returns the index of the next non-disabled tab after current,
// wrapping around. If no enabled tab exists (other than possibly current), it
// returns current. The search is bounded to avoid infinite loops.
func (t TabModel) moveForward() int {
	n := len(t.tabs)
	for step := 1; step <= n; step++ {
		candidate := (t.current + step) % n
		if t.tabs[candidate].State() != model.Disabled {
			return candidate
		}
	}
	return t.current
}

// moveBackward returns the index of the previous non-disabled tab before
// current, wrapping around. If no enabled tab exists (other than possibly
// current), it returns current. The search is bounded to avoid infinite loops.
func (t TabModel) moveBackward() int {
	n := len(t.tabs)
	for step := 1; step <= n; step++ {
		candidate := (t.current - step%n + n) % n
		if t.tabs[candidate].State() != model.Disabled {
			return candidate
		}
	}
	return t.current
}

// Update handles navigation keys and tab clicks itself; every other message is
// forwarded to the active tab's body so its state is updated and preserved.
func (t TabModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if len(t.tabs) == 0 {
		return t, nil
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if cmd, handled := t.handleKey(msg); handled {
			return t, cmd
		}
	case tabClickMsg:
		if msg.index >= 0 && msg.index < len(t.tabs) &&
			t.tabs[msg.index].State() != model.Disabled {
			return t, t.activate(msg.index)
		}
		return t, nil
	case tea.WindowSizeMsg:
		return t.handleResize(msg)
	}

	// Default: forward to the active tab's body and persist the result. If the
	// body is built lazily here, prime it with the cached size first so it is
	// sized before it processes this message.
	primeCmd := t.buildAndPrime(t.current)
	cur := &t.tabs[t.current]
	if cur.Body() == nil {
		return t, primeCmd
	}
	updated, cmd := cur.Body().Update(msg)
	cur.SetBody(updated)
	return t, tea.Batch(primeCmd, cmd)
}

// handleKey processes navigation key presses. It reports whether the key was a
// navigation key (and thus consumed); when false the caller forwards the
// message to the active body instead.
func (t *TabModel) handleKey(msg tea.KeyPressMsg) (tea.Cmd, bool) {
	switch {
	case msg.Code == tea.KeyTab && msg.Mod&tea.ModShift != 0:
		return t.activate(t.moveBackward()), true
	case msg.Code == tea.KeyTab:
		return t.activate(t.moveForward()), true
	case msg.Code >= '1' && msg.Code <= '9':
		index := int(msg.Code - '1')
		if index < len(t.tabs) && t.tabs[index].State() != model.Disabled {
			return t.activate(index), true
		}
		// Out-of-range or disabled jump: consume the key, do nothing.
		return nil, true
	}
	return nil, false
}

// handleResize stores the new dimensions and forwards the size message to every
// already-constructed tab body so off-screen tabs are sized correctly before
// they are first shown. Factory tabs that have not been built are left alone,
// except the current one, which is ensured.
func (t TabModel) handleResize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	t.width = msg.Width
	t.height = msg.Height
	t.haveSize = true

	(&t.tabs[t.current]).EnsureBody()

	var cmds []tea.Cmd
	for i := range t.tabs {
		tab := &t.tabs[i]
		if tab.Body() == nil {
			continue
		}
		updated, cmd := tab.Body().Update(msg)
		tab.SetBody(updated)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return t, tea.Batch(cmds...)
}

// View renders the header row followed by the active tab's body (when the tab
// grants permission). The returned view carries a mouse handler that maps
// clicks on the header row to tab activations.
func (t TabModel) View() tea.View {
	if len(t.tabs) == 0 {
		return tea.NewView("")
	}

	headers := make([]string, len(t.tabs))
	for i := range t.tabs {
		tab := &t.tabs[i]
		headers[i] = t.styles.Header(tab.State()).Render(tab.Name())
	}

	var content strings.Builder
	content.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, headers...))
	content.WriteString("\n")

	cur := &t.tabs[t.current]
	if cur.HasPermission() {
		cur.EnsureBody()
		if cur.Body() != nil {
			content.WriteString(
				t.styles.Body(cur.State()).Render(cur.Body().View().Content),
			)
		}
	}

	view := tea.NewView(content.String())
	view.OnMouse = makeHeaderMouseHandler(headers, t.paddingX, t.paddingY)
	return view
}

// makeHeaderMouseHandler builds a mouse handler that hit-tests clicks against
// the rendered header strings. A left click anywhere within a header's
// rendered block (its horizontal extent and the header's height, which may be
// more than one row when borders are drawn) emits a tabClickMsg for that tab.
//
// offsetX and offsetY are the view's leading left/top padding: click
// coordinates are translated by these offsets before hit-testing so that an
// indented or padded view still maps clicks to the correct tab.
func makeHeaderMouseHandler(headers []string, offsetX, offsetY int) func(tea.MouseMsg) tea.Cmd {
	// Precompute the [start, end) X range of each header for cheap hit-testing.
	type span struct{ start, end int }
	spans := make([]span, len(headers))
	x, height := 0, 0
	for i, h := range headers {
		w := lipgloss.Width(h)
		spans[i] = span{start: x, end: x + w}
		x += w
		height = max(height, lipgloss.Height(h))
	}

	return func(msg tea.MouseMsg) tea.Cmd {
		click, ok := msg.(tea.MouseClickMsg)
		if !ok {
			return nil
		}
		m := click.Mouse()
		if m.Button != tea.MouseLeft {
			return nil
		}
		// Translate the click into header-local coordinates, rejecting clicks
		// that fall in the view's leading padding (negative after the shift).
		localX := m.X - offsetX
		localY := m.Y - offsetY
		if localX < 0 || localY < 0 || localY >= height {
			return nil
		}
		for i, s := range spans {
			if localX >= s.start && localX < s.end {
				index := i
				return func() tea.Msg { return tabClickMsg{index: index} }
			}
		}
		return nil
	}
}
