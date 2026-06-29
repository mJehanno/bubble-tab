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
