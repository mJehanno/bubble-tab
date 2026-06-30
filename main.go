// Package bubbletab provides a tabbed-navigation model for Bubble Tea v2
// (charm.land/bubbletea/v2). It renders a horizontal row of clickable tab
// headers and delegates rendering and message handling to the body of the
// currently active tab.
//
// # Quick start
//
//	tabs := []model.Tab{
//	    *model.NewTab(model.WithName("Home"), model.WithBody(homeModel{})),
//	    *model.NewTab(model.WithName("Settings"), model.WithBody(settingsModel{})),
//	}
//	tm := bubbletab.New(
//	    bubbletab.WithTabs(tabs),
//	    bubbletab.WithTheme(theme.Catppuccin()),
//	)
//	// Embed tm in your root model and delegate Init/Update/View to it.
//
// # Key behaviors
//
//   - Lazy init: a tab body's Init runs exactly once, on first activation.
//     State is preserved across subsequent tab switches; the body is never
//     re-initialized.
//   - Lazy construction: WithBodyFunc(func() tea.Model) defers even the
//     allocation of a body until first activation. Never-visited tabs are
//     never constructed.
//   - Message forwarding: non-navigation messages are forwarded only to the
//     active tab's body. WindowSizeMsg is broadcast to all already-built bodies.
//   - Navigation: Tab (next), Shift+Tab (previous), 1-9 (direct jump by
//     one-based index), and left-click on any header. Disabled tabs are skipped.
//   - Theming: import github.com/mJehanno/bubble-tab/pkg/theme and use one of
//     Gruvbox(), TokyoNight(), or Catppuccin(). Custom themes are supported by
//     building a Theme struct directly.
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
		// mouseMode is applied to the view returned by View. Header clicks are
		// only delivered when mouse reporting is enabled, so this defaults to
		// tea.MouseModeCellMotion; set it to tea.MouseModeNone via WithMouseMode
		// to opt out (e.g. when a parent model owns mouse configuration).
		mouseMode tea.MouseMode
		// hasCustomStyles reports whether WithStyles supplied an explicit Styles
		// value; when false the styles are derived from the theme in New.
		hasCustomStyles bool
		// padX/padY and marginX/marginY are the horizontal/vertical header spacing
		// requested via WithPadding/WithMargin. The has* flags record whether each
		// was set so a value of zero still means "set to zero" rather than "unset".
		padX, padY       int
		marginX, marginY int
		hasPadding       bool
		hasMargin        bool
	}
	// TabModelOption configures a TabModel during construction with New.
	TabModelOption func(*TabModel)
)

// tabClickMsg is emitted by the view's mouse handler when a tab header is
// clicked; Update reacts by activating the corresponding tab.
type tabClickMsg struct{ index int }

// WithTabs sets the ordered list of tabs displayed by the model. Each Tab
// should be constructed with model.NewTab. The slice is stored by value so
// subsequent mutations to the caller's slice do not affect the TabModel.
func WithTabs(tabs []model.Tab) TabModelOption {
	return func(tm *TabModel) {
		tm.tabs = tabs
	}
}

// WithCurrent selects the initially active tab by zero-based index. Values
// outside [0, len(tabs)-1] are clamped silently so the model is always in a
// consistent state after New returns.
func WithCurrent(current int) TabModelOption {
	return func(tm *TabModel) {
		tm.current = current
	}
}

// WithTheme sets the theme whose palette and border configuration are used to
// derive the model's lipgloss styles. Styles are computed once in New; to
// switch variant at runtime (e.g. dark/light toggle) build a new TabModel with
// the updated theme or call theme.Toggle and pass the result here.
func WithTheme(theme thm.Theme) TabModelOption {
	return func(tm *TabModel) {
		tm.theme = theme
	}
}

// WithKeyMap overrides the default navigation key bindings. The KeyMap fields
// are key.Binding values and can be reused in a bubbles/v2 help component to
// display a key-binding legend to the user.
func WithKeyMap(keymap KeyMap) TabModelOption {
	return func(tm *TabModel) {
		tm.keymap = keymap
	}
}

// WithMouseMode overrides the mouse mode applied to the view. The default is
// tea.MouseModeCellMotion so that header clicks work when TabModel is used as
// the root model. Pass tea.MouseModeNone to disable, e.g. when a parent model
// is responsible for mouse configuration.
func WithMouseMode(mode tea.MouseMode) TabModelOption {
	return func(tm *TabModel) {
		tm.mouseMode = mode
	}
}

// WithStyles replaces the model's styles entirely, taking precedence over the
// theme: when supplied, New uses these styles verbatim instead of deriving them
// from the theme's palette. Build a base set with theme.Theme.Styles (or
// theme.New) and adjust the per-state lipgloss styles as needed. Any spacing
// from WithPadding/WithMargin is layered on top of these styles afterwards.
func WithStyles(styles thm.Styles) TabModelOption {
	return func(tm *TabModel) {
		tm.styles = styles
		tm.hasCustomStyles = true
	}
}

// WithPadding sets the padding applied inside each tab header's border, with x
// controlling the left/right padding and y the top/bottom padding. It layers on
// top of the active styles (theme-derived or from WithStyles) and is reflected
// in mouse hit-testing, so header clicks stay aligned.
func WithPadding(x, y int) TabModelOption {
	return func(tm *TabModel) {
		tm.padX, tm.padY = x, y
		tm.hasPadding = true
	}
}

// WithMargin sets the margin applied outside each tab header's border, with x
// controlling the left/right margin (the gap between tabs) and y the top/bottom
// margin. It layers on top of the active styles (theme-derived or from
// WithStyles) and is reflected in mouse hit-testing, so header clicks stay
// aligned.
func WithMargin(x, y int) TabModelOption {
	return func(tm *TabModel) {
		tm.marginX, tm.marginY = x, y
		tm.hasMargin = true
	}
}

// New builds a TabModel from the given options and returns a pointer to it.
// Defaults applied before options are processed:
//   - theme: theme.Catppuccin() (dark Mocha / light Latte)
//   - keymap: DefaultKeyMap() (Tab/Shift+Tab/1-9)
//   - mouseMode: tea.MouseModeCellMotion (enables header click-to-activate)
//
// After all options are applied:
//  1. Styles are derived from the active theme palette, unless WithStyles
//     supplied an explicit set, in which case those are used as-is.
//  2. Any WithPadding/WithMargin spacing is layered onto the header styles and
//     reflected in mouse hit-testing.
//  3. The current index is clamped to [0, len(tabs)-1].
//  4. The tab at current is unconditionally set to model.Active; all others
//     keep whatever state they were given.
func New(options ...TabModelOption) *TabModel {
	tabModel := &TabModel{
		theme:     thm.Catppuccin(),
		keymap:    DefaultKeyMap(),
		mouseMode: tea.MouseModeCellMotion,
	}
	for _, o := range options {
		o(tabModel)
	}

	if !tabModel.hasCustomStyles {
		tabModel.styles = tabModel.theme.Styles()
	}
	tabModel.applyHeaderSpacing()
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

// applyHeaderSpacing layers the WithPadding/WithMargin spacing onto every
// header style. Because View computes mouse hit-test spans from the rendered
// (and therefore already-spaced) header strings, no separate offset bookkeeping
// is needed for clicks to remain aligned. lipgloss takes spacing as (vertical,
// horizontal), so y maps to top/bottom and x to left/right.
func (t *TabModel) applyHeaderSpacing() {
	if !t.hasPadding && !t.hasMargin {
		return
	}
	space := func(s lipgloss.Style) lipgloss.Style {
		if t.hasPadding {
			s = s.Padding(t.padY, t.padX)
		}
		if t.hasMargin {
			s = s.Margin(t.marginY, t.marginX)
		}
		return s
	}
	t.styles.ActiveHeader = space(t.styles.ActiveHeader)
	t.styles.InactiveHeader = space(t.styles.InactiveHeader)
	t.styles.DisabledHeader = space(t.styles.DisabledHeader)
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

// Init initializes only the current tab's body and returns its Init command (if
// any). All other tabs remain uninitialized and are built lazily on first
// activation via activate/buildAndPrime. Init satisfies the tea.Model interface.
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

// Update handles navigation messages (Tab, Shift+Tab, 1-9 jump, mouse header
// click, WindowSizeMsg) and forwards all other messages to the active tab's
// body, persisting the returned model so child state survives tab switches.
// Update satisfies the tea.Model interface and always returns a *TabModel.
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

// View renders the horizontal header row followed by the body of the active tab
// (when HasPermission is true). The returned tea.View has its OnMouse handler
// set to a hit-tester that emits a tab-activation event for left clicks on any
// header, and MouseMode set per the configured mouseMode (default:
// tea.MouseModeCellMotion). View satisfies the tea.Model interface.
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
	view.MouseMode = t.mouseMode
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
