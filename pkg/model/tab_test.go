package model

// White-box tests for the Tab type. Same package gives access to unexported
// fields when needed, but we prefer to exercise the exported API surface.

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

// ---- minimal stub tea.Model ----

type stubModel struct {
	id int
}

func (s stubModel) Init() tea.Cmd                        { return nil }
func (s stubModel) Update(tea.Msg) (tea.Model, tea.Cmd)  { return s, nil }
func (s stubModel) View() tea.View                       { return tea.NewView("stub") }

// ---- NewTab defaults ----

func TestNewTab_Defaults(t *testing.T) {
	tab := NewTab()

	if tab.Name() != "" {
		t.Errorf("expected empty name, got %q", tab.Name())
	}
	if tab.State() != Inactive {
		t.Errorf("expected Inactive state, got %q", tab.State())
	}
	if !tab.HasPermission() {
		t.Error("expected hasPermission=true by default")
	}
	if tab.Body() != nil {
		t.Errorf("expected nil body, got %v", tab.Body())
	}
	if tab.Initialized() {
		t.Error("expected initialized=false by default")
	}
}

// ---- WithName ----

func TestWithName(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"ordinary", "Home"},
		{"empty string", ""},
		{"unicode", "タブ"},
		{"spaces", "My Tab"},
		{"control chars", "tab\t1"},
		{"very long", "This is a very long tab name that probably wraps in some UIs"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tab := NewTab(WithName(tt.input))
			if tab.Name() != tt.input {
				t.Errorf("got %q, want %q", tab.Name(), tt.input)
			}
		})
	}
}

// ---- WithState ----

func TestWithState(t *testing.T) {
	for _, state := range []TabState{Active, Inactive, Disabled} {
		t.Run(string(state), func(t *testing.T) {
			tab := NewTab(WithState(state))
			if tab.State() != state {
				t.Errorf("got %q, want %q", tab.State(), state)
			}
		})
	}
}

// ---- WithHasPermission ----

func TestWithHasPermission(t *testing.T) {
	t.Run("false", func(t *testing.T) {
		tab := NewTab(WithHasPermission(false))
		if tab.HasPermission() {
			t.Error("expected HasPermission=false")
		}
	})
	t.Run("true overrides default which is already true", func(t *testing.T) {
		tab := NewTab(WithHasPermission(true))
		if !tab.HasPermission() {
			t.Error("expected HasPermission=true")
		}
	})
}

// ---- WithBody ----

func TestWithBody(t *testing.T) {
	m := stubModel{id: 1}
	tab := NewTab(WithBody(m))
	if tab.Body() == nil {
		t.Fatal("expected non-nil body")
	}
	got, ok := tab.Body().(stubModel)
	if !ok || got.id != 1 {
		t.Errorf("body mismatch: got %v", tab.Body())
	}
}

// ---- WithBodyFunc ----

func TestWithBodyFunc_LazyConstruction(t *testing.T) {
	calls := 0
	factory := func() tea.Model {
		calls++
		return stubModel{id: 42}
	}

	tab := NewTab(WithBodyFunc(factory))

	// Factory must not be called during construction.
	if calls != 0 {
		t.Fatalf("factory called %d times before EnsureBody", calls)
	}
	if tab.Body() != nil {
		t.Error("expected nil body before EnsureBody")
	}

	// First EnsureBody: factory called once, body set.
	tab.EnsureBody()
	if calls != 1 {
		t.Fatalf("factory called %d times after first EnsureBody, want 1", calls)
	}
	if tab.Body() == nil {
		t.Fatal("expected non-nil body after EnsureBody")
	}

	// Subsequent EnsureBody calls: idempotent, factory NOT called again.
	tab.EnsureBody()
	tab.EnsureBody()
	if calls != 1 {
		t.Errorf("factory called %d times after repeated EnsureBody, want 1", calls)
	}
}

// ---- WithBody + WithBodyFunc together: eager body wins ----

func TestWithBody_AndWithBodyFunc_EagerWins(t *testing.T) {
	factoryCalls := 0
	factory := func() tea.Model {
		factoryCalls++
		return stubModel{id: 99}
	}
	eager := stubModel{id: 7}

	// The spec says: "if both given, eager content wins and factory is never called".
	// Options are applied in order; the last write to content wins.
	// WithBody sets content; WithBodyFunc sets factory (does not touch content).
	// EnsureBody only runs if content==nil, so the eager body prevents the factory.
	tab := NewTab(WithBody(eager), WithBodyFunc(factory))
	tab.EnsureBody()

	if factoryCalls != 0 {
		t.Errorf("factory called %d times, want 0 (eager body should prevent factory)", factoryCalls)
	}
	got, ok := tab.Body().(stubModel)
	if !ok || got.id != 7 {
		t.Errorf("body is %v, want stubModel{id:7}", tab.Body())
	}
}

// When WithBodyFunc comes first and WithBody comes second the eager body still wins.
func TestWithBodyFunc_ThenWithBody_EagerWins(t *testing.T) {
	factoryCalls := 0
	factory := func() tea.Model {
		factoryCalls++
		return stubModel{id: 99}
	}
	eager := stubModel{id: 5}

	tab := NewTab(WithBodyFunc(factory), WithBody(eager))
	tab.EnsureBody()

	if factoryCalls != 0 {
		t.Errorf("factory called %d times, want 0", factoryCalls)
	}
	if got, ok := tab.Body().(stubModel); !ok || got.id != 5 {
		t.Errorf("body = %v, want stubModel{id:5}", tab.Body())
	}
}

// ---- EnsureBody: neither body nor factory ----

func TestEnsureBody_NeitherSet_StaysNil(t *testing.T) {
	tab := NewTab()
	tab.EnsureBody()
	if tab.Body() != nil {
		t.Errorf("expected body to remain nil, got %v", tab.Body())
	}
}

// ---- EnsureBody: already set body (no factory) ----

func TestEnsureBody_EagerBody_NotReplaced(t *testing.T) {
	m := stubModel{id: 3}
	tab := NewTab(WithBody(m))
	tab.EnsureBody()

	got, ok := tab.Body().(stubModel)
	if !ok || got.id != 3 {
		t.Errorf("body changed after EnsureBody: got %v", tab.Body())
	}
}

// ---- Option ordering: last WithName wins ----

func TestOptionOrder_LastWins(t *testing.T) {
	tab := NewTab(WithName("first"), WithName("second"))
	if tab.Name() != "second" {
		t.Errorf("got %q, want 'second'", tab.Name())
	}
}

// ---- SetState ----

func TestSetState(t *testing.T) {
	tab := NewTab()
	tab.SetState(Active)
	if tab.State() != Active {
		t.Errorf("got %q, want Active", tab.State())
	}
	tab.SetState(Disabled)
	if tab.State() != Disabled {
		t.Errorf("got %q, want Disabled", tab.State())
	}
}

// ---- SetHasPermission ----

func TestSetHasPermission(t *testing.T) {
	tab := NewTab()
	tab.SetHasPermission(false)
	if tab.HasPermission() {
		t.Error("expected HasPermission=false after SetHasPermission(false)")
	}
	tab.SetHasPermission(true)
	if !tab.HasPermission() {
		t.Error("expected HasPermission=true after SetHasPermission(true)")
	}
}

// ---- SetInitialized ----

func TestSetInitialized(t *testing.T) {
	tab := NewTab()
	if tab.Initialized() {
		t.Error("expected initialized=false")
	}
	tab.SetInitialized(true)
	if !tab.Initialized() {
		t.Error("expected initialized=true after SetInitialized(true)")
	}
	tab.SetInitialized(false)
	if tab.Initialized() {
		t.Error("expected initialized=false after SetInitialized(false)")
	}
}

// ---- SetBody persistence ----

func TestSetBody_Persists(t *testing.T) {
	tab := NewTab()
	if tab.Body() != nil {
		t.Fatal("precondition: body should be nil")
	}

	m := stubModel{id: 10}
	tab.SetBody(m)
	got, ok := tab.Body().(stubModel)
	if !ok || got.id != 10 {
		t.Errorf("body after SetBody: got %v, want stubModel{id:10}", tab.Body())
	}

	// SetBody replaces with a different model.
	m2 := stubModel{id: 20}
	tab.SetBody(m2)
	got2, ok := tab.Body().(stubModel)
	if !ok || got2.id != 20 {
		t.Errorf("body after second SetBody: got %v, want stubModel{id:20}", tab.Body())
	}
}

// ---- Initialized flag transitions ----

func TestInitialized_Transitions(t *testing.T) {
	tab := NewTab()
	// false -> true
	tab.SetInitialized(true)
	if !tab.Initialized() {
		t.Error("should be true")
	}
	// true -> true (idempotent)
	tab.SetInitialized(true)
	if !tab.Initialized() {
		t.Error("should still be true")
	}
	// true -> false
	tab.SetInitialized(false)
	if tab.Initialized() {
		t.Error("should be false after reset")
	}
}

// ---- Constants ----

func TestTabStateConstants(t *testing.T) {
	if Active != "active" {
		t.Errorf("Active = %q, want 'active'", Active)
	}
	if Inactive != "inactive" {
		t.Errorf("Inactive = %q, want 'inactive'", Inactive)
	}
	if Disabled != "disabled" {
		t.Errorf("Disabled = %q, want 'disabled'", Disabled)
	}
}
