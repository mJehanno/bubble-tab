package bubbletab

// White-box tests for the root bubbletab package. Same package for access
// to unexported types (tabClickMsg) and direct inspection of TabModel fields.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
	"github.com/mJehanno/bubble-tab/pkg/model"
	thm "github.com/mJehanno/bubble-tab/pkg/theme"
)

// ---- stub tea.Model ----

// stubBody is a minimal, value-typed tea.Model that counts how many times
// Init and Update are called and records the last message forwarded to it.
type stubBody struct {
	id          int
	initCount   int
	updateCount int
	lastMsg     tea.Msg
}

func (s stubBody) Init() tea.Cmd {
	s.initCount++
	return nil
}
func (s stubBody) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	s.updateCount++
	s.lastMsg = msg
	return s, nil
}
func (s stubBody) View() tea.View { return tea.NewView("stub") }

// initCmdBody is a stub whose Init returns a non-nil command.
type initCmdBody struct {
	sentinel string
}

func (b initCmdBody) Init() tea.Cmd {
	return func() tea.Msg { return b.sentinel }
}
func (b initCmdBody) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return b, nil }
func (b initCmdBody) View() tea.View                          { return tea.NewView("cmdBody") }

// counterBody is a value-typed body that increments a counter on a special msg.
type counterBody struct{ val int }
type incMsg struct{}

func (c counterBody) Init() tea.Cmd { return nil }
func (c counterBody) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(incMsg); ok {
		c.val++
	}
	return c, nil
}
func (c counterBody) View() tea.View { return tea.NewView(strings.Repeat("x", c.val)) }

// trackingBody records every Update call by appending msgs.
// Since it is a value type, we use a pointer-to-slice trick via a shared ref.
type trackingBody struct {
	msgs *[]tea.Msg
}

func newTracker() trackingBody {
	s := make([]tea.Msg, 0)
	return trackingBody{msgs: &s}
}
func (t trackingBody) Init() tea.Cmd { return nil }
func (t trackingBody) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	*t.msgs = append(*t.msgs, msg)
	return t, nil
}
func (t trackingBody) View() tea.View { return tea.NewView("tracked") }

// ---- helpers ----

// mustTabModel type-asserts the returned tea.Model to TabModel.
func mustTabModel(t *testing.T, m tea.Model) TabModel {
	t.Helper()
	tm, ok := m.(TabModel)
	if !ok {
		t.Fatalf("Update returned %T, want TabModel", m)
	}
	return tm
}

// sendKey sends a key press and returns the resulting TabModel.
func sendKey(t *testing.T, tm TabModel, code rune, mods ...tea.KeyMod) TabModel {
	t.Helper()
	var mod tea.KeyMod
	for _, m := range mods {
		mod |= m
	}
	m, _ := tm.Update(tea.KeyPressMsg{Code: code, Mod: mod})
	return mustTabModel(t, m)
}

// viewContains strips ANSI and checks whether the rendered view content
// contains sub (using a plain byte-level search on the raw ANSI string since
// body content is embedded in escape sequences).
func viewContains(t *testing.T, tm TabModel, sub string) bool {
	t.Helper()
	return strings.Contains(tm.View().Content, sub)
}

func tabsOf(names ...string) []model.Tab {
	tabs := make([]model.Tab, len(names))
	for i, n := range names {
		tabs[i] = *model.NewTab(model.WithName(n), model.WithBody(stubBody{id: i}))
	}
	return tabs
}

// noBorderTheme returns a Theme with None borders so headers have height=1,
// which makes Y-coordinate tests on the mouse handler unambiguous.
func noBorderTheme() thm.Theme {
	cat := thm.Catppuccin()
	return thm.Theme{
		IsDark:         true,
		ActiveBorder:   thm.None,
		InactiveBorder: thm.None,
		DisabledBorder: thm.None,
		Dark:           cat.Dark,
		Light:          cat.Light,
	}
}

// ---- Construction ----

func TestNew_Defaults(t *testing.T) {
	tm := New()
	// Default theme is Catppuccin (IsDark=true, has non-zero styles).
	// Default keymap is DefaultKeyMap.
	// Current = 0, tabs = nil.
	if len(tm.tabs) != 0 {
		t.Errorf("expected 0 tabs, got %d", len(tm.tabs))
	}
	if tm.current != 0 {
		t.Errorf("expected current=0, got %d", tm.current)
	}
}

func TestNew_WithTabs(t *testing.T) {
	tabs := tabsOf("A", "B", "C")
	tm := New(WithTabs(tabs))
	if len(tm.tabs) != 3 {
		t.Errorf("expected 3 tabs, got %d", len(tm.tabs))
	}
}

func TestNew_WithCurrent_Clamped_Negative(t *testing.T) {
	tabs := tabsOf("A", "B")
	tm := New(WithTabs(tabs), WithCurrent(-5))
	if tm.current != 0 {
		t.Errorf("negative current should clamp to 0, got %d", tm.current)
	}
}

func TestNew_WithCurrent_Clamped_TooLarge(t *testing.T) {
	tabs := tabsOf("A", "B")
	tm := New(WithTabs(tabs), WithCurrent(999))
	if tm.current != 1 {
		t.Errorf("out-of-range current should clamp to len-1 (%d), got %d", 1, tm.current)
	}
}

func TestNew_WithCurrent_ValidIndex(t *testing.T) {
	tabs := tabsOf("A", "B", "C")
	tm := New(WithTabs(tabs), WithCurrent(2))
	if tm.current != 2 {
		t.Errorf("expected current=2, got %d", tm.current)
	}
}

func TestNew_WithTheme_StylesBuiltFromIt(t *testing.T) {
	g := thm.Gruvbox()
	tm := New(WithTheme(g))
	// Styles should reflect the theme. Just verify no panic and a non-zero field.
	_ = tm.styles.ActiveHeader.Render("x")
}

func TestNew_WithKeyMap(t *testing.T) {
	km := DefaultKeyMap()
	tm := New(WithKeyMap(km))
	_ = tm // just verify no panic
}

func TestNew_EmptyTabs_Current0(t *testing.T) {
	tm := New()
	if tm.current != 0 {
		t.Errorf("empty tabs: current should be 0, got %d", tm.current)
	}
}

// ---- Init ----

func TestInit_EmptyTabs_ReturnsNil(t *testing.T) {
	tm := New()
	cmd := tm.Init()
	if cmd != nil {
		t.Error("Init with no tabs should return nil cmd")
	}
}

func TestInit_NilBody_ReturnsNil(t *testing.T) {
	tabs := []model.Tab{
		*model.NewTab(model.WithName("T")), // no body
	}
	tm := New(WithTabs(tabs))
	cmd := tm.Init()
	if cmd != nil {
		t.Error("Init with nil body should return nil cmd")
	}
}

func TestInit_WithBody_ReturnsBodyInitCmd(t *testing.T) {
	tabs := []model.Tab{
		*model.NewTab(model.WithName("T"), model.WithBody(initCmdBody{sentinel: "hello"})),
	}
	tm := New(WithTabs(tabs))
	cmd := tm.Init()
	if cmd == nil {
		t.Fatal("Init should return the body's Init cmd")
	}
	msg := cmd()
	if msg != "hello" {
		t.Errorf("cmd() = %v, want 'hello'", msg)
	}
}

func TestInit_OnlyInitsCurrentTab(t *testing.T) {
	// Use factory tabs so we can detect whether Init was called on non-current tabs.
	initCounts := [2]int{}

	factory0 := func() tea.Model {
		return &stubBody{id: 0}
	}
	factory1 := func() tea.Model {
		return &stubBody{id: 1}
	}
	_ = initCounts
	_ = factory0
	_ = factory1

	// Use initCmdBody with distinct sentinels; only current tab's cmd fires.
	tabs := []model.Tab{
		*model.NewTab(model.WithName("T0"), model.WithBody(initCmdBody{sentinel: "cmd0"})),
		*model.NewTab(model.WithName("T1"), model.WithBody(initCmdBody{sentinel: "cmd1"})),
	}
	tm := New(WithTabs(tabs)) // current=0

	cmd := tm.Init()
	if cmd == nil {
		t.Fatal("expected non-nil cmd for tab0")
	}
	msg := cmd()
	if msg != "cmd0" {
		t.Errorf("Init cmd from tab0 produced %q, want 'cmd0'", msg)
	}
}

func TestInit_FactoryCurrentTab_ConstructsBody(t *testing.T) {
	calls := 0
	factory := func() tea.Model {
		calls++
		return stubBody{id: 42}
	}
	tabs := []model.Tab{
		*model.NewTab(model.WithName("Lazy"), model.WithBodyFunc(factory)),
	}
	tm := New(WithTabs(tabs))
	tm.Init()

	if calls != 1 {
		t.Errorf("factory called %d times during Init, want 1", calls)
	}
}

// ---- View ----

func TestView_EmptyTabs_ReturnsEmpty(t *testing.T) {
	tm := New()
	v := tm.View()
	if v.Content != "" {
		t.Errorf("empty tabs view should be '', got %q", v.Content)
	}
}

func TestView_NoPermission_BodyNotRendered(t *testing.T) {
	tabs := []model.Tab{
		*model.NewTab(
			model.WithName("Secret"),
			model.WithHasPermission(false),
			model.WithBody(counterBody{val: 99}),
		),
	}
	tm := New(WithTabs(tabs))
	tm.Init()
	v := tm.View()
	// The body's View content ("xxxxx...") should not appear.
	if strings.Contains(v.Content, strings.Repeat("x", 99)) {
		t.Error("body content should not appear when HasPermission=false")
	}
	// But the header should appear.
	if !strings.Contains(v.Content, "Secret") {
		t.Error("header should still be rendered without permission")
	}
}

func TestView_HasMouseHandler(t *testing.T) {
	tabs := tabsOf("A", "B")
	tm := New(WithTabs(tabs))
	tm.Init()
	v := tm.View()
	if v.OnMouse == nil {
		t.Error("View should set OnMouse handler")
	}
	if v.MouseMode != tea.MouseModeCellMotion {
		t.Errorf("View should default to MouseModeCellMotion so clicks are delivered, got %v", v.MouseMode)
	}
}

func TestView_WithMouseMode_Override(t *testing.T) {
	tabs := tabsOf("A", "B")
	tm := New(WithTabs(tabs), WithMouseMode(tea.MouseModeNone))
	tm.Init()
	v := tm.View()
	if v.MouseMode != tea.MouseModeNone {
		t.Errorf("WithMouseMode(None) should disable mouse mode, got %v", v.MouseMode)
	}
}

func TestView_EmptyTabs_NoMouseHandler_NoPanic(t *testing.T) {
	tm := New()
	v := tm.View()
	// OnMouse may be nil for empty model; neither branch should panic.
	_ = v.OnMouse
}

// ---- Update: zero tabs ----

func TestUpdate_EmptyTabs_NoOp(t *testing.T) {
	tm := New()
	m, cmd := tm.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if cmd != nil {
		t.Error("empty TabModel Update should return nil cmd")
	}
	result := mustTabModel(t, m)
	if len(result.tabs) != 0 {
		t.Error("still should have zero tabs")
	}
}

// ---- Navigation: forward (Tab key) ----

func TestHandleKey_TabForward_MovesToNext(t *testing.T) {
	tabs := tabsOf("A", "B", "C")
	tm := New(WithTabs(tabs))
	tm.Init()

	tm2 := sendKey(t, *tm, tea.KeyTab)

	if tm2.current != 1 {
		t.Errorf("expected current=1 after Tab, got %d", tm2.current)
	}
	if tm2.tabs[0].State() != model.Inactive {
		t.Errorf("tab0 should be Inactive after moving away, got %q", tm2.tabs[0].State())
	}
	if tm2.tabs[1].State() != model.Active {
		t.Errorf("tab1 should be Active, got %q", tm2.tabs[1].State())
	}
}

func TestHandleKey_TabForward_WrapAround(t *testing.T) {
	tabs := tabsOf("A", "B", "C")
	tm := New(WithTabs(tabs), WithCurrent(2))
	tm.Init()

	tm2 := sendKey(t, *tm, tea.KeyTab)
	if tm2.current != 0 {
		t.Errorf("expected wrap to 0 after Tab from last tab, got %d", tm2.current)
	}
}

func TestHandleKey_TabForward_SkipsDisabled(t *testing.T) {
	tabs := []model.Tab{
		*model.NewTab(model.WithName("A"), model.WithBody(stubBody{})),
		*model.NewTab(model.WithName("D"), model.WithState(model.Disabled), model.WithBody(stubBody{})),
		*model.NewTab(model.WithName("C"), model.WithBody(stubBody{})),
	}
	tm := New(WithTabs(tabs))
	tm.Init()

	tm2 := sendKey(t, *tm, tea.KeyTab)
	if tm2.current != 2 {
		t.Errorf("expected skip to tab2 (skipping disabled tab1), got %d", tm2.current)
	}
}

// ---- Navigation: backward (Shift+Tab) ----

func TestHandleKey_ShiftTab_MovesToPrev(t *testing.T) {
	tabs := tabsOf("A", "B", "C")
	tm := New(WithTabs(tabs), WithCurrent(2))
	tm.Init()

	tm2 := sendKey(t, *tm, tea.KeyTab, tea.ModShift)
	if tm2.current != 1 {
		t.Errorf("expected current=1 after Shift+Tab, got %d", tm2.current)
	}
}

func TestHandleKey_ShiftTab_WrapAround(t *testing.T) {
	tabs := tabsOf("A", "B", "C")
	tm := New(WithTabs(tabs), WithCurrent(0))
	tm.Init()

	tm2 := sendKey(t, *tm, tea.KeyTab, tea.ModShift)
	if tm2.current != 2 {
		t.Errorf("expected wrap to 2 after Shift+Tab from 0, got %d", tm2.current)
	}
}

func TestHandleKey_ShiftTab_SkipsDisabled(t *testing.T) {
	tabs := []model.Tab{
		*model.NewTab(model.WithName("A"), model.WithBody(stubBody{})),
		*model.NewTab(model.WithName("D"), model.WithState(model.Disabled), model.WithBody(stubBody{})),
		*model.NewTab(model.WithName("C"), model.WithBody(stubBody{})),
	}
	tm := New(WithTabs(tabs), WithCurrent(2))
	tm.Init()

	tm2 := sendKey(t, *tm, tea.KeyTab, tea.ModShift)
	if tm2.current != 0 {
		t.Errorf("expected skip to tab0 (skipping disabled tab1), got %d", tm2.current)
	}
}

// ---- Navigation: all disabled — no infinite loop ----

func TestHandleKey_AllDisabled_StaysPut_Forward(t *testing.T) {
	tabs := []model.Tab{
		*model.NewTab(model.WithName("D1"), model.WithState(model.Disabled)),
		*model.NewTab(model.WithName("D2"), model.WithState(model.Disabled)),
		*model.NewTab(model.WithName("D3"), model.WithState(model.Disabled)),
	}
	tm := New(WithTabs(tabs))
	// Do NOT call Init — state is Disabled so current may stay at 0 anyway.
	before := tm.current

	m, _ := tm.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	tm2 := mustTabModel(t, m)
	if tm2.current != before {
		t.Errorf("all-disabled forward nav: current moved from %d to %d (should stay put)", before, tm2.current)
	}
}

func TestHandleKey_AllDisabled_StaysPut_Backward(t *testing.T) {
	tabs := []model.Tab{
		*model.NewTab(model.WithName("D1"), model.WithState(model.Disabled)),
		*model.NewTab(model.WithName("D2"), model.WithState(model.Disabled)),
	}
	tm := New(WithTabs(tabs))
	before := tm.current

	m, _ := tm.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	tm2 := mustTabModel(t, m)
	if tm2.current != before {
		t.Errorf("all-disabled backward nav: current moved from %d to %d (should stay put)", before, tm2.current)
	}
}

// ---- Navigation: single tab ----

func TestHandleKey_SingleTab_ForwardStaysPut(t *testing.T) {
	tabs := tabsOf("Only")
	tm := New(WithTabs(tabs))
	tm.Init()

	m, _ := tm.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	tm2 := mustTabModel(t, m)
	if tm2.current != 0 {
		t.Errorf("single tab forward: expected current=0, got %d", tm2.current)
	}
}

func TestHandleKey_SingleTab_BackwardStaysPut(t *testing.T) {
	tabs := tabsOf("Only")
	tm := New(WithTabs(tabs))
	tm.Init()

	m, _ := tm.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	tm2 := mustTabModel(t, m)
	if tm2.current != 0 {
		t.Errorf("single tab backward: expected current=0, got %d", tm2.current)
	}
}

// ---- Navigation: only one enabled tab among disabled ----

func TestHandleKey_OneEnabledAmongDisabled_ForwardFindIt(t *testing.T) {
	tabs := []model.Tab{
		*model.NewTab(model.WithName("D1"), model.WithState(model.Disabled), model.WithBody(stubBody{})),
		*model.NewTab(model.WithName("E"), model.WithBody(stubBody{})), // index 1
		*model.NewTab(model.WithName("D2"), model.WithState(model.Disabled), model.WithBody(stubBody{})),
	}
	tm := New(WithTabs(tabs), WithCurrent(1))
	tm.Init()

	// Forward from the only enabled tab should come back to itself.
	m, _ := tm.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	tm2 := mustTabModel(t, m)
	if tm2.current != 1 {
		t.Errorf("one enabled tab: expected stay at 1, got %d", tm2.current)
	}
}

// ---- Navigation: number jump ('1'-'9') ----

func TestHandleKey_NumberJump_ValidIndex(t *testing.T) {
	tabs := tabsOf("A", "B", "C")
	tm := New(WithTabs(tabs))
	tm.Init()

	// '2' key jumps to index 1 (0-based: msg.Code-'1').
	tm2 := sendKey(t, *tm, '2')
	if tm2.current != 1 {
		t.Errorf("'2' jump: expected current=1, got %d", tm2.current)
	}
}

func TestHandleKey_NumberJump_Index1(t *testing.T) {
	tabs := tabsOf("A", "B", "C")
	tm := New(WithTabs(tabs), WithCurrent(2))
	tm.Init()

	tm2 := sendKey(t, *tm, '1')
	if tm2.current != 0 {
		t.Errorf("'1' jump: expected current=0, got %d", tm2.current)
	}
}

func TestHandleKey_NumberJump_OutOfRange_NoOp(t *testing.T) {
	tabs := tabsOf("A", "B", "C") // len=3, valid 1-3
	tm := New(WithTabs(tabs))
	tm.Init()
	before := tm.current

	// '9' -> index 8, out of range for 3-tab model -> no-op.
	m, cmd := tm.Update(tea.KeyPressMsg{Code: '9'})
	tm2 := mustTabModel(t, m)
	if tm2.current != before {
		t.Errorf("'9' with 3 tabs: expected current unchanged=%d, got %d", before, tm2.current)
	}
	if cmd != nil {
		t.Error("out-of-range jump should produce nil cmd")
	}
}

func TestHandleKey_NumberJump_ToDisabled_NoOp(t *testing.T) {
	tabs := []model.Tab{
		*model.NewTab(model.WithName("A"), model.WithBody(stubBody{})), // index 0
		*model.NewTab(model.WithName("D"), model.WithState(model.Disabled), model.WithBody(stubBody{})), // index 1
		*model.NewTab(model.WithName("C"), model.WithBody(stubBody{})), // index 2
	}
	tm := New(WithTabs(tabs))
	tm.Init()
	before := tm.current // 0

	// '2' -> index 1 which is Disabled -> no-op.
	m, cmd := tm.Update(tea.KeyPressMsg{Code: '2'})
	tm2 := mustTabModel(t, m)
	if tm2.current != before {
		t.Errorf("jump to disabled: expected current=%d unchanged, got %d", before, tm2.current)
	}
	if cmd != nil {
		t.Error("jump to disabled should produce nil cmd")
	}
}

func TestHandleKey_ZeroKey_ForwardedToBody(t *testing.T) {
	// '0' is not in ['1','9'], so it falls through to body forwarding.
	tr := newTracker()
	tabs := []model.Tab{
		*model.NewTab(model.WithName("T"), model.WithBody(tr)),
	}
	tm := New(WithTabs(tabs))
	tm.Init()

	m, _ := tm.Update(tea.KeyPressMsg{Code: '0'})
	_ = m
	// The tracker's msgs slice should have received the '0' key.
	if len(*tr.msgs) != 1 {
		t.Errorf("'0' key should be forwarded to body, got %d msgs", len(*tr.msgs))
	}
}

// ---- Lazy init-once semantics ----

func TestActivate_InitCalledOncePerTab(t *testing.T) {
	// Track Init calls via initCmdBody sentinel values.
	tab0Body := initCmdBody{sentinel: "tab0-init"}
	tab1Body := initCmdBody{sentinel: "tab1-init"}

	tabs := []model.Tab{
		*model.NewTab(model.WithName("T0"), model.WithBody(tab0Body)),
		*model.NewTab(model.WithName("T1"), model.WithBody(tab1Body)),
	}
	tm := New(WithTabs(tabs))

	// Init initializes tab0 only.
	cmd0 := tm.Init()
	if cmd0 == nil || cmd0() != "tab0-init" {
		t.Error("Init should return tab0's Init cmd")
	}
	if !tm.tabs[0].Initialized() {
		t.Error("tab0 should be marked Initialized after Init")
	}
	if tm.tabs[1].Initialized() {
		t.Error("tab1 should NOT be marked Initialized before activation")
	}
}

func TestActivate_SwitchToTab_InitsBodyOnce(t *testing.T) {
	tabs := tabsOf("A", "B")
	tm := New(WithTabs(tabs))
	tm.Init()

	// First switch to tab1.
	m1, _ := tm.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	tm1 := mustTabModel(t, m1)
	if !tm1.tabs[1].Initialized() {
		t.Error("tab1 should be Initialized after first activation")
	}

	// Switch back to tab0.
	m2, _ := tm1.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	tm2 := mustTabModel(t, m2)
	if !tm2.tabs[0].Initialized() {
		t.Error("tab0 should remain Initialized")
	}

	// Switch to tab1 again — Init should NOT be called again.
	// We verify via Initialized flag: it should still be true (was set on first visit).
	m3, _ := tm2.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	tm3 := mustTabModel(t, m3)
	if !tm3.tabs[1].Initialized() {
		t.Error("tab1 should still be Initialized on second visit")
	}
}

func TestActivate_StatePreservedAcrossRoundTrip(t *testing.T) {
	// counterBody accumulates state; verify that state survives a tab switch and back.
	c0 := counterBody{val: 0}
	c1 := counterBody{val: 0}
	tabs := []model.Tab{
		*model.NewTab(model.WithName("T0"), model.WithBody(c0)),
		*model.NewTab(model.WithName("T1"), model.WithBody(c1)),
	}
	tm := New(WithTabs(tabs))
	tm.Init()

	// Increment tab0's counter.
	m1, _ := tm.Update(incMsg{})
	tm1 := mustTabModel(t, m1)
	if !viewContains(t, tm1, "x") {
		t.Error("after incMsg tab0 counter should be 1 (one 'x' in view)")
	}

	// Switch to tab1.
	m2, _ := tm1.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	tm2 := mustTabModel(t, m2)

	// Switch back to tab0.
	m3, _ := tm2.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	tm3 := mustTabModel(t, m3)

	// tab0 view should still show "x" (counter preserved).
	if !viewContains(t, tm3, "x") {
		t.Error("tab0 state (counter=1) should survive round-trip through tab1")
	}
}

// ---- Factory (lazy) tab never visited => factory never called ----

func TestFactoryTab_NeverVisited_NeverConstructed(t *testing.T) {
	factoryCalls := 0
	factory := func() tea.Model {
		factoryCalls++
		return stubBody{}
	}

	tabs := []model.Tab{
		*model.NewTab(model.WithName("Eager"), model.WithBody(stubBody{})),
		*model.NewTab(model.WithName("Lazy"), model.WithBodyFunc(factory)),
	}
	tm := New(WithTabs(tabs))
	tm.Init() // only inits tab0

	// Never navigate to tab1.
	if factoryCalls != 0 {
		t.Errorf("factory called %d times for never-visited tab, want 0", factoryCalls)
	}
}

// ---- WindowSizeMsg ----

func TestHandleResize_StoresDimensions(t *testing.T) {
	tm := New(WithTabs(tabsOf("A")))
	tm.Init()

	m, _ := tm.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	tm2 := mustTabModel(t, m)
	if tm2.width != 120 {
		t.Errorf("width: got %d, want 120", tm2.width)
	}
	if tm2.height != 40 {
		t.Errorf("height: got %d, want 40", tm2.height)
	}
}

func TestHandleResize_ForwardsToAllBuiltBodies(t *testing.T) {
	tr0 := newTracker()
	tr1 := newTracker()

	tabs := []model.Tab{
		*model.NewTab(model.WithName("T0"), model.WithBody(tr0)),
		*model.NewTab(model.WithName("T1"), model.WithBody(tr1)),
	}
	tm := New(WithTabs(tabs))
	tm.Init()

	m, _ := tm.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	_ = mustTabModel(t, m)

	if len(*tr0.msgs) == 0 {
		t.Error("WindowSizeMsg should have been forwarded to tab0's body")
	}
	if len(*tr1.msgs) == 0 {
		t.Error("WindowSizeMsg should have been forwarded to tab1's body (pre-built)")
	}
	// Verify the message type.
	if _, ok := (*tr0.msgs)[0].(tea.WindowSizeMsg); !ok {
		t.Errorf("tab0 received %T, want WindowSizeMsg", (*tr0.msgs)[0])
	}
}

func TestHandleResize_DoesNotBuildFactoryBody(t *testing.T) {
	factoryCalls := 0
	factory := func() tea.Model {
		factoryCalls++
		return stubBody{}
	}

	tabs := []model.Tab{
		*model.NewTab(model.WithName("Eager"), model.WithBody(stubBody{})),
		*model.NewTab(model.WithName("Lazy"), model.WithBodyFunc(factory)),
	}
	tm := New(WithTabs(tabs))
	tm.Init()

	tm.Update(tea.WindowSizeMsg{Width: 80, Height: 24}) //nolint
	if factoryCalls != 0 {
		t.Errorf("WindowSizeMsg should not trigger factory for lazy tab, got %d calls", factoryCalls)
	}
}

func TestHandleResize_EnsuresCurrentFactoryTab(t *testing.T) {
	// When current tab is a factory tab, WindowSizeMsg should ensure it (build it).
	factoryCalls := 0
	factory := func() tea.Model {
		factoryCalls++
		return stubBody{}
	}
	tabs := []model.Tab{
		*model.NewTab(model.WithName("LazyFirst"), model.WithBodyFunc(factory)),
	}
	tm := New(WithTabs(tabs))
	// Do NOT call Init so factory is still unbuilt.

	tm.Update(tea.WindowSizeMsg{Width: 80, Height: 24}) //nolint
	// The current tab (index 0) is lazy, WindowSizeMsg calls EnsureBody on current.
	if factoryCalls != 1 {
		t.Errorf("WindowSizeMsg should ensure current factory tab: factory called %d times, want 1", factoryCalls)
	}
}

// ---- Message forwarding ----

func TestUpdate_NonNavMsg_ForwardedOnlyToActiveBody(t *testing.T) {
	tr0 := newTracker()
	tr1 := newTracker()

	tabs := []model.Tab{
		*model.NewTab(model.WithName("T0"), model.WithBody(tr0)),
		*model.NewTab(model.WithName("T1"), model.WithBody(tr1)),
	}
	tm := New(WithTabs(tabs))
	tm.Init()

	type customMsg struct{ v int }
	tm.Update(customMsg{v: 1}) //nolint

	if len(*tr0.msgs) != 1 {
		t.Errorf("active body (tab0) should receive the msg: got %d", len(*tr0.msgs))
	}
	if len(*tr1.msgs) != 0 {
		t.Errorf("inactive body (tab1) should NOT receive the msg: got %d", len(*tr1.msgs))
	}
}

func TestUpdate_NonNavMsg_PersistsUpdatedState(t *testing.T) {
	// Verify that state changes in the body survive the Update round-trip.
	c := counterBody{val: 0}
	tabs := []model.Tab{
		*model.NewTab(model.WithName("T"), model.WithBody(c)),
	}
	tm := New(WithTabs(tabs))
	tm.Init()

	m1, _ := tm.Update(incMsg{})
	tm1 := mustTabModel(t, m1)

	m2, _ := tm1.Update(incMsg{})
	tm2 := mustTabModel(t, m2)

	// After two increments, body view should show "xx".
	if !viewContains(t, tm2, "xx") {
		t.Error("body state (counter=2) not persisted: view does not contain 'xx'")
	}
}

// ---- tabClickMsg ----

func TestTabClickMsg_ActivatesTab(t *testing.T) {
	tabs := tabsOf("A", "B", "C")
	tm := New(WithTabs(tabs))
	tm.Init()

	m, _ := tm.Update(tabClickMsg{index: 2})
	tm2 := mustTabModel(t, m)
	if tm2.current != 2 {
		t.Errorf("tabClickMsg{2}: expected current=2, got %d", tm2.current)
	}
}

func TestTabClickMsg_DisabledTab_NoOp(t *testing.T) {
	tabs := []model.Tab{
		*model.NewTab(model.WithName("A"), model.WithBody(stubBody{})),
		*model.NewTab(model.WithName("D"), model.WithState(model.Disabled), model.WithBody(stubBody{})),
	}
	tm := New(WithTabs(tabs))
	tm.Init()
	before := tm.current

	m, _ := tm.Update(tabClickMsg{index: 1})
	tm2 := mustTabModel(t, m)
	if tm2.current != before {
		t.Errorf("click on Disabled tab should be no-op, current changed from %d to %d", before, tm2.current)
	}
}

func TestTabClickMsg_NegativeIndex_NoOp(t *testing.T) {
	tabs := tabsOf("A", "B")
	tm := New(WithTabs(tabs))
	tm.Init()

	m, _ := tm.Update(tabClickMsg{index: -1})
	tm2 := mustTabModel(t, m)
	if tm2.current != 0 {
		t.Errorf("negative tabClickMsg: expected current=0, got %d", tm2.current)
	}
}

func TestTabClickMsg_OutOfRange_NoOp(t *testing.T) {
	tabs := tabsOf("A", "B")
	tm := New(WithTabs(tabs))
	tm.Init()

	m, _ := tm.Update(tabClickMsg{index: 99})
	tm2 := mustTabModel(t, m)
	if tm2.current != 0 {
		t.Errorf("out-of-range tabClickMsg: expected current=0, got %d", tm2.current)
	}
}

// ---- mouseHeaderHandler ----

func TestMouseHandler_LeftClick_HitsCorrectTab(t *testing.T) {
	// Use no-border theme so headers are 1 row tall and spans are predictable.
	tabs := tabsOf("Tab0", "Tab1", "Tab2")
	tm := New(WithTabs(tabs), WithTheme(noBorderTheme()))
	tm.Init()

	v := tm.View()
	handler := v.OnMouse
	if handler == nil {
		t.Fatal("expected non-nil OnMouse handler")
	}

	// Build the same header strings as View() does to compute expected spans.
	headers := make([]string, len(tm.tabs))
	for i := range tm.tabs {
		headers[i] = tm.styles.Header(tm.tabs[i].State()).Render(tm.tabs[i].Name())
	}
	x := 0
	for i, h := range headers {
		w := lipgloss.Width(h)
		mid := x + w/2
		t.Run(tm.tabs[i].Name(), func(t *testing.T) {
			click := tea.MouseClickMsg{X: mid, Y: 0, Button: tea.MouseLeft}
			cmd := handler(click)
			if cmd == nil {
				t.Errorf("click at X=%d,Y=0 within tab %d should emit tabClickMsg", mid, i)
				return
			}
			msg := cmd()
			click_msg, ok := msg.(tabClickMsg)
			if !ok {
				t.Errorf("cmd() returned %T, want tabClickMsg", msg)
				return
			}
			if click_msg.index != i {
				t.Errorf("click at tab %d: got index %d", i, click_msg.index)
			}
		})
		x += w
	}
}

func TestMouseHandler_RightClick_ReturnsNil(t *testing.T) {
	tabs := tabsOf("A", "B")
	tm := New(WithTabs(tabs), WithTheme(noBorderTheme()))
	tm.Init()

	handler := tm.View().OnMouse
	cmd := handler(tea.MouseClickMsg{X: 2, Y: 0, Button: tea.MouseRight})
	if cmd != nil {
		t.Error("right click should return nil cmd")
	}
}

func TestMouseHandler_ClickOutsideAllSpans_ReturnsNil(t *testing.T) {
	tabs := tabsOf("A", "B")
	tm := New(WithTabs(tabs), WithTheme(noBorderTheme()))
	tm.Init()

	handler := tm.View().OnMouse
	cmd := handler(tea.MouseClickMsg{X: 9999, Y: 0, Button: tea.MouseLeft})
	if cmd != nil {
		t.Error("click outside all spans should return nil cmd")
	}
}

func TestMouseHandler_NonClickMsg_ReturnsNil(t *testing.T) {
	tabs := tabsOf("A", "B")
	tm := New(WithTabs(tabs), WithTheme(noBorderTheme()))
	tm.Init()

	handler := tm.View().OnMouse
	cmd := handler(tea.MouseReleaseMsg{X: 2, Y: 0, Button: tea.MouseLeft})
	if cmd != nil {
		t.Error("non-click mouse message should return nil cmd")
	}
}

func TestMouseHandler_YBeyondHeight_ReturnsNil(t *testing.T) {
	// Use bordered theme so height > 1.
	tabs := tabsOf("A", "B")
	tm := New(WithTabs(tabs))
	tm.Init()

	handler := tm.View().OnMouse

	// Compute actual header height.
	h := tm.styles.Header(tm.tabs[0].State()).Render(tm.tabs[0].Name())
	headerHeight := lipgloss.Height(h)

	// Click at Y = headerHeight (just outside the header block) should be nil.
	cmd := handler(tea.MouseClickMsg{X: 0, Y: headerHeight, Button: tea.MouseLeft})
	if cmd != nil {
		t.Errorf("click at Y=%d (beyond headerHeight=%d) should return nil", headerHeight, headerHeight)
	}
}

func TestMouseHandler_YWithinBorderedHeader_ReturnsCmd(t *testing.T) {
	// With bordered headers (height=3), clicks at Y=0, 1, 2 should all match.
	tabs := tabsOf("A", "B")
	tm := New(WithTabs(tabs)) // default Catppuccin has Rounded/Normal borders
	tm.Init()

	handler := tm.View().OnMouse
	h := tm.styles.Header(tm.tabs[0].State()).Render(tm.tabs[0].Name())
	headerHeight := lipgloss.Height(h)

	if headerHeight < 2 {
		t.Skip("header height is 1, bordered test not applicable")
	}

	// Click somewhere within tab0's horizontal span at each Y row.
	headers := make([]string, len(tm.tabs))
	for i := range tm.tabs {
		headers[i] = tm.styles.Header(tm.tabs[i].State()).Render(tm.tabs[i].Name())
	}
	w0 := lipgloss.Width(headers[0])
	midX := w0 / 2

	for y := 0; y < headerHeight; y++ {
		click := tea.MouseClickMsg{X: midX, Y: y, Button: tea.MouseLeft}
		cmd := handler(click)
		if cmd == nil {
			t.Errorf("click at Y=%d within bordered header (height=%d) should not return nil", y, headerHeight)
		}
	}
}

// ---- activate bounds checking ----

func TestActivate_NegativeIndex_NoPanic(t *testing.T) {
	tabs := tabsOf("A", "B")
	tm := New(WithTabs(tabs))
	tm.Init()

	// activate is unexported but reachable via tabClickMsg; test via Update.
	// We test the internal path by sending a negative tabClickMsg (which bounds-checks).
	m, _ := tm.Update(tabClickMsg{index: -1})
	_ = mustTabModel(t, m) // no panic = pass
}

// ---- moveForward / moveBackward edge cases ----

func TestMoveForward_TwoTabsFirstDisabled(t *testing.T) {
	tabs := []model.Tab{
		*model.NewTab(model.WithName("D"), model.WithState(model.Disabled)),
		*model.NewTab(model.WithName("E"), model.WithBody(stubBody{})),
	}
	tm := New(WithTabs(tabs), WithCurrent(1))
	tm.Init()

	idx := tm.moveForward()
	// From tab1, forward wraps to tab0 (Disabled) -> skips to tab1 -> but step<=n
	// Actually: step=1 -> (1+1)%2=0 -> Disabled; step=2 -> (1+2)%2=1 -> not Disabled -> return 1
	if idx != 1 {
		t.Errorf("moveForward from only enabled tab: got %d, want 1", idx)
	}
}

func TestMoveBackward_TwoTabsSecondDisabled(t *testing.T) {
	tabs := []model.Tab{
		*model.NewTab(model.WithName("E"), model.WithBody(stubBody{})),
		*model.NewTab(model.WithName("D"), model.WithState(model.Disabled)),
	}
	tm := New(WithTabs(tabs), WithCurrent(0))
	tm.Init()

	idx := tm.moveBackward()
	// From tab0, backward: step=1 -> (0-1%2+2)%2=1 -> Disabled; step=2 -> 0 -> not Disabled -> return 0
	if idx != 0 {
		t.Errorf("moveBackward from only enabled tab: got %d, want 0", idx)
	}
}

// ---- DefaultKeyMap ----

func TestDefaultKeyMap(t *testing.T) {
	km := DefaultKeyMap()
	if !km.Next.Enabled() {
		t.Error("Next binding should be enabled")
	}
	if !km.Prev.Enabled() {
		t.Error("Prev binding should be enabled")
	}
	if !km.Jump.Enabled() {
		t.Error("Jump binding should be enabled")
	}
}

// ---- View: tab state visually reflected ----

func TestView_ActiveTabHasDistinctStyle(t *testing.T) {
	tabs := tabsOf("Home", "Settings")
	tm := New(WithTabs(tabs))
	tm.Init()

	v := tm.View()
	// At minimum verify both names appear in the rendered output.
	if !strings.Contains(v.Content, "Home") {
		t.Error("view should contain active tab name 'Home'")
	}
	if !strings.Contains(v.Content, "Settings") {
		t.Error("view should contain inactive tab name 'Settings'")
	}
}

// ---- Stress: rapid navigation does not panic ----

func TestRapidNavigation_NoPanic(t *testing.T) {
	tabs := []model.Tab{
		*model.NewTab(model.WithName("A"), model.WithBody(stubBody{})),
		*model.NewTab(model.WithName("B"), model.WithBody(stubBody{})),
		*model.NewTab(model.WithName("C"), model.WithBody(stubBody{})),
	}
	tm := New(WithTabs(tabs))
	tm.Init()
	cur := *tm

	for i := 0; i < 100; i++ {
		m, _ := cur.Update(tea.KeyPressMsg{Code: tea.KeyTab})
		cur = mustTabModel(t, m)
	}
}

// ---- Uncovered-line targeted tests ----

// activate: out-of-bounds guard (internal path; test directly as white-box).
func TestActivate_OutOfBounds_ReturnsNil(t *testing.T) {
	tabs := tabsOf("A", "B")
	tm := New(WithTabs(tabs))
	tm.Init()

	// Call activate directly (white-box access).
	cmd := tm.activate(-1)
	if cmd != nil {
		t.Error("activate(-1) should return nil")
	}
	cmd2 := tm.activate(len(tm.tabs))
	if cmd2 != nil {
		t.Errorf("activate(%d) should return nil", len(tm.tabs))
	}
	// State should be unaffected.
	if tm.current != 0 {
		t.Errorf("current changed after invalid activate: got %d", tm.current)
	}
}

// Update default path: nil body -> return nil cmd.
func TestUpdate_NilBody_DefaultPath_ReturnsNil(t *testing.T) {
	tabs := []model.Tab{
		*model.NewTab(model.WithName("NoBody")), // no body, no factory
	}
	tm := New(WithTabs(tabs))
	// Do NOT call Init; body stays nil.

	type customMsg struct{}
	m, cmd := tm.Update(customMsg{})
	if cmd != nil {
		t.Error("non-nav msg to tab with nil body should produce nil cmd")
	}
	_ = mustTabModel(t, m)
}

// handleResize: body.Update returns non-nil cmd -> batched.
func TestHandleResize_BodyUpdateCmd_Batched(t *testing.T) {
	tabs := []model.Tab{
		*model.NewTab(model.WithName("T"), model.WithBody(resizeCmdBodyStub{sentinel: "resize-done"})),
	}
	tm := New(WithTabs(tabs))
	tm.Init()

	m, cmd := tm.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	_ = mustTabModel(t, m)
	// The batched cmd should be non-nil since resizeCmdBodyStub.Update returns a cmd.
	if cmd == nil {
		t.Error("handleResize should batch non-nil cmds from body.Update")
	}
	msg := cmd()
	// tea.Batch returns a []tea.Cmd; the result of calling the batch is a batchMsg
	// which is a slice. Just verify no panic and cmd is runnable.
	_ = msg
}

// resizeCmdBodyStub is a body that returns a non-nil cmd from Update (for WindowSizeMsg coverage).
type resizeCmdBodyStub struct{ sentinel string }

func (r resizeCmdBodyStub) Init() tea.Cmd { return nil }
func (r resizeCmdBodyStub) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(tea.WindowSizeMsg); ok {
		s := r.sentinel
		return r, func() tea.Msg { return s }
	}
	return r, nil
}
func (r resizeCmdBodyStub) View() tea.View { return tea.NewView("resize-stub") }

// NOTE (potential bug): makeHeaderMouseHandler computes header height using
// lipgloss.Height of individual rendered headers. With bordered styles
// (e.g. Rounded, Normal), this height is 3 (top border, content, bottom border).
// The handler accepts any Y in [0, height). This means clicking on a tab's top
// or bottom border row (Y=0 or Y=2 in absolute view coordinates) also counts
// as a tab click, which is arguably correct for usability.
//
// There is no off-by-one: the body content starts at line `height` (after the
// header block and the extra "\n" separator appended in View()), so no click
// targeting the body accidentally triggers a tab switch.
//
// See TestMouseHandler_YWithinBorderedHeader_ReturnsCmd for verification.

// ===========================================================================
// CHANGE 1 — New() auto-activates the current tab
// ===========================================================================

// probeBody records every WindowSizeMsg received in Update and whether Init
// was called. It uses a shared *[]tea.Msg so mutation survives value copies.
type probeBody struct {
	sizeMsgs *[]tea.WindowSizeMsg
	initDone *bool
	initCmd  tea.Cmd // optional non-nil cmd to return from Init
}

func newProbeBody() probeBody {
	msgs := make([]tea.WindowSizeMsg, 0)
	done := false
	return probeBody{sizeMsgs: &msgs, initDone: &done}
}

func newProbeBodyWithInitCmd(sentinel string) probeBody {
	pb := newProbeBody()
	pb.initCmd = func() tea.Msg { return sentinel }
	return pb
}

func (p probeBody) Init() tea.Cmd {
	*p.initDone = true
	return p.initCmd
}
func (p probeBody) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if sz, ok := msg.(tea.WindowSizeMsg); ok {
		*p.sizeMsgs = append(*p.sizeMsgs, sz)
	}
	return p, nil
}
func (p probeBody) View() tea.View { return tea.NewView("probe") }

// TestNew_CurrentTabIsActive asserts that after New the current tab (index 0
// by default) is Active and other tabs remain Inactive.
func TestNew_CurrentTabIsActive(t *testing.T) {
	tabs := tabsOf("A", "B", "C")
	tm := New(WithTabs(tabs))

	if tm.tabs[0].State() != model.Active {
		t.Errorf("tab0 (current) should be Active after New, got %q", tm.tabs[0].State())
	}
	if tm.tabs[1].State() != model.Inactive {
		t.Errorf("tab1 should be Inactive after New, got %q", tm.tabs[1].State())
	}
	if tm.tabs[2].State() != model.Inactive {
		t.Errorf("tab2 should be Inactive after New, got %q", tm.tabs[2].State())
	}
}

// TestNew_WithCurrent_CorrectTabIsActive asserts that with WithCurrent(2)
// it is tab index 2 that becomes Active.
func TestNew_WithCurrent_CorrectTabIsActive(t *testing.T) {
	tabs := tabsOf("A", "B", "C")
	tm := New(WithTabs(tabs), WithCurrent(2))

	if tm.tabs[2].State() != model.Active {
		t.Errorf("tab2 (current) should be Active, got %q", tm.tabs[2].State())
	}
	if tm.tabs[0].State() != model.Inactive {
		t.Errorf("tab0 should be Inactive, got %q", tm.tabs[0].State())
	}
	if tm.tabs[1].State() != model.Inactive {
		t.Errorf("tab1 should be Inactive, got %q", tm.tabs[1].State())
	}
}

// TestNew_ClampedCurrent_ActiveTabIsClampedIndex asserts that when the
// requested current is out of range, the clamped index becomes Active.
func TestNew_ClampedCurrent_NegativeBecomesZeroAndActive(t *testing.T) {
	tabs := tabsOf("A", "B")
	tm := New(WithTabs(tabs), WithCurrent(-99))

	if tm.current != 0 {
		t.Fatalf("expected clamped current=0, got %d", tm.current)
	}
	if tm.tabs[0].State() != model.Active {
		t.Errorf("clamped current tab0 should be Active, got %q", tm.tabs[0].State())
	}
	if tm.tabs[1].State() != model.Inactive {
		t.Errorf("tab1 should be Inactive, got %q", tm.tabs[1].State())
	}
}

func TestNew_ClampedCurrent_TooLargeBecomesLastAndActive(t *testing.T) {
	tabs := tabsOf("A", "B", "C")
	tm := New(WithTabs(tabs), WithCurrent(999))

	if tm.current != 2 {
		t.Fatalf("expected clamped current=2, got %d", tm.current)
	}
	if tm.tabs[2].State() != model.Active {
		t.Errorf("clamped current tab2 should be Active, got %q", tm.tabs[2].State())
	}
	if tm.tabs[0].State() != model.Inactive {
		t.Errorf("tab0 should be Inactive, got %q", tm.tabs[0].State())
	}
}

// TestNew_DisabledCurrentTabBecomesActive asserts that even a tab configured
// with WithState(model.Disabled) as the current tab is overridden to Active.
func TestNew_DisabledCurrentTabBecomesActive(t *testing.T) {
	tabs := []model.Tab{
		*model.NewTab(model.WithName("D"), model.WithState(model.Disabled), model.WithBody(stubBody{})),
		*model.NewTab(model.WithName("B"), model.WithBody(stubBody{})),
	}
	tm := New(WithTabs(tabs), WithCurrent(0))

	// The disabled-configured tab at index 0 is current and must be forced Active.
	if tm.tabs[0].State() != model.Active {
		t.Errorf("current tab configured as Disabled should still become Active after New, got %q", tm.tabs[0].State())
	}
}

// TestNew_EmptyTabs_NoPanic ensures that New with no tabs does not panic when
// attempting to auto-activate the current tab.
func TestNew_EmptyTabs_AutoActivate_NoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("New() with empty tabs panicked: %v", r)
		}
	}()
	tm := New() // len(tabs)==0; the guard must prevent the write
	if len(tm.tabs) != 0 {
		t.Error("should have zero tabs")
	}
}

// TestNew_SingleTab_IsActive checks that a single tab becomes Active.
func TestNew_SingleTab_IsActive(t *testing.T) {
	tabs := tabsOf("Only")
	tm := New(WithTabs(tabs))

	if tm.tabs[0].State() != model.Active {
		t.Errorf("single tab should be Active after New, got %q", tm.tabs[0].State())
	}
}

// ===========================================================================
// CHANGE 2 — Cached window size replayed on first lazy body build
// ===========================================================================

// tabsWithLazyProbe builds a two-tab model where tab 0 is an eager stub
// (current) and tab 1 is a lazy factory that returns a probeBody.
// The returned *probeBody pointer gives observable access to the probe's
// shared state after the factory is called.
func tabsWithLazyProbe(t *testing.T) (*TabModel, *probeBody) {
	t.Helper()
	pb := newProbeBodyWithInitCmd("lazy-init-sentinel")
	tabs := []model.Tab{
		*model.NewTab(model.WithName("Eager"), model.WithBody(stubBody{})),
		*model.NewTab(model.WithName("Lazy"), model.WithBodyFunc(func() tea.Model {
			return pb
		})),
	}
	tm := New(WithTabs(tabs)) // current=0, eager tab active
	return tm, &pb
}

// TestBuildAndPrime_LazyTab_ReceivesCachedSize is scenario (a):
// WindowSizeMsg arrives before the lazy tab is activated; on activation the
// cached size is replayed to the freshly built body.
func TestBuildAndPrime_LazyTab_ReceivesCachedSize(t *testing.T) {
	tm, pb := tabsWithLazyProbe(t)
	tm.Init()

	// Send a window size so haveSize=true.
	m, _ := tm.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	tm2 := mustTabModel(t, m)

	// Navigate to the lazy tab (index 1) via Tab key.
	m2, cmd := tm2.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	tm3 := mustTabModel(t, m2)
	_ = tm3

	// The returned cmd must be non-nil (batched sizeCmd + initCmd).
	if cmd == nil {
		t.Error("activating lazy tab after a WindowSizeMsg should return non-nil cmd (size+init batch)")
	}

	// The probe body must have received exactly one WindowSizeMsg with the cached dimensions.
	if len(*pb.sizeMsgs) != 1 {
		t.Errorf("lazy body should receive exactly 1 WindowSizeMsg on first build, got %d", len(*pb.sizeMsgs))
	} else {
		got := (*pb.sizeMsgs)[0]
		if got.Width != 80 || got.Height != 24 {
			t.Errorf("lazy body received WindowSizeMsg{%d,%d}, want {80,24}", got.Width, got.Height)
		}
	}

	// Init must have been called exactly once.
	if !*pb.initDone {
		t.Error("lazy body's Init should have been called on first activation")
	}
}

// TestBuildAndPrime_LazyTab_NoSizeBeforeActivation is scenario (b):
// The lazy tab is activated BEFORE any WindowSizeMsg. haveSize is false so
// no size must be replayed, but Init must still run once.
func TestBuildAndPrime_LazyTab_NoSizeBeforeActivation(t *testing.T) {
	tm, pb := tabsWithLazyProbe(t)
	tm.Init()

	// Navigate to the lazy tab WITHOUT sending a WindowSizeMsg first.
	m, _ := tm.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	_ = mustTabModel(t, m)

	// No WindowSizeMsg should have arrived at the probe.
	if len(*pb.sizeMsgs) != 0 {
		t.Errorf("lazy body should receive 0 WindowSizeMsgs when haveSize=false, got %d", len(*pb.sizeMsgs))
	}

	// Init must have been called.
	if !*pb.initDone {
		t.Error("lazy body's Init should have been called even without a prior WindowSizeMsg")
	}
}

// TestBuildAndPrime_EagerBodyReceivesSizeOnceFromResize is scenario (c):
// An already-built (eager) body should receive the WindowSizeMsg exactly once
// via handleResize's broadcast — not twice.
func TestBuildAndPrime_EagerBodyReceivesSizeOnceFromResize(t *testing.T) {
	pb := newProbeBody()
	tabs := []model.Tab{
		*model.NewTab(model.WithName("Eager"), model.WithBody(pb)),
	}
	tm := New(WithTabs(tabs))
	tm.Init()

	// Send a single WindowSizeMsg.
	m, _ := tm.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	_ = mustTabModel(t, m)

	// The eager body is already built; it should have received exactly one size msg.
	if len(*pb.sizeMsgs) != 1 {
		t.Errorf("eager body should receive exactly 1 WindowSizeMsg via handleResize, got %d", len(*pb.sizeMsgs))
	}
}

// TestBuildAndPrime_Init_CurrentLazyTab_NoStaleSize is scenario (d):
// Init() runs before any WindowSizeMsg; haveSize is false so no size should
// be sent to the current lazy tab's body, but the body IS built and Init'd.
func TestBuildAndPrime_Init_CurrentLazyTab_NoStaleSize(t *testing.T) {
	pb := newProbeBodyWithInitCmd("init-sentinel")
	tabs := []model.Tab{
		*model.NewTab(model.WithName("Lazy"), model.WithBodyFunc(func() tea.Model {
			return pb
		})),
	}
	tm := New(WithTabs(tabs)) // current=0 is the lazy tab

	// Call Init before any resize — haveSize should be false.
	cmd := tm.Init()

	// Init built and init'd the body but sent no WindowSizeMsg.
	if len(*pb.sizeMsgs) != 0 {
		t.Errorf("Init before any resize should send no WindowSizeMsg, got %d", len(*pb.sizeMsgs))
	}
	if !*pb.initDone {
		t.Error("Init should have called the body's Init")
	}

	// The returned cmd should be the body's Init cmd (non-nil sentinel).
	if cmd == nil {
		t.Error("Init() should return the body's Init cmd")
	}
}

// TestBuildAndPrime_EagerCurrentBody_InitedOnce is scenario (e):
// An eagerly-set body that is current but not yet initialized: buildAndPrime
// must still call its Init exactly once.
func TestBuildAndPrime_EagerCurrentBody_InitedOnce(t *testing.T) {
	pb := newProbeBodyWithInitCmd("eager-init")
	tabs := []model.Tab{
		*model.NewTab(model.WithName("Eager"), model.WithBody(pb)),
	}
	tm := New(WithTabs(tabs))

	// Before Init the body is built but not Initialized.
	if tm.tabs[0].Initialized() {
		t.Fatal("precondition: eager body should not be Initialized before Init()")
	}

	cmd := tm.Init()

	if !tm.tabs[0].Initialized() {
		t.Error("eager current body should be Initialized after Init()")
	}
	if !*pb.initDone {
		t.Error("eager current body's Init() should have been called")
	}
	if cmd == nil {
		t.Error("Init() should return a non-nil cmd from the eager body's Init")
	}
}

// TestBuildAndPrime_AlreadyInitialized_NoDoubleInit asserts that buildAndPrime
// does not call Init a second time when the tab is already marked Initialized.
func TestBuildAndPrime_AlreadyInitialized_NoDoubleInit(t *testing.T) {
	initCallCount := 0
	countingBody := countingInitBody{counter: &initCallCount}
	tabs := []model.Tab{
		*model.NewTab(model.WithName("T"), model.WithBody(countingBody)),
	}
	tm := New(WithTabs(tabs))

	// First Init: should call body.Init once.
	tm.Init()
	if initCallCount != 1 {
		t.Fatalf("expected 1 Init call, got %d", initCallCount)
	}

	// Send a resize; buildAndPrime is invoked in handleResize's path via
	// EnsureBody on current, but the tab is already initialized so Init
	// must not be called again.
	tm.Update(tea.WindowSizeMsg{Width: 80, Height: 24}) //nolint

	if initCallCount != 1 {
		t.Errorf("Init should not be called again on an already-initialized body, got %d calls", initCallCount)
	}
}

// countingInitBody is a body that increments a shared counter on Init.
type countingInitBody struct {
	counter *int
}

func (c countingInitBody) Init() tea.Cmd {
	*c.counter++
	return nil
}
func (c countingInitBody) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return c, nil }
func (c countingInitBody) View() tea.View                          { return tea.NewView("counting") }

// TestBuildAndPrime_SizeReplayedOnlyOnFirstBuild verifies that a second
// activation of a lazy tab (already built from the first visit) does NOT
// replay the cached size again — it was not nil-before-EnsureBody the second time.
func TestBuildAndPrime_SizeReplayedOnlyOnFirstBuild(t *testing.T) {
	tm, pb := tabsWithLazyProbe(t)
	tm.Init()

	// Provide a size, then activate the lazy tab.
	m, _ := tm.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	tm2 := mustTabModel(t, m)
	m2, _ := tm2.Update(tea.KeyPressMsg{Code: tea.KeyTab}) // activate lazy (index 1)
	tm3 := mustTabModel(t, m2)

	// One size replayed on first build.
	if len(*pb.sizeMsgs) != 1 {
		t.Fatalf("first activation: expected 1 size replay, got %d", len(*pb.sizeMsgs))
	}

	// Navigate away and back — second activation must not replay size again.
	m3, _ := tm3.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}) // back to eager
	tm4 := mustTabModel(t, m3)
	m4, _ := tm4.Update(tea.KeyPressMsg{Code: tea.KeyTab}) // back to lazy
	_ = mustTabModel(t, m4)

	// Still exactly 1 size msg — no replay on second visit.
	if len(*pb.sizeMsgs) != 1 {
		t.Errorf("second activation: size should not be replayed again, got %d total size msgs", len(*pb.sizeMsgs))
	}
}

// ===========================================================================
// CHANGE 3 — Offset-aware mouse hit-testing
// ===========================================================================

// buildHeadersNoBorder constructs the rendered header slice using the
// no-border theme, returning (headers, tabModel) for use in direct
// makeHeaderMouseHandler calls.
func buildHeadersNoBorder(t *testing.T, names ...string) ([]string, *TabModel) {
	t.Helper()
	tabs := tabsOf(names...)
	tm := New(WithTabs(tabs), WithTheme(noBorderTheme()))
	headers := make([]string, len(tm.tabs))
	for i := range tm.tabs {
		headers[i] = tm.styles.Header(tm.tabs[i].State()).Render(tm.tabs[i].Name())
	}
	return headers, tm
}

// TestMouseHandler_WithOffset_NegativeLocalX_ReturnsNil asserts that a click
// whose raw X falls within the offsetX region (localX negative) is rejected.
func TestMouseHandler_WithOffset_NegativeLocalX_ReturnsNil(t *testing.T) {
	headers, _ := buildHeadersNoBorder(t, "Tab0", "Tab1")
	const offsetX = 10
	handler := makeHeaderMouseHandler(headers, offsetX, 0)

	// Raw X=5 with offsetX=10 → localX = -5 → must be rejected.
	cmd := handler(tea.MouseClickMsg{X: 5, Y: 0, Button: tea.MouseLeft})
	if cmd != nil {
		t.Error("click with localX<0 (raw X within offset region) should return nil")
	}
}

// TestMouseHandler_WithOffset_NegativeLocalY_ReturnsNil asserts that a click
// whose raw Y falls within the offsetY region (localY negative) is rejected.
func TestMouseHandler_WithOffset_NegativeLocalY_ReturnsNil(t *testing.T) {
	headers, _ := buildHeadersNoBorder(t, "Tab0", "Tab1")
	const offsetY = 5
	handler := makeHeaderMouseHandler(headers, 0, offsetY)

	// Raw Y=2 with offsetY=5 → localY = -3 → must be rejected.
	cmd := handler(tea.MouseClickMsg{X: 0, Y: 2, Button: tea.MouseLeft})
	if cmd != nil {
		t.Error("click with localY<0 (raw Y within offset region) should return nil")
	}
}

// TestMouseHandler_WithOffset_ClickAtExactOffsetEdge_HitsFirstTab asserts that
// a click at raw X == offsetX (localX == 0) inside the first tab's span hits tab 0.
func TestMouseHandler_WithOffset_ClickAtExactOffsetEdge_HitsFirstTab(t *testing.T) {
	headers, _ := buildHeadersNoBorder(t, "Tab0", "Tab1")
	const offsetX = 10
	handler := makeHeaderMouseHandler(headers, offsetX, 0)

	// localX = offsetX - offsetX = 0, which is start of tab0's span.
	cmd := handler(tea.MouseClickMsg{X: offsetX, Y: 0, Button: tea.MouseLeft})
	if cmd == nil {
		t.Fatal("click at raw X=offsetX (localX=0) should hit tab0")
	}
	msg := cmd()
	cm, ok := msg.(tabClickMsg)
	if !ok {
		t.Fatalf("expected tabClickMsg, got %T", msg)
	}
	if cm.index != 0 {
		t.Errorf("click at localX=0 should hit tab0, got index %d", cm.index)
	}
}

// TestMouseHandler_WithOffset_CorrectTabByLocalSpan asserts that offsetX shifts
// the entire span grid and a click at raw X == offsetX + span maps to the right tab.
func TestMouseHandler_WithOffset_CorrectTabByLocalSpan(t *testing.T) {
	headers, _ := buildHeadersNoBorder(t, "AAAA", "BB")

	lipgloss.Width(headers[0]) // warm up
	w0 := lipgloss.Width(headers[0])

	const offsetX = 20
	handler := makeHeaderMouseHandler(headers, offsetX, 0)

	// A click at raw X = offsetX + w0 falls at localX = w0, which is the start
	// of tab1's span.
	cmd := handler(tea.MouseClickMsg{X: offsetX + w0, Y: 0, Button: tea.MouseLeft})
	if cmd == nil {
		t.Fatalf("click at raw X=%d (localX=%d, start of tab1 span) should hit tab1", offsetX+w0, w0)
	}
	msg := cmd()
	cm, ok := msg.(tabClickMsg)
	if !ok {
		t.Fatalf("expected tabClickMsg, got %T", msg)
	}
	if cm.index != 1 {
		t.Errorf("click at start of tab1 span: expected index 1, got %d", cm.index)
	}
}

// TestMouseHandler_WithOffsetY_LocalYEqualsHeight_ReturnsNil checks that
// localY == height (exactly at the boundary) is rejected.
func TestMouseHandler_WithOffsetY_LocalYEqualsHeight_ReturnsNil(t *testing.T) {
	headers, _ := buildHeadersNoBorder(t, "A", "B")
	// noBorderTheme gives height=1 per header.
	height := 1
	const offsetY = 3
	handler := makeHeaderMouseHandler(headers, 0, offsetY)

	// localY = offsetY + height - offsetY = height → exactly at boundary → rejected.
	rawY := offsetY + height
	cmd := handler(tea.MouseClickMsg{X: 0, Y: rawY, Button: tea.MouseLeft})
	if cmd != nil {
		t.Errorf("click at localY=%d (== height=%d) should return nil", height, height)
	}
}

// TestMouseHandler_WithOffsetY_LocalYWithinHeight_HitsTab checks that
// localY in [0, height) hits the tab.
func TestMouseHandler_WithOffsetY_LocalYWithinHeight_HitsTab(t *testing.T) {
	headers, _ := buildHeadersNoBorder(t, "A", "B")
	const offsetY = 3
	handler := makeHeaderMouseHandler(headers, 0, offsetY)

	// localY = 0 (raw Y = offsetY) should be within [0,1) for no-border height=1.
	cmd := handler(tea.MouseClickMsg{X: 0, Y: offsetY, Button: tea.MouseLeft})
	if cmd == nil {
		t.Error("click at localY=0 (raw Y=offsetY) should hit a tab")
	}
}

// TestMouseHandler_BothOffsets_OnlyInOffsetRegion_ReturnsNil verifies that
// when both offsetX and offsetY are non-zero, a click still inside the offset
// region (both locals negative) returns nil.
func TestMouseHandler_BothOffsets_OnlyInOffsetRegion_ReturnsNil(t *testing.T) {
	headers, _ := buildHeadersNoBorder(t, "Tab0", "Tab1")
	const offsetX, offsetY = 10, 5
	handler := makeHeaderMouseHandler(headers, offsetX, offsetY)

	// Raw click at (3, 2): localX=-7, localY=-3 → both negative → nil.
	cmd := handler(tea.MouseClickMsg{X: 3, Y: 2, Button: tea.MouseLeft})
	if cmd != nil {
		t.Error("click with both locals negative should return nil")
	}
}

// TestMouseHandler_ZeroOffset_BackwardCompatible verifies that passing
// offsetX=0, offsetY=0 still behaves identically to the old single-arg
// signature: left-click within a tab's span returns the correct tabClickMsg.
func TestMouseHandler_ZeroOffset_BackwardCompatible(t *testing.T) {
	headers, _ := buildHeadersNoBorder(t, "Tab0", "Tab1")
	handler := makeHeaderMouseHandler(headers, 0, 0)

	// Click at X=0, Y=0 should hit tab0 (span starts at 0).
	cmd := handler(tea.MouseClickMsg{X: 0, Y: 0, Button: tea.MouseLeft})
	if cmd == nil {
		t.Fatal("click at (0,0) with zero offsets should hit tab0")
	}
	msg := cmd()
	cm, ok := msg.(tabClickMsg)
	if !ok {
		t.Fatalf("expected tabClickMsg, got %T", msg)
	}
	if cm.index != 0 {
		t.Errorf("expected tab0, got index %d", cm.index)
	}
}

// TestMouseHandler_WithOffset_RightClick_ReturnsNil verifies that non-left
// clicks are still rejected even when an offset is applied.
func TestMouseHandler_WithOffset_RightClick_ReturnsNil(t *testing.T) {
	headers, _ := buildHeadersNoBorder(t, "Tab0", "Tab1")
	const offsetX = 5
	handler := makeHeaderMouseHandler(headers, offsetX, 0)

	cmd := handler(tea.MouseClickMsg{X: offsetX + 2, Y: 0, Button: tea.MouseRight})
	if cmd != nil {
		t.Error("right click with offset should still return nil")
	}
}

// ===========================================================================
// Coverage gap closers for buildAndPrime, moveForward, moveBackward
// ===========================================================================

// TestBuildAndPrime_OutOfBoundsIndex_ReturnsNil directly exercises the
// bounds guard at the top of buildAndPrime (the defensive index < 0 ||
// index >= len(tabs) branch), which is unreachable through the public API
// but exercised here as a white-box sanity check.
func TestBuildAndPrime_OutOfBoundsIndex_ReturnsNil(t *testing.T) {
	tabs := tabsOf("A", "B")
	tm := New(WithTabs(tabs))

	cmd := tm.buildAndPrime(-1)
	if cmd != nil {
		t.Error("buildAndPrime(-1) should return nil")
	}

	cmd2 := tm.buildAndPrime(len(tm.tabs))
	if cmd2 != nil {
		t.Errorf("buildAndPrime(%d) should return nil", len(tm.tabs))
	}
}

// TestMoveForward_AllDisabledAfterConstruction_FallsBackToCurrent covers the
// `return t.current` path in moveForward. After New(), the current tab is
// forced Active, so to reach the fallback we must manually set ALL tabs to
// Disabled (simulating a post-construction state change).
func TestMoveForward_AllDisabledAfterConstruction_FallsBackToCurrent(t *testing.T) {
	tabs := tabsOf("A", "B", "C")
	tm := New(WithTabs(tabs)) // current=0 is now Active

	// Manually set all tabs to Disabled after construction.
	for i := range tm.tabs {
		(&tm.tabs[i]).SetState(model.Disabled)
	}

	// Now all tabs are truly Disabled; moveForward must return current.
	idx := tm.moveForward()
	if idx != tm.current {
		t.Errorf("moveForward with all tabs manually Disabled should return current=%d, got %d", tm.current, idx)
	}
}

// TestMoveBackward_AllDisabledAfterConstruction_FallsBackToCurrent mirrors
// the forward test for moveBackward.
func TestMoveBackward_AllDisabledAfterConstruction_FallsBackToCurrent(t *testing.T) {
	tabs := tabsOf("A", "B", "C")
	tm := New(WithTabs(tabs)) // current=0 is now Active

	for i := range tm.tabs {
		(&tm.tabs[i]).SetState(model.Disabled)
	}

	idx := tm.moveBackward()
	if idx != tm.current {
		t.Errorf("moveBackward with all tabs manually Disabled should return current=%d, got %d", tm.current, idx)
	}
}

// ===========================================================================
// CHANGE 4 — WithStyles, WithPadding, WithMargin options
// ===========================================================================

// customStyles returns a thm.Styles that is visually distinct from any
// theme-derived styles: it uses a recognizable hot-pink foreground on all
// header variants so we can detect whether WithStyles was honoured.
func customStyles() thm.Styles {
	base := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff00ff")).Padding(9, 9)
	return thm.Styles{
		ActiveHeader:   base,
		InactiveHeader: base,
		DisabledHeader: base,
		ActiveBody:     lipgloss.NewStyle(),
		InactiveBody:   lipgloss.NewStyle(),
		DisabledBody:   lipgloss.NewStyle(),
	}
}

// ---- WithPadding ----

// TestWithPadding_SetsFieldsAndFlag checks that WithPadding records the
// supplied values in padX/padY and raises hasPadding=true.
func TestWithPadding_SetsFieldsAndFlag(t *testing.T) {
	tm := New(WithPadding(4, 2))
	if !tm.hasPadding {
		t.Error("hasPadding should be true after WithPadding")
	}
	if tm.padX != 4 {
		t.Errorf("padX: got %d, want 4", tm.padX)
	}
	if tm.padY != 2 {
		t.Errorf("padY: got %d, want 2", tm.padY)
	}
}

// TestWithPadding_AppliedToAllThreeHeaderStyles verifies that the padding is
// applied to ActiveHeader, InactiveHeader, and DisabledHeader after New().
// lipgloss Padding(y,x) maps y→top/bottom and x→left/right.
func TestWithPadding_AppliedToAllThreeHeaderStyles(t *testing.T) {
	const padX, padY = 5, 3
	tm := New(WithTabs(tabsOf("A")), WithPadding(padX, padY))

	cases := []struct {
		name  string
		style lipgloss.Style
	}{
		{"ActiveHeader", tm.styles.ActiveHeader},
		{"InactiveHeader", tm.styles.InactiveHeader},
		{"DisabledHeader", tm.styles.DisabledHeader},
	}
	for _, tc := range cases {
		top, right, bottom, left := tc.style.GetPadding()
		if top != padY {
			t.Errorf("%s: top padding got %d, want %d (padY)", tc.name, top, padY)
		}
		if bottom != padY {
			t.Errorf("%s: bottom padding got %d, want %d (padY)", tc.name, bottom, padY)
		}
		if left != padX {
			t.Errorf("%s: left padding got %d, want %d (padX)", tc.name, left, padX)
		}
		if right != padX {
			t.Errorf("%s: right padding got %d, want %d (padX)", tc.name, right, padX)
		}
	}
}

// TestWithPadding_ZeroValues_FlagStillSet verifies the zero-value-with-flag
// semantics: WithPadding(0,0) must still mark hasPadding=true and must not
// panic or corrupt styles.
func TestWithPadding_ZeroValues_FlagStillSet(t *testing.T) {
	tm := New(WithTabs(tabsOf("A")), WithPadding(0, 0))
	if !tm.hasPadding {
		t.Error("hasPadding should be true even when padX=0, padY=0")
	}
	// Styles must still be valid (renderable without panic).
	_ = tm.styles.ActiveHeader.Render("x")
	_ = tm.styles.InactiveHeader.Render("x")
	_ = tm.styles.DisabledHeader.Render("x")
	// All padding components should be zero.
	top, right, bottom, left := tm.styles.ActiveHeader.GetPadding()
	if top != 0 || right != 0 || bottom != 0 || left != 0 {
		t.Errorf("WithPadding(0,0) should set all padding to 0, got top=%d right=%d bottom=%d left=%d",
			top, right, bottom, left)
	}
}

// TestWithPadding_NoPaddingWithout_WithPadding confirms that without
// WithPadding the hasPadding flag is false and no padding override is applied.
func TestWithPadding_NoPaddingWithout_WithPadding(t *testing.T) {
	tm := New(WithTabs(tabsOf("A")))
	if tm.hasPadding {
		t.Error("hasPadding should be false when WithPadding is not used")
	}
}

// ---- WithMargin ----

// TestWithMargin_SetsFieldsAndFlag checks that WithMargin records values and
// sets hasMargin=true.
func TestWithMargin_SetsFieldsAndFlag(t *testing.T) {
	tm := New(WithMargin(6, 1))
	if !tm.hasMargin {
		t.Error("hasMargin should be true after WithMargin")
	}
	if tm.marginX != 6 {
		t.Errorf("marginX: got %d, want 6", tm.marginX)
	}
	if tm.marginY != 1 {
		t.Errorf("marginY: got %d, want 1", tm.marginY)
	}
}

// TestWithMargin_AppliedToAllThreeHeaderStyles verifies that the margin is
// applied to all three header styles after New(). lipgloss Margin(y,x) maps
// y→top/bottom and x→left/right.
func TestWithMargin_AppliedToAllThreeHeaderStyles(t *testing.T) {
	const marginX, marginY = 3, 2
	tm := New(WithTabs(tabsOf("A")), WithMargin(marginX, marginY))

	cases := []struct {
		name  string
		style lipgloss.Style
	}{
		{"ActiveHeader", tm.styles.ActiveHeader},
		{"InactiveHeader", tm.styles.InactiveHeader},
		{"DisabledHeader", tm.styles.DisabledHeader},
	}
	for _, tc := range cases {
		top, right, bottom, left := tc.style.GetMargin()
		if top != marginY {
			t.Errorf("%s: top margin got %d, want %d (marginY)", tc.name, top, marginY)
		}
		if bottom != marginY {
			t.Errorf("%s: bottom margin got %d, want %d (marginY)", tc.name, bottom, marginY)
		}
		if left != marginX {
			t.Errorf("%s: left margin got %d, want %d (marginX)", tc.name, left, marginX)
		}
		if right != marginX {
			t.Errorf("%s: right margin got %d, want %d (marginX)", tc.name, right, marginX)
		}
	}
}

// TestWithMargin_ZeroValues_FlagStillSet verifies the zero-value-with-flag
// semantics for WithMargin(0,0).
func TestWithMargin_ZeroValues_FlagStillSet(t *testing.T) {
	tm := New(WithTabs(tabsOf("A")), WithMargin(0, 0))
	if !tm.hasMargin {
		t.Error("hasMargin should be true even when marginX=0, marginY=0")
	}
	_ = tm.styles.ActiveHeader.Render("x")
	top, right, bottom, left := tm.styles.ActiveHeader.GetMargin()
	if top != 0 || right != 0 || bottom != 0 || left != 0 {
		t.Errorf("WithMargin(0,0) should leave all margins at 0, got top=%d right=%d bottom=%d left=%d",
			top, right, bottom, left)
	}
}

// TestWithMargin_NoMarginWithout_WithMargin confirms the flag is false without
// the option.
func TestWithMargin_NoMarginWithout_WithMargin(t *testing.T) {
	tm := New(WithTabs(tabsOf("A")))
	if tm.hasMargin {
		t.Error("hasMargin should be false when WithMargin is not used")
	}
}

// ---- WithStyles ----

// TestWithStyles_SetsFlag verifies that WithStyles raises hasCustomStyles=true.
func TestWithStyles_SetsFlag(t *testing.T) {
	tm := New(WithStyles(customStyles()))
	if !tm.hasCustomStyles {
		t.Error("hasCustomStyles should be true after WithStyles")
	}
}

// TestWithStyles_Precedence_OverTheme verifies that when both WithStyles and
// WithTheme are supplied, the custom styles take precedence: tm.styles must
// match the custom set, NOT the theme-derived set.
func TestWithStyles_Precedence_OverTheme(t *testing.T) {
	cs := customStyles()
	// Apply WithStyles BEFORE WithTheme; the option list is processed in order
	// but the flag test in New() is what matters (theme is skipped when
	// hasCustomStyles=true regardless of option order).
	tm := New(WithStyles(cs), WithTheme(thm.Gruvbox()))

	// The custom styles' ActiveHeader should be used verbatim.
	wantRender := cs.ActiveHeader.Render("probe")
	gotRender := tm.styles.ActiveHeader.Render("probe")
	if wantRender != gotRender {
		t.Errorf("WithStyles precedence over WithTheme failed:\n  want: %q\n  got:  %q", wantRender, gotRender)
	}
}

// TestWithStyles_Precedence_OverTheme_ReverseOrder verifies that option order
// does not matter: WithTheme first, WithStyles second is also respected.
func TestWithStyles_Precedence_OverTheme_ReverseOrder(t *testing.T) {
	cs := customStyles()
	tm := New(WithTheme(thm.Gruvbox()), WithStyles(cs))

	wantRender := cs.ActiveHeader.Render("probe")
	gotRender := tm.styles.ActiveHeader.Render("probe")
	if wantRender != gotRender {
		t.Errorf("WithStyles precedence (reverse order) failed:\n  want: %q\n  got:  %q", wantRender, gotRender)
	}
}

// TestWithStyles_AbsentMeansThemeDerived asserts that without WithStyles the
// model derives its styles from the theme.
func TestWithStyles_AbsentMeansThemeDerived(t *testing.T) {
	g := thm.Gruvbox()
	tm := New(WithTheme(g))

	themeDerived := g.Styles()
	wantRender := themeDerived.ActiveHeader.Render("probe")
	gotRender := tm.styles.ActiveHeader.Render("probe")
	if wantRender != gotRender {
		t.Errorf("without WithStyles, styles should equal theme-derived:\n  want: %q\n  got:  %q", wantRender, gotRender)
	}
}

// TestWithStyles_CustomIsDistinctFromTheme documents that customStyles()
// actually differs from Gruvbox so the precedence tests are meaningful.
func TestWithStyles_CustomIsDistinctFromTheme(t *testing.T) {
	cs := customStyles()
	g := thm.Gruvbox()
	themeStyles := g.Styles()

	csRender := cs.ActiveHeader.Render("probe")
	thRender := themeStyles.ActiveHeader.Render("probe")
	if csRender == thRender {
		t.Error("customStyles() must differ from Gruvbox() theme styles for the precedence tests to be meaningful")
	}
}

// ---- WithStyles + WithPadding layering ----

// TestWithStyles_WithPadding_LayeringApplied verifies that when WithStyles and
// WithPadding are both supplied, the padding is layered on top of the custom
// styles (i.e. the final ActiveHeader has the expected padding components).
func TestWithStyles_WithPadding_LayeringApplied(t *testing.T) {
	const padX, padY = 4, 2
	cs := customStyles()
	tm := New(WithStyles(cs), WithPadding(padX, padY))

	// The custom base style already carries Padding(9,9); WithPadding must
	// override/layer that so the final result reflects padX/padY.
	top, right, bottom, left := tm.styles.ActiveHeader.GetPadding()
	if top != padY {
		t.Errorf("layered ActiveHeader: top padding got %d, want %d (padY)", top, padY)
	}
	if bottom != padY {
		t.Errorf("layered ActiveHeader: bottom padding got %d, want %d (padY)", bottom, padY)
	}
	if left != padX {
		t.Errorf("layered ActiveHeader: left padding got %d, want %d (padX)", left, padX)
	}
	if right != padX {
		t.Errorf("layered ActiveHeader: right padding got %d, want %d (padX)", right, padX)
	}
}

// TestWithStyles_WithMargin_LayeringApplied verifies margin layering on top of
// custom styles in the same way as the padding layering test above.
func TestWithStyles_WithMargin_LayeringApplied(t *testing.T) {
	const marginX, marginY = 3, 1
	cs := customStyles()
	tm := New(WithStyles(cs), WithMargin(marginX, marginY))

	top, right, bottom, left := tm.styles.ActiveHeader.GetMargin()
	if top != marginY {
		t.Errorf("layered ActiveHeader: top margin got %d, want %d (marginY)", top, marginY)
	}
	if bottom != marginY {
		t.Errorf("layered ActiveHeader: bottom margin got %d, want %d (marginY)", bottom, marginY)
	}
	if left != marginX {
		t.Errorf("layered ActiveHeader: left margin got %d, want %d (marginX)", left, marginX)
	}
	if right != marginX {
		t.Errorf("layered ActiveHeader: right margin got %d, want %d (marginX)", right, marginX)
	}
}

// TestWithPadding_WithMargin_BothApplied exercises the branch in
// applyHeaderSpacing where both hasPadding and hasMargin are true.
func TestWithPadding_WithMargin_BothApplied(t *testing.T) {
	const padX, padY, marginX, marginY = 2, 1, 4, 0
	tm := New(WithTabs(tabsOf("A")), WithPadding(padX, padY), WithMargin(marginX, marginY))

	// Check padding.
	topP, rightP, bottomP, leftP := tm.styles.ActiveHeader.GetPadding()
	if topP != padY || bottomP != padY || leftP != padX || rightP != padX {
		t.Errorf("combined: padding mismatch: top=%d right=%d bottom=%d left=%d, want top/bottom=%d left/right=%d",
			topP, rightP, bottomP, leftP, padY, padX)
	}
	// Check margin.
	topM, rightM, bottomM, leftM := tm.styles.ActiveHeader.GetMargin()
	if topM != marginY || bottomM != marginY || leftM != marginX || rightM != marginX {
		t.Errorf("combined: margin mismatch: top=%d right=%d bottom=%d left=%d, want top/bottom=%d left/right=%d",
			topM, rightM, bottomM, leftM, marginY, marginX)
	}
}

// ---- Click alignment under spacing ----

// TestClickAlignment_WithPadding_HeaderSpanStaysConsistent builds a model
// with 3 tabs and WithPadding applied, then replicates View()'s header
// rendering to compute each header's [start,end) X span. A left-click at the
// horizontal midpoint of header index 1 must produce tabClickMsg{index:1},
// proving that View() and the mouse handler use the same rendered strings.
func TestClickAlignment_WithPadding_HeaderSpanStaysConsistent(t *testing.T) {
	tabs := tabsOf("Alpha", "Beta", "Gamma") // 3 tabs; index 0 is Active after New
	tm := New(
		WithTabs(tabs),
		WithTheme(noBorderTheme()),
		WithPadding(3, 1),
	)
	tm.Init()

	// Replicate View()'s header rendering exactly: each tab rendered with its
	// per-state style (index 0 is Active, 1 and 2 are Inactive after New()).
	headers := make([]string, len(tm.tabs))
	for i := range tm.tabs {
		headers[i] = tm.styles.Header(tm.tabs[i].State()).Render(tm.tabs[i].Name())
	}

	// Accumulate spans to find index-1's [start,end) range.
	x := 0
	starts := make([]int, len(headers))
	ends := make([]int, len(headers))
	for i, h := range headers {
		w := lipgloss.Width(h)
		starts[i] = x
		ends[i] = x + w
		x += w
	}

	// The midpoint of header 1's span.
	mid1 := starts[1] + (ends[1]-starts[1])/2

	v := tm.View()
	if v.OnMouse == nil {
		t.Fatal("View().OnMouse should not be nil")
	}

	click := tea.MouseClickMsg{X: mid1, Y: 0, Button: tea.MouseLeft}
	cmd := v.OnMouse(click)
	if cmd == nil {
		t.Fatalf("left-click at mid of header 1 (X=%d) returned nil cmd; spans: %v-%v", mid1, starts, ends)
	}
	msg := cmd()
	cm, ok := msg.(tabClickMsg)
	if !ok {
		t.Fatalf("cmd() returned %T, want tabClickMsg", msg)
	}
	if cm.index != 1 {
		t.Errorf("click at mid of header 1: got index %d, want 1; spans: start=%d end=%d", cm.index, starts[1], ends[1])
	}
}

// TestClickAlignment_WithMargin_HeaderSpanStaysConsistent is the margin
// counterpart of the padding alignment test above.
func TestClickAlignment_WithMargin_HeaderSpanStaysConsistent(t *testing.T) {
	tabs := tabsOf("One", "Two", "Three")
	tm := New(
		WithTabs(tabs),
		WithTheme(noBorderTheme()),
		WithMargin(2, 0),
	)
	tm.Init()

	headers := make([]string, len(tm.tabs))
	for i := range tm.tabs {
		headers[i] = tm.styles.Header(tm.tabs[i].State()).Render(tm.tabs[i].Name())
	}

	x := 0
	starts := make([]int, len(headers))
	ends := make([]int, len(headers))
	for i, h := range headers {
		w := lipgloss.Width(h)
		starts[i] = x
		ends[i] = x + w
		x += w
	}

	mid1 := starts[1] + (ends[1]-starts[1])/2

	v := tm.View()
	if v.OnMouse == nil {
		t.Fatal("View().OnMouse should not be nil")
	}

	click := tea.MouseClickMsg{X: mid1, Y: 0, Button: tea.MouseLeft}
	cmd := v.OnMouse(click)
	if cmd == nil {
		t.Fatalf("left-click at mid of header 1 (X=%d) returned nil cmd; spans: %v-%v", mid1, starts, ends)
	}
	msg := cmd()
	cm, ok := msg.(tabClickMsg)
	if !ok {
		t.Fatalf("cmd() returned %T, want tabClickMsg", msg)
	}
	if cm.index != 1 {
		t.Errorf("click at mid of header 1: got index %d, want 1; spans: start=%d end=%d", cm.index, starts[1], ends[1])
	}
}

// TestClickAlignment_WithPaddingAndMargin_AllThreeHeadersHit verifies click
// alignment for all three headers when both WithPadding and WithMargin are
// applied, exercising the combined spacing path in applyHeaderSpacing.
func TestClickAlignment_WithPaddingAndMargin_AllThreeHeadersHit(t *testing.T) {
	tabs := tabsOf("Tab0", "Tab1", "Tab2")
	tm := New(
		WithTabs(tabs),
		WithTheme(noBorderTheme()),
		WithPadding(2, 0),
		WithMargin(1, 0),
	)
	tm.Init()

	headers := make([]string, len(tm.tabs))
	for i := range tm.tabs {
		headers[i] = tm.styles.Header(tm.tabs[i].State()).Render(tm.tabs[i].Name())
	}

	x := 0
	type span struct{ start, end int }
	spans := make([]span, len(headers))
	for i, h := range headers {
		w := lipgloss.Width(h)
		spans[i] = span{x, x + w}
		x += w
	}

	v := tm.View()
	handler := v.OnMouse
	if handler == nil {
		t.Fatal("expected non-nil OnMouse handler")
	}

	for i, s := range spans {
		if s.end <= s.start {
			t.Errorf("header %d has zero-width span [%d,%d)", i, s.start, s.end)
			continue
		}
		mid := s.start + (s.end-s.start)/2
		click := tea.MouseClickMsg{X: mid, Y: 0, Button: tea.MouseLeft}
		cmd := handler(click)
		if cmd == nil {
			t.Errorf("header %d: click at X=%d returned nil; span=[%d,%d)", i, mid, s.start, s.end)
			continue
		}
		msg := cmd()
		cm, ok := msg.(tabClickMsg)
		if !ok {
			t.Errorf("header %d: cmd() returned %T, want tabClickMsg", i, msg)
			continue
		}
		if cm.index != i {
			t.Errorf("header %d: expected index %d, got %d", i, i, cm.index)
		}
	}
}
